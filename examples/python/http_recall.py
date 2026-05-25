#!/usr/bin/env python3
"""Minimal Python example for Goncho's local HTTP recall API.

Start Goncho first:

    goncho-server serve -db ./goncho.db -addr 127.0.0.1:8765

This example uses only Python's standard library and talks to the loopback
server. It does not configure hosts or mutate connector settings.
"""

from __future__ import annotations

import json
import urllib.request

BASE_URL = "http://127.0.0.1:8765"
WORKSPACE = "gormes"
PEER = "demo-operator"


def post_json(url: str, payload: dict) -> dict:
    body = json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(  # noqa: S310 local loopback Goncho server
        url,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=5) as response:  # noqa: S310 local loopback Goncho server
        return json.loads(response.read().decode("utf-8"))


def main() -> None:
    result = post_json(
        f"{BASE_URL}/v3/workspaces/{WORKSPACE}/peers/{PEER}/recall",
        {"query": "What should I verify before action?", "limit": 5},
    )
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
