#!/usr/bin/env python3
"""Compatibility wrapper for scripts.datasets.locomo_test."""
from __future__ import annotations

import unittest

try:
    from shared.compat import add_scripts_root
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.compat import add_scripts_root

add_scripts_root(__file__)
from datasets.locomo_test import *  # noqa: F401,F403,E402

if __name__ == "__main__":
    unittest.main(module="datasets.locomo_test")
