#!/usr/bin/env python3

import argparse
import logging
import os
import signal
import sys
from typing import Any, Optional

from dev_browser_mcp.browser import BrowserManager
from dev_browser_mcp.stdio_server import serve_stdio


def main(argv: Optional[list[str]] = None) -> int:
    if argv is None:
        argv = sys.argv[1:]
    parser = argparse.ArgumentParser(prog="dev-browser-mcp-server", add_help=True)
    parser.add_argument("--profile", default=os.environ.get("DEV_BROWSER_PROFILE", "default"))
    parser.add_argument(
        "--headless",
        action="store_true",
        default=os.environ.get("HEADLESS", "").lower() in {"1", "true", "yes"},
        help="Run Chromium in headless mode (default: false unless HEADLESS env var is set).",
    )
    parser.add_argument("--log-level", default=os.environ.get("DEV_BROWSER_LOG_LEVEL", "INFO"))
    args = parser.parse_args(argv)

    logging.basicConfig(level=args.log_level.upper(), stream=sys.stderr, format="%(asctime)s %(levelname)s %(message)s")

    manager = BrowserManager(profile=args.profile, headless=bool(args.headless))

    def handle_sigterm(_signum: int, _frame: Any) -> None:
        manager.close()
        raise SystemExit(0)

    signal.signal(signal.SIGTERM, handle_sigterm)

    try:
        return serve_stdio(manager)
    finally:
        manager.close()


if __name__ == "__main__":
    raise SystemExit(main())
