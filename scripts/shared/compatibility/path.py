"""Compatibility-wrapper helpers for moved script modules."""
from __future__ import annotations

import sys
from pathlib import Path


__all__ = ["add_scripts_root"]


def add_scripts_root(anchor: str) -> None:
    """Ensure subpackages under scripts/ are importable from legacy wrappers."""
    scripts_root = str(Path(anchor).resolve().parent)
    if scripts_root not in sys.path:
        sys.path.insert(0, scripts_root)
