#!/usr/bin/env python3
"""LOCOMO agentmemory stable-ID adapter probe.

The Go benchmark harness owns scoring. This script only emits retrieval rows when
agentmemory can preserve caller-supplied LOCOMO memory_id values. Current
agentmemory public memory_save/REST surfaces generate internal mem_* IDs and do
not expose an external_id/metadata field in search results, so the adapter fails
closed as not comparable.
"""
from __future__ import annotations

import argparse
import importlib.util
import json
import platform
import sys
from collections import defaultdict
from pathlib import Path
from typing import Any

BACKEND = "agentmemory"
AGENTMEMORY_SOURCE_VERSION = "@agentmemory/agentmemory 0.9.20 (local source docs/opensource-memory-systems/agentmemory/package.json)"
REASON = (
    "not comparable: agentmemory memory_save/REST surfaces generate internal mem_* IDs "
    "and the public tool schema has no external_id or metadata passthrough that search returns; "
    "content-only matching is rejected because LOCOMO contains duplicate content"
)


def package_status() -> dict[str, Any]:
    return {
        "python_version": platform.python_version(),
        "package_importable": importlib.util.find_spec("agentmemory") is not None,
        "pinned_source": AGENTMEMORY_SOURCE_VERSION,
        "install_commands": [
            "npm install -g @agentmemory/agentmemory@0.9.20",
            "agentmemory",
        ],
    }


def capability() -> dict[str, Any]:
    return {
        "backend": BACKEND,
        "comparable": False,
        "reason": REASON,
        "package": package_status(),
        "id_strategy": "requires returned memory_id from metadata/external_id; not available in public schema",
        "required_contract": {
            "reset": "clear local benchmark state",
            "insert": "insert memory_id, content, metadata without rewriting memory_id",
            "search": "return ranked results containing the original memory_id and numeric score",
        },
    }


def load_jsonl(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def content_collision_report(memories: list[dict[str, Any]]) -> dict[str, Any]:
    by_key: dict[tuple[str, str], list[str]] = defaultdict(list)
    by_content: dict[str, list[str]] = defaultdict(list)
    for m in memories:
        mid = str(m.get("memory_id", ""))
        content = str(m.get("content", ""))
        conv = str(m.get("conversation_id", ""))
        by_key[(conv, content)].append(mid)
        by_content[content].append(mid)
    duplicate_content = {k: v for k, v in by_content.items() if len(v) > 1}
    duplicate_composite = {f"{k[0]}\u241f{k[1]}": v for k, v in by_key.items() if len(v) > 1}
    return {
        "duplicate_content_count": len(duplicate_content),
        "duplicate_conversation_content_count": len(duplicate_composite),
        "collision_safe_content_only": len(duplicate_content) == 0,
        "collision_safe_conversation_content": len(duplicate_composite) == 0,
    }


def write_not_comparable(out: Path | None, memories: Path | None = None, questions: Path | None = None) -> None:
    row = capability()
    if memories and memories.exists():
        row["collision_check"] = content_collision_report(load_jsonl(memories))
    if questions and questions.exists():
        row["question_count"] = len(load_jsonl(questions))
    line = json.dumps(row, sort_keys=True)
    if out:
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(line + "\n", encoding="utf-8")
    else:
        print(line)


def smoke() -> int:
    fixture = [
        {"memory_id": "m1", "conversation_id": "c1", "content": "duplicate text"},
        {"memory_id": "m2", "conversation_id": "c2", "content": "duplicate text"},
        {"memory_id": "m3", "conversation_id": "c1", "content": "unique text"},
    ]
    report = content_collision_report(fixture)
    ok = report["collision_safe_content_only"] is False and report["collision_safe_conversation_content"] is True
    print(json.dumps({"backend": BACKEND, "smoke": ok, "comparable": False, "reason": REASON, "collision_check": report, "package": package_status()}, indent=2, sort_keys=True))
    return 0 if ok else 1


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--capability", action="store_true")
    parser.add_argument("--smoke", action="store_true")
    parser.add_argument("--memories", type=Path)
    parser.add_argument("--questions", type=Path)
    parser.add_argument("--out", type=Path)
    args = parser.parse_args()
    if args.smoke:
        return smoke()
    if args.capability:
        print(json.dumps(capability(), indent=2, sort_keys=True))
        return 0
    write_not_comparable(args.out, args.memories, args.questions)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
