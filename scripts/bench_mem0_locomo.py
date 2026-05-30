#!/usr/bin/env python3
"""Compatibility wrapper for scripts.benchmarks.mem0_locomo."""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from benchmarks.mem0_locomo import *  # noqa: F401,F403,E402
from benchmarks.mem0_locomo import main  # noqa: E402

if __name__ == "__main__":
    raise SystemExit(main())
