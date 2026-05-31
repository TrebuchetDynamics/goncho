#!/usr/bin/env python3
"""Compatibility wrapper for scripts.benchmarks.agentmemory_locomo."""
from __future__ import annotations

try:
    from shared.compat import add_scripts_root
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.compat import add_scripts_root

add_scripts_root(__file__)
from benchmarks.agentmemory_locomo import *  # noqa: F401,F403,E402
from benchmarks.agentmemory_locomo import main  # noqa: E402

# External setup source: https://github.com/rohitg00/agentmemory

if __name__ == "__main__":
    raise SystemExit(main())
