#!/usr/bin/env python3
"""Compatibility wrapper for scripts.datasets.locomo."""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from datasets.locomo import *  # noqa: F401,F403,E402
from datasets.locomo import main  # noqa: E402

if __name__ == "__main__":
    main()
