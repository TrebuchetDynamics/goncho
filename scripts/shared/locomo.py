"""Compatibility facade for shared LOCOMO helpers."""
from __future__ import annotations

try:
    from shared.benchmark.locomo import content_collision_report, duplicate_content_fixture_report
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.benchmark.locomo import content_collision_report, duplicate_content_fixture_report

__all__ = ["content_collision_report", "duplicate_content_fixture_report"]
