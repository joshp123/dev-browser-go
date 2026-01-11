#!/usr/bin/env python3

import argparse
import json
import logging
import os
import socket
import signal
import sys
import threading
import urllib.parse
from http.server import BaseHTTPRequestHandler, HTTPServer
from typing import Any, Optional

from dev_browser_mcp.host import BrowserHost


LOGGER = logging.getLogger("dev-browser-daemon")


def _state_file_path(host: BrowserHost) -> str:
    return str(host.user_data_dir.parent / "daemon.json")


def _write_state_file(path: str, data: dict[str, Any]) -> None:
    directory = os.path.dirname(path)
    if directory:
        os.makedirs(directory, exist_ok=True)
    tmp = path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
    os.replace(tmp, path)


def _remove_state_file(path: str) -> None:
    try:
        os.remove(path)
    except FileNotFoundError:
        pass


def _choose_free_port() -> int:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    try:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])
    finally:
        sock.close()


class Handler(BaseHTTPRequestHandler):
    server_version = "dev-browser-daemon/0.1.0"

    def do_GET(self) -> None:  # noqa: N802
        if self.path == "/health":
            self._send_json(
                200,
                {
                    "ok": True,
                    "pid": os.getpid(),
                    "profile": self.server.profile,
                    "version": "0.1.0",
                    "wsEndpoint": self.server.host.ws_endpoint,
                },
            )
            return
        if self.path == "/":
            self._send_json(200, {"wsEndpoint": self.server.host.ws_endpoint})
            return
        if self.path == "/pages":
            self._send_json(200, {"pages": self.server.host.list_pages()})
            return
        self._send_json(404, {"ok": False, "error": "not found"})

    def do_POST(self) -> None:  # noqa: N802
        if self.path == "/shutdown":
            self._send_json(200, {"ok": True})
            threading.Thread(target=self.server.shutdown, daemon=True).start()
            return

        if self.path != "/pages":
            self._send_json(404, {"ok": False, "error": "not found"})
            return

        body = self._read_json()
        if body is None:
            self._send_json(400, {"ok": False, "error": "invalid json"})
            return

        name = body.get("name")
        if not isinstance(name, str) or not name.strip():
            self._send_json(400, {"ok": False, "error": "name is required and must be a non-empty string"})
            return

        try:
            entry = self.server.host.get_or_create_page(name.strip())
        except Exception as exc:
            LOGGER.exception("page_failed name=%s", name)
            self._send_json(500, {"ok": False, "error": str(exc)})
            return

        self._send_json(200, {"wsEndpoint": self.server.host.ws_endpoint, "name": entry.name, "targetId": entry.target_id})

    def do_DELETE(self) -> None:  # noqa: N802
        if not self.path.startswith("/pages/"):
            self._send_json(404, {"ok": False, "error": "not found"})
            return
        name = urllib.parse.unquote(self.path[len("/pages/") :])
        if not name:
            self._send_json(400, {"ok": False, "error": "name required"})
            return
        try:
            closed = self.server.host.close_page(name)
        except Exception as exc:
            LOGGER.exception("close_failed name=%s", name)
            self._send_json(500, {"ok": False, "error": str(exc)})
            return
        if not closed:
            self._send_json(404, {"ok": False, "error": "page not found"})
            return
        self._send_json(200, {"ok": True})

    def log_message(self, _format: str, *_args: Any) -> None:  # noqa: D401, N802
        # Keep daemon quiet by default; logs go through LOGGER.
        return

    def _read_json(self) -> Optional[dict[str, Any]]:
        length = self.headers.get("Content-Length")
        if length is None:
            return None
        try:
            size = int(length)
        except ValueError:
            return None
        raw = self.rfile.read(size)
        try:
            data = json.loads(raw.decode("utf-8"))
        except Exception:
            return None
        return data if isinstance(data, dict) else None

    def _send_json(self, status: int, data: dict[str, Any]) -> None:
        payload = json.dumps(data, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)


class Server(HTTPServer):
    def __init__(self, host: str, port: int, browser_host: BrowserHost, profile: str):
        super().__init__((host, port), Handler)
        self.host = browser_host
        self.profile = profile


def main(argv: Optional[list[str]] = None) -> int:
    if argv is None:
        argv = sys.argv[1:]
    parser = argparse.ArgumentParser(prog="dev-browser-daemon", add_help=True)
    parser.add_argument("--profile", default=os.environ.get("DEV_BROWSER_PROFILE", "default"))
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=int(os.environ.get("DEV_BROWSER_PORT", "0")))
    parser.add_argument("--cdp-port", type=int, default=int(os.environ.get("DEV_BROWSER_CDP_PORT", "0")))
    parser.add_argument(
        "--headless",
        action="store_true",
        default=os.environ.get("HEADLESS", "").lower() in {"1", "true", "yes"},
    )
    parser.add_argument("--state-file", default=os.environ.get("DEV_BROWSER_STATE_FILE", ""), help="Override daemon state file path.")
    parser.add_argument("--log-level", default=os.environ.get("DEV_BROWSER_LOG_LEVEL", "INFO"))
    args = parser.parse_args(argv)

    logging.basicConfig(level=args.log_level.upper(), stream=sys.stderr, format="%(asctime)s %(levelname)s %(message)s")

    cdp_port = int(args.cdp_port)
    if cdp_port == 0:
        cdp_port = _choose_free_port()
    if cdp_port < 1 or cdp_port > 65535:
        raise ValueError("cdp-port must be between 1 and 65535")

    browser_host = BrowserHost(profile=args.profile, headless=bool(args.headless), cdp_port=cdp_port)
    browser_host.start()

    state_file = args.state_file or _state_file_path(browser_host)
    server = Server(args.host, int(args.port), browser_host, args.profile)
    _write_state_file(
        state_file,
        {
            "pid": os.getpid(),
            "host": args.host,
            "port": server.server_address[1],
            "profile": args.profile,
            "cdpPort": cdp_port,
            "wsEndpoint": browser_host.ws_endpoint,
        },
    )

    def handle_sigterm(_signum: int, _frame: Any) -> None:
        server.shutdown()

    signal.signal(signal.SIGTERM, handle_sigterm)

    try:
        server.serve_forever(poll_interval=0.2)
    finally:
        _remove_state_file(state_file)
        browser_host.stop()

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
