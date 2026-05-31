"""Compatibility facade for moved script path helpers."""
from __future__ import annotations

try:
    from shared.compatibility.runtime.path import add_scripts_root
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.compatibility.runtime.path import add_scripts_root

__all__ = ["add_scripts_root"]
