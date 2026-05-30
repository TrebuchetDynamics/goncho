#!/usr/bin/env python3
"""Compatibility wrapper for scripts.datasets.locomo_test."""
from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from datasets.locomo_test import *  # noqa: F401,F403,E402

if __name__ == "__main__":
    unittest.main(module="datasets.locomo_test")
