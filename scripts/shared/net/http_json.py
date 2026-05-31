"""Small HTTP JSON helpers for local smoke scripts."""
from __future__ import annotations

import json
import urllib.error
import urllib.request
from typing import Any


def read_json_url(url: str, timeout: float = 5) -> dict[str, Any]:
    """Read a JSON object from a URL."""
    with urllib.request.urlopen(url, timeout=timeout) as response:  # noqa: S310 callers restrict URLs
        payload = json.loads(response.read().decode("utf-8"))
    if not isinstance(payload, dict):
        raise TypeError(f"expected JSON object from {url}, got {type(payload).__name__}")
    return payload


def post_json_url(url: str, body: dict[str, Any], timeout: float = 5) -> dict[str, Any]:
    """POST a JSON object and read a JSON-object response."""
    raw = json.dumps(body).encode("utf-8")
    request = urllib.request.Request(url, data=raw, headers={"Content-Type": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:  # noqa: S310 callers restrict URLs
            payload = json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"POST {url} failed with {exc.code}: {detail}") from exc
    if not isinstance(payload, dict):
        raise TypeError(f"expected JSON object from {url}, got {type(payload).__name__}")
    return payload
