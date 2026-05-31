"""Shared health polling helpers for smoke tests."""
from __future__ import annotations

import time
from collections.abc import Callable

try:
    from shared.http_json import read_json_url
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.http_json import read_json_url


def wait_for_json_status_ok(
    url: str,
    *,
    timeout_seconds: float,
    request_timeout: float,
    interval_seconds: float,
    before_attempt: Callable[[], None] | None = None,
    failure_context: Callable[[], str] | None = None,
) -> dict:
    """Poll a JSON health endpoint until it reports ``{"status": "ok"}``."""
    deadline = time.monotonic() + timeout_seconds
    last_error: Exception | None = None
    while time.monotonic() < deadline:
        if before_attempt is not None:
            before_attempt()
        try:
            payload = read_json_url(url, timeout=request_timeout)
            if payload.get("status") == "ok":
                return payload
        except Exception as exc:  # noqa: BLE001 - retry transient startup failures
            last_error = exc
        time.sleep(interval_seconds)
    suffix = f"; {failure_context()}" if failure_context is not None else ""
    raise RuntimeError(f"health check {url} did not become ok: {last_error}{suffix}")
