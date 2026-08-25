#!/usr/bin/env python3
"""StackWatch static SPA server with same-origin API proxy.

The production host intentionally uses only the Python standard library. Static
files are served from web/public; /api/* and /health are proxied to the local
api-gateway so the browser uses one origin and no CORS configuration is needed.
"""
from __future__ import annotations

import argparse
import mimetypes
import os
import posixpath
import urllib.error
import urllib.request
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Iterable


class StackWatchHandler(SimpleHTTPRequestHandler):
    server_version = "StackWatchWeb/1.0"

    def __init__(self, *args, directory: str, api_base: str, **kwargs):
        self.api_base = api_base.rstrip("/")
        super().__init__(*args, directory=directory, **kwargs)

    def do_GET(self):  # noqa: N802
        if self.path.startswith("/api/") or self.path == "/health":
            self._proxy()
            return
        self._spa_or_static()

    def do_POST(self):  # noqa: N802
        self._proxy()

    def do_PUT(self):  # noqa: N802
        self._proxy()

    def do_PATCH(self):  # noqa: N802
        self._proxy()

    def do_DELETE(self):  # noqa: N802
        self._proxy()

    def _spa_or_static(self):
        path = self.path.split("?", 1)[0]
        candidate = Path(self.directory) / path.lstrip("/")
        try:
            candidate.resolve().relative_to(Path(self.directory).resolve())
        except ValueError:
            self.send_error(403, "forbidden")
            return
        if candidate.is_file():
            super().do_GET()
            return
        # Serve index.html for directory requests (e.g. /marketing/ → /marketing/index.html).
        # This preserves SPA fallback for app routes (no index.html → /index.html).
        if candidate.is_dir():
            index = candidate / "index.html"
            if index.is_file():
                # Rewrite self.path to the explicit index.html URL so simpleHTTP serves it
                self.path = "/" + str(index.relative_to(Path(self.directory))).replace(os.sep, "/")
                super().do_GET()
                return
        # Vite's BrowserRouter needs the shell on deep links.
        self.path = "/index.html"
        super().do_GET()

    def _proxy(self):
        target = self.api_base + self.path
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length) if length else None
        headers = {
            key: value
            for key, value in self.headers.items()
            if key.lower() not in {"host", "content-length", "connection"}
        }
        request = urllib.request.Request(target, data=body, headers=headers, method=self.command)
        try:
            with urllib.request.urlopen(request, timeout=60) as response:
                payload = response.read()
                self.send_response(response.status)
                self._copy_response_headers(response.headers.items(), len(payload))
                self.end_headers()
                self.wfile.write(payload)
        except urllib.error.HTTPError as exc:
            payload = exc.read()
            self.send_response(exc.code)
            self._copy_response_headers(exc.headers.items(), len(payload))
            self.end_headers()
            self.wfile.write(payload)
        except (urllib.error.URLError, TimeoutError) as exc:
            payload = ("{\"ok\":false,\"error\":\"api gateway unavailable\",\"details\":\"%s\"}" % exc).encode()
            self.send_response(502)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)

    def _copy_response_headers(self, headers: Iterable[tuple[str, str]], length: int):
        hop_by_hop = {"connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade"}
        copied = set()
        for key, value in headers:
            if key.lower() in hop_by_hop or key.lower() == "content-length":
                continue
            self.send_header(key, value)
            copied.add(key.lower())
        if "content-type" not in copied:
            self.send_header("Content-Type", "application/octet-stream")
        self.send_header("Content-Length", str(length))

    def log_message(self, format: str, *args):
        # Keep access logs concise and systemd/journal friendly.
        print("%s - %s" % (self.address_string(), format % args), flush=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", default=str(Path(__file__).parent / "public"))
    parser.add_argument("--api", default="http://127.0.0.1:8080")
    parser.add_argument("--port", type=int, default=8090)
    args = parser.parse_args()
    root = str(Path(args.root).resolve())
    os.chdir(root)
    factory = lambda *a, **kw: StackWatchHandler(*a, directory=root, api_base=args.api, **kw)
    server = ThreadingHTTPServer(("0.0.0.0", args.port), factory)
    print(f"StackWatch web listening on :{args.port}, root={root}, api={args.api}", flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
