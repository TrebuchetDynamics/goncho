#!/usr/bin/env python3
"""Compatibility wrapper for scripts.datasets.locomo_test."""
from __future__ import annotations

import unittest

try:
    from shared.legacy import export_public
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.legacy import export_public

export_public("datasets.locomo_test", __file__, globals())

if __name__ == "__main__":
    unittest.main(module="datasets.locomo_test")
