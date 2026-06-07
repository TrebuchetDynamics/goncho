#!/usr/bin/env python3
"""Compatibility wrapper for scripts.smoke.compose.docker_compose."""
from __future__ import annotations

try:
    from shared.legacy import export_module
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.legacy import export_module

main = export_module("smoke.compose.docker_compose", __file__, globals())

if __name__ == "__main__":
    raise SystemExit(main())
