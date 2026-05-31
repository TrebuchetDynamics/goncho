#!/usr/bin/env python3
"""Compatibility wrapper for scripts.benchmarks.agentmemory_locomo."""
from __future__ import annotations

try:
    from shared.legacy import export_module
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.legacy import export_module

main = export_module("benchmarks.agentmemory_locomo", __file__, globals())

# External setup source: https://github.com/rohitg00/agentmemory

if __name__ == "__main__":
    raise SystemExit(main())
