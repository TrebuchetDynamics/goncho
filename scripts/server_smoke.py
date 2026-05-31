#!/usr/bin/env python3
"""Compatibility wrapper for scripts.smoke.server."""
from __future__ import annotations

import sys

try:
    from shared.compat import add_scripts_root
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.compat import add_scripts_root

add_scripts_root(__file__)
from smoke.server import *  # noqa: F401,F403,E402
from smoke.server import main  # noqa: E402

if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        print(f"server smoke failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
