"""Shared LOCOMO benchmark adapter helpers."""
from __future__ import annotations

from collections import defaultdict
from typing import Any


def content_collision_report(memories: list[dict[str, Any]]) -> dict[str, Any]:
    by_key: dict[tuple[str, str], list[str]] = defaultdict(list)
    by_content: dict[str, list[str]] = defaultdict(list)
    for memory in memories:
        mid = str(memory.get("memory_id", ""))
        content = str(memory.get("content", ""))
        conv = str(memory.get("conversation_id", ""))
        by_key[(conv, content)].append(mid)
        by_content[content].append(mid)
    duplicate_content = {key: value for key, value in by_content.items() if len(value) > 1}
    duplicate_composite = {f"{key[0]}\u241f{key[1]}": value for key, value in by_key.items() if len(value) > 1}
    return {
        "duplicate_content_count": len(duplicate_content),
        "duplicate_conversation_content_count": len(duplicate_composite),
        "collision_safe_content_only": len(duplicate_content) == 0,
        "collision_safe_conversation_content": len(duplicate_composite) == 0,
    }
