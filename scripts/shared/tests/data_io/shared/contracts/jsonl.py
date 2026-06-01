"""JSONL helper contracts shared by data I/O tests."""
from __future__ import annotations

from support import import_attrs

load_jsonl, write_jsonl = import_attrs(
    ("shared.jsonl", "scripts.shared.jsonl"),
    "load_jsonl",
    "write_jsonl",
)

__all__ = ["load_jsonl", "write_jsonl"]
