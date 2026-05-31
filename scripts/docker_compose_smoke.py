#!/usr/bin/env python3
"""Compatibility wrapper for scripts.smoke.docker_compose."""
from __future__ import annotations

try:
    from shared.compat import add_scripts_root
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.compat import add_scripts_root

add_scripts_root(__file__)
from smoke.docker_compose import *  # noqa: F401,F403,E402
from smoke.docker_compose import main  # noqa: E402

if __name__ == "__main__":
    raise SystemExit(main())
