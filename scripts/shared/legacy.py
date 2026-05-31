"""Compatibility facade for legacy script wrapper helpers."""
from __future__ import annotations

try:
    from shared.compatibility.wrappers.legacy import export_module, export_public
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.compatibility.wrappers.legacy import export_module, export_public

__all__ = ["export_module", "export_public"]
