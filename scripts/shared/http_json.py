"""Compatibility facade for shared HTTP JSON helpers."""
from __future__ import annotations

try:
    from shared.net.http_json import post_json_url, read_json_url
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.net.http_json import post_json_url, read_json_url

__all__ = ["post_json_url", "read_json_url"]
