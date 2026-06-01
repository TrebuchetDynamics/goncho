"""Checksum helper contracts shared by data I/O tests."""
from __future__ import annotations

from support import import_attrs

sha256, = import_attrs(("shared.checksums", "scripts.shared.checksums"), "sha256")

__all__ = ["sha256"]
