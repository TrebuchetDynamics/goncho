"""Polling helpers for smoke tests."""
from __future__ import annotations

import json
import time
import urllib.error
import urllib.request
from collections.abc import Callable
from typing import Any


def wait_for_json_status_ok(
    url: str,
    *,
    timeout_seconds: float,
    request_timeout: float,
    interval_seconds: float,
    before_attempt: Callable[[], None] | None = None,
    failure_context: Callable[[], str] | None = None,
    failure_message: Callable[[BaseException | None, str], str] | None = None,
) -> dict[str, Any]:
    """Poll ``url`` until it returns JSON with ``status == 'ok'``."""
    deadline = time.monotonic() + timeout_seconds
    last_error: BaseException | None = None
    while time.monotonic() < deadline:
        if before_attempt is not None:
            before_attempt()
        try:
            with urllib.request.urlopen(url, timeout=request_timeout) as response:  # noqa: S310 - loopback smoke helper
                payload = json.loads(response.read().decode("utf-8"))
            if isinstance(payload, dict) and payload.get("status") == "ok":
                return payload
            last_error = RuntimeError(f"status not ok: {payload!r}")
        except (OSError, urllib.error.URLError, json.JSONDecodeError) as exc:
            last_error = exc
        time.sleep(interval_seconds)
    context = failure_context() if failure_context is not None else ""
    if failure_message is not None:
        raise TimeoutError(failure_message(last_error, context))
    raise TimeoutError(f"timed out waiting for {url}: {last_error}; {context}")
