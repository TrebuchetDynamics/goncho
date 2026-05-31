"""Shared sample payloads for JSONL round-trip behavior tests."""
from __future__ import annotations

from typing import Any

ROUNDTRIP_ROWS: list[dict[str, Any]] = [{"text": "café"}, {"n": 2}]

__all__ = ["ROUNDTRIP_ROWS"]
