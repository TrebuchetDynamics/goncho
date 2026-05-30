#!/usr/bin/env python3
"""Compatibility wrapper for scripts.benchmarks.agentmemory_locomo."""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from benchmarks.agentmemory_locomo import *  # noqa: F401,F403,E402
from benchmarks.agentmemory_locomo import main  # noqa: E402

# External setup source: https://github.com/rohitg00/agentmemory

if __name__ == "__main__":
    raise SystemExit(main())
