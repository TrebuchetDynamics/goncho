#!/usr/bin/env python3
"""Compatibility wrapper for scripts.smoke.server."""
from __future__ import annotations

import sys

try:
    from shared.legacy import export_module
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.legacy import export_module

main = export_module("smoke.server", __file__, globals())

if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        print(f"server smoke failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
