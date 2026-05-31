#!/usr/bin/env python3
"""Compatibility wrapper for scripts.datasets.longmemeval_s."""
from __future__ import annotations

try:
    from shared.compat import add_scripts_root
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.compat import add_scripts_root

add_scripts_root(__file__)
from datasets.longmemeval_s import *  # noqa: F401,F403,E402
from datasets.longmemeval_s import main  # noqa: E402

if __name__ == "__main__":
    main()
