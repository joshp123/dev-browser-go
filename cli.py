#!/usr/bin/env python3

import argparse
import json
import os
import sys
import urllib.parse
from typing import Any, Optional

from dev_browser_mcp.cli_support import (
    artifact_dir,
    daemon_base_url,
    ensure_page,
    http_json,
    is_daemon_healthy,
    parse_args_json,
    start_daemon,
    stop_daemon,
    write_output,
)
from dev_browser_mcp.runner import open_page, run_actions, run_call


def parse_crop_arg(raw: str | None) -> Optional[dict[str, int]]:
    if not raw:
        return None
    parts = [p.strip() for p in raw.split(",") if p.strip()]
    if len(parts) != 4:
        raise ValueError("--crop must be x,y,width,height")
    vals: list[int] = []
    for part in parts:
        if not part.isdigit():
            raise ValueError("--crop values must be integers")
        val = int(part)
        if val < 0:
            raise ValueError("--crop values must be non-negative")
        vals.append(val)
    x, y, w, h = vals
    if w < 1 or h < 1:
        raise ValueError("--crop width/height must be positive")
    max_wh = 2000
    return {"x": x, "y": y, "width": min(w, max_wh), "height": min(h, max_wh)}


def main(argv: Optional[list[str]] = None) -> int:
    if argv is None:
        argv = sys.argv[1:]
    # First pass: capture global flags no matter where they appear so
    # `dev-browser start --headless` works (argparse normally requires
    # globals before the subcommand).
    defaults_parser = argparse.ArgumentParser(add_help=False)
    defaults_parser.add_argument("--profile", default=os.environ.get("DEV_BROWSER_PROFILE", "default"))
    defaults_parser.add_argument("--headless", action="store_true", default=os.environ.get("HEADLESS", "").lower() in {"1", "true", "yes"})
    defaults_parser.add_argument("--output", choices=["summary", "json", "path"], default="summary")
    defaults_parser.add_argument("--out", default="", help="When --output=path, optional relative path under artifact dir.")
    defaults, remaining = defaults_parser.parse_known_args(argv)

    shared = argparse.ArgumentParser(add_help=False)
    shared.add_argument("--profile", default=argparse.SUPPRESS)
    shared.add_argument("--headless", action="store_true", default=argparse.SUPPRESS)
    shared.add_argument("--output", choices=["summary", "json", "path"], default=argparse.SUPPRESS)
    shared.add_argument("--out", default=argparse.SUPPRESS, help="When --output=path, optional relative path under artifact dir.")

    parser = argparse.ArgumentParser(prog="dev-browser", add_help=True, parents=[shared])
    sub = parser.add_subparsers(dest="cmd", required=True)

    sub.add_parser("status", help="Show daemon status for the profile.", parents=[shared])
    sub.add_parser("start", help="Start daemon (if not running).", parents=[shared])
    sub.add_parser("stop", help="Stop daemon (if running).", parents=[shared])
    sub.add_parser("list-pages", help="List named pages managed by the daemon.", parents=[shared])

    p_call = sub.add_parser("call", help="Call a tool by name with JSON args.", parents=[shared])
    p_call.add_argument("tool")
    p_call.add_argument("--args", default="{}")
    p_call.add_argument("--page", default="main")

    p_goto = sub.add_parser("goto", help="Navigate a page to a URL.", parents=[shared])
    p_goto.add_argument("url")
    p_goto.add_argument("--page", default="main")
    p_goto.add_argument("--wait-until", default="domcontentloaded")
    p_goto.add_argument("--timeout-ms", type=int, default=45000)

    p_snapshot = sub.add_parser("snapshot", help="Get a token-light snapshot (refs).", parents=[shared])
    p_snapshot.add_argument("--page", default="main")
    p_snapshot.add_argument("--engine", choices=["simple", "aria"], default="simple", help="Snapshot engine: simple is fastest; aria is more reliable on complex UIs.")
    p_snapshot.add_argument("--format", choices=["list", "tree"], default="list", help="Snapshot format: list is smallest; tree keeps UI structure (best with --engine aria).")
    p_snapshot.add_argument("--interactive-only", action=argparse.BooleanOptionalAction, default=True)
    p_snapshot.add_argument("--include-headings", action=argparse.BooleanOptionalAction, default=True)
    p_snapshot.add_argument("--max-items", type=int, default=80)
    p_snapshot.add_argument("--max-chars", type=int, default=8000)

    p_click = sub.add_parser("click-ref", help="Click a ref from the latest snapshot.", parents=[shared])
    p_click.add_argument("ref")
    p_click.add_argument("--page", default="main")
    p_click.add_argument("--timeout-ms", type=int, default=15000)

    p_fill = sub.add_parser("fill-ref", help="Fill a ref from the latest snapshot.", parents=[shared])
    p_fill.add_argument("ref")
    p_fill.add_argument("text")
    p_fill.add_argument("--page", default="main")
    p_fill.add_argument("--timeout-ms", type=int, default=15000)

    p_press = sub.add_parser("press", help="Send a key press to the page.", parents=[shared])
    p_press.add_argument("key")
    p_press.add_argument("--page", default="main")

    p_shot = sub.add_parser("screenshot", help="Save a screenshot and print path.", parents=[shared])
    p_shot.add_argument("--page", default="main")
    p_shot.add_argument("--path", default="")
    p_shot.add_argument("--full-page", action=argparse.BooleanOptionalAction, default=True)
    p_shot.add_argument("--annotate-refs", action=argparse.BooleanOptionalAction, default=False, help="Overlay [ref=eN] labels from the latest snapshot before capturing.")
    p_shot.add_argument("--crop", default="", help="Crop region x,y,width,height (clamped to 2000x2000).")

    p_html = sub.add_parser("save-html", help="Save page HTML and print path.", parents=[shared])
    p_html.add_argument("--page", default="main")
    p_html.add_argument("--path", default="")

    p_actions = sub.add_parser("actions", help="Run a batch of calls (JSON).", parents=[shared])
    p_actions.add_argument("--calls", default="", help="JSON array of {name, arguments}; if empty, read from stdin.")
    p_actions.add_argument("--page", default="main")

    p_wait = sub.add_parser("wait", help="Wait for a load state (non-throwing on timeout).", parents=[shared])
    p_wait.add_argument("--page", default="main")
    p_wait.add_argument("--strategy", default="playwright", choices=["playwright", "perf"], help="Wait strategy: playwright load states (fast) or perf (more robust on long-polling/analytics).")
    p_wait.add_argument("--state", default="load", choices=["load", "domcontentloaded", "networkidle", "commit"])
    p_wait.add_argument("--timeout-ms", type=int, default=10000)
    p_wait.add_argument("--min-wait-ms", type=int, default=0)

    p_close = sub.add_parser("close-page", help="Close a named page.", parents=[shared])
    p_close.add_argument("page")

    args = parser.parse_args(remaining, namespace=defaults)

    profile = args.profile
    headless = bool(args.headless)
    output_mode = args.output
    out_path = args.out or None

    if args.cmd == "status":
        if is_daemon_healthy(profile):
            sys.stdout.write(f"ok profile={profile} url={daemon_base_url(profile)}\n")
            return 0
        sys.stdout.write(f"not running profile={profile}\n")
        return 1

    if args.cmd == "start":
        start_daemon(profile, headless=headless)
        sys.stdout.write(f"started profile={profile} url={daemon_base_url(profile)}\n")
        return 0

    if args.cmd == "stop":
        stopped = stop_daemon(profile)
        sys.stdout.write(("stopped" if stopped else "not running") + f" profile={profile}\n")
        return 0 if stopped else 1

    if args.cmd == "list-pages":
        start_daemon(profile, headless=headless)
        base = daemon_base_url(profile)
        if not base:
            raise RuntimeError("Daemon state missing after start")
        pages = http_json("GET", f"{base}/pages", None, timeout_s=3.0)
        write_output(profile, output_mode, {"pages": pages.get("pages", [])}, out_path)
        return 0

    if args.cmd == "call":
        conn = ensure_page(profile, headless, args.page)
        with open_page(conn["wsEndpoint"], conn["targetId"]) as (_browser, _context, page):
            result = run_call(page, args.tool, parse_args_json(args.args), artifact_dir=artifact_dir(profile))
        write_output(profile, output_mode, result, out_path)
        return 0

    if args.cmd == "goto":
        conn = ensure_page(profile, headless, args.page)
        with open_page(conn["wsEndpoint"], conn["targetId"]) as (_browser, _context, page):
            result = run_call(page, "goto", {"url": args.url, "wait_until": args.wait_until, "timeout_ms": args.timeout_ms}, artifact_dir=artifact_dir(profile))
        write_output(profile, output_mode, result, out_path)
        return 0

    if args.cmd == "snapshot":
        conn = ensure_page(profile, headless, args.page)
        with open_page(conn["wsEndpoint"], conn["targetId"]) as (_browser, _context, page):
            result = run_call(
                page,
                "snapshot",
                {
                    "engine": str(args.engine),
                    "format": str(args.format),
                    "interactive_only": bool(args.interactive_only),
                    "include_headings": bool(args.include_headings),
                    "max_items": args.max_items,
                    "max_chars": args.max_chars,
                },
                artifact_dir=artifact_dir(profile),
            )
        write_output(profile, output_mode, result, out_path)
        return 0

    if args.cmd == "click-ref":
        conn = ensure_page(profile, headless, args.page)
        with open_page(conn["wsEndpoint"], conn["targetId"]) as (_browser, _context, page):
            result = run_call(page, "click_ref", {"ref": args.ref, "timeout_ms": args.timeout_ms}, artifact_dir=artifact_dir(profile))
        write_output(profile, output_mode, result, out_path)
        return 0

    if args.cmd == "fill-ref":
        conn = ensure_page(profile, headless, args.page)
        with open_page(conn["wsEndpoint"], conn["targetId"]) as (_browser, _context, page):
            result = run_call(page, "fill_ref", {"ref": args.ref, "text": args.text, "timeout_ms": args.timeout_ms}, artifact_dir=artifact_dir(profile))
        write_output(profile, output_mode, result, out_path)
        return 0

    if args.cmd == "press":
        conn = ensure_page(profile, headless, args.page)
        with open_page(conn["wsEndpoint"], conn["targetId"]) as (_browser, _context, page):
            result = run_call(page, "press", {"key": args.key}, artifact_dir=artifact_dir(profile))
        write_output(profile, output_mode, result, out_path)
        return 0

    if args.cmd == "screenshot":
        crop = parse_crop_arg(args.crop)
        payload: dict[str, Any] = {
            "path": args.path or None,
            "full_page": bool(args.full_page),
            "annotate_refs": bool(args.annotate_refs),
        }
        if crop is not None:
            payload["crop"] = crop
        conn = ensure_page(profile, headless, args.page)
        with open_page(conn["wsEndpoint"], conn["targetId"]) as (_browser, _context, page):
            result = run_call(
                page,
                "screenshot",
                payload,
                artifact_dir=artifact_dir(profile),
            )
        write_output(profile, output_mode, result, out_path)
        return 0

    if args.cmd == "save-html":
        conn = ensure_page(profile, headless, args.page)
        with open_page(conn["wsEndpoint"], conn["targetId"]) as (_browser, _context, page):
            result = run_call(page, "save_html", {"path": args.path or None}, artifact_dir=artifact_dir(profile))
        write_output(profile, output_mode, result, out_path)
        return 0

    if args.cmd == "actions":
        raw = args.calls or sys.stdin.read()
        try:
            calls = json.loads(raw)
        except Exception as exc:
            raise ValueError("Invalid JSON for --calls/stdin") from exc
        if not isinstance(calls, list):
            raise ValueError("--calls/stdin must be a JSON array")

        conn = ensure_page(profile, headless, args.page)
        with open_page(conn["wsEndpoint"], conn["targetId"]) as (_browser, _context, page):
            results = run_actions(page, calls, artifact_dir=artifact_dir(profile))
        last_snapshot = ""
        for item in results:
            if item.get("name") == "snapshot":
                res = item.get("result")
                if isinstance(res, dict) and isinstance(res.get("snapshot"), str):
                    last_snapshot = res["snapshot"]
        result = {"results": results}
        if last_snapshot:
            result["snapshot"] = last_snapshot
        write_output(profile, output_mode, result, out_path)
        return 0

    if args.cmd == "wait":
        conn = ensure_page(profile, headless, args.page)
        with open_page(conn["wsEndpoint"], conn["targetId"]) as (_browser, _context, page):
            result = run_call(
                page,
                "wait",
                {"strategy": args.strategy, "state": args.state, "timeout_ms": int(args.timeout_ms), "min_wait_ms": int(args.min_wait_ms)},
                artifact_dir=artifact_dir(profile),
            )
        write_output(profile, output_mode, result, out_path)
        return 0

    if args.cmd == "close-page":
        start_daemon(profile, headless=headless)
        base = daemon_base_url(profile)
        if not base:
            raise RuntimeError("Daemon state missing after start")
        encoded = urllib.parse.quote(args.page, safe="")
        data = http_json("DELETE", f"{base}/pages/{encoded}", None, timeout_s=5.0)
        if data.get("ok") is not True:
            raise RuntimeError(str(data.get("error", "close failed")))
        write_output(profile, output_mode, {"page": args.page, "closed": True}, out_path)
        return 0

    raise RuntimeError("unreachable")


if __name__ == "__main__":
    raise SystemExit(main())
