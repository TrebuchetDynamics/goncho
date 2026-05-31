"""Compatibility facade for shared JSONL helpers."""
from __future__ import annotations

try:
    from shared.io.jsonl import load_jsonl, write_jsonl
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.io.jsonl import load_jsonl, write_jsonl

__all__ = ["load_jsonl", "write_jsonl"]
