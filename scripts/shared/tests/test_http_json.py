#!/usr/bin/env python3
from __future__ import annotations

import json
import sys
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

try:
    from shared.http_json import post_json_url, read_json_url
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.http_json import post_json_url, read_json_url


class JsonHandler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:  # noqa: N802 - stdlib callback name
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps({"status": "ok"}).encode("utf-8"))

    def do_POST(self) -> None:  # noqa: N802 - stdlib callback name
        length = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(length).decode("utf-8"))
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps({"received": body}).encode("utf-8"))

    def log_message(self, format: str, *args: object) -> None:
        return


class HttpHelperTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), JsonHandler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.base_url = f"http://127.0.0.1:{cls.server.server_port}"

    @classmethod
    def tearDownClass(cls) -> None:
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join(timeout=5)

    def test_read_json_url_returns_object(self) -> None:
        self.assertEqual(read_json_url(f"{self.base_url}/health"), {"status": "ok"})

    def test_post_json_url_round_trips_object(self) -> None:
        self.assertEqual(post_json_url(f"{self.base_url}/echo", {"x": 1}), {"received": {"x": 1}})


if __name__ == "__main__":
    unittest.main()
