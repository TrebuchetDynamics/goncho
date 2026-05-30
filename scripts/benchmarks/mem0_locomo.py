#!/usr/bin/env python3
"""LOCOMO mem0 stable-ID adapter probe.

The Go benchmark harness owns scoring. This script emits comparable rows only if
a local mem0 installation can return caller-supplied LOCOMO memory_id values from
retrieval results. In this environment mem0 is not installed; the script fails
closed with a reproducible not-comparable reason.
"""
from __future__ import annotations

import argparse
import importlib.metadata
import importlib.util
import json
import platform
from pathlib import Path
from typing import Any

from shared.jsonl import load_jsonl
from shared.locomo import content_collision_report

BACKEND = "mem0"
REASON_PACKAGE_MISSING = "not comparable: Python package mem0/mem0ai is not installed in this environment"
REASON_STABLE_ID = (
    "not comparable: mem0 adapter has not proven stable caller-supplied memory_id round-trip; "
    "content-only matching is rejected because LOCOMO contains duplicate content"
)


def installed_version() -> str | None:
    for dist in ("mem0ai", "mem0"):
        try:
            return f"{dist} {importlib.metadata.version(dist)}"
        except importlib.metadata.PackageNotFoundError:
            continue
    return None


def package_status() -> dict[str, Any]:
    return {
        "python_version": platform.python_version(),
        "package_importable": importlib.util.find_spec("mem0") is not None,
        "installed_version": installed_version(),
        "install_commands": [
            "pip install mem0ai",
            "configure local vector store/embedder per upstream mem0 docs",
        ],
    }


def capability() -> dict[str, Any]:
    version = installed_version()
    reason = REASON_STABLE_ID if version else REASON_PACKAGE_MISSING
    return {
        "backend": BACKEND,
        "comparable": False,
        "reason": reason,
        "package": package_status(),
        "id_strategy": "metadata/external_id passthrough required; no successful local stable-ID run recorded",
        "required_contract": {
            "reset": "clear local benchmark state",
            "insert": "insert memory_id, content, metadata without rewriting memory_id",
            "search": "return ranked results containing the original memory_id and numeric score",
        },
    }


def extract_memory_id_from_hit(hit: dict[str, Any]) -> str:
    for key in ("memory_id", "external_id", "id"):
        value = hit.get(key)
        if isinstance(value, str) and value.startswith("locomo-"):
            return value
    metadata = hit.get("metadata") or {}
    if isinstance(metadata, dict):
        for key in ("memory_id", "external_id", "locomo_memory_id"):
            value = metadata.get(key)
            if isinstance(value, str):
                return value
    return ""


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
    metadata_hit = {"id": "backend-raw", "metadata": {"memory_id": "m1"}}
    ok = report["collision_safe_content_only"] is False and extract_memory_id_from_hit(metadata_hit) == "m1"
    print(json.dumps({"backend": BACKEND, "smoke": ok, "comparable": False, "reason": capability()["reason"], "collision_check": report, "package": package_status()}, indent=2, sort_keys=True))
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
