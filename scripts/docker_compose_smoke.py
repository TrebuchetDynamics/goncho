#!/usr/bin/env python3
"""Compatibility wrapper for scripts.smoke.docker_compose."""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from smoke.docker_compose import *  # noqa: F401,F403,E402
from smoke.docker_compose import main  # noqa: E402

if __name__ == "__main__":
    raise SystemExit(main())
