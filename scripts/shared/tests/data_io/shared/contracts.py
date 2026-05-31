"""Import contracts shared by data I/O helper tests."""
from __future__ import annotations

from support import import_attrs

sha256, = import_attrs(("shared.checksums", "scripts.shared.checksums"), "sha256")
load_jsonl, write_jsonl = import_attrs(
    ("shared.jsonl", "scripts.shared.jsonl"),
    "load_jsonl",
    "write_jsonl",
)

__all__ = ["load_jsonl", "sha256", "write_jsonl"]
