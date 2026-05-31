"""Compatibility facade for shared checksum helpers."""
from __future__ import annotations

try:
    from shared.io.checksums import sha256
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.io.checksums import sha256

__all__ = ["sha256"]
