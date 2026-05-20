#!/usr/bin/env python3
"""Probe/placeholder for LOCOMO mem0 backend comparison.

The Go benchmark harness owns scoring. This script intentionally fails closed until
a mem0 integration can prove it preserves caller-supplied memory_id values through
retrieval results without answer-generation scoring.
"""
import argparse
import importlib.util
import json


def capability() -> dict:
    installed = importlib.util.find_spec("mem0") is not None
    return {
        "backend": "mem0",
        "installed": installed,
        "comparable": False,
        "reason": "not comparable: no stable-memory-id LOCOMO adapter is wired for mem0; scoring requires retrieval results to return the inserted memory_id exactly",
        "required_contract": {
            "reset": "clear local benchmark state",
            "insert": "insert memory_id, content, metadata without rewriting memory_id",
            "search": "return ranked results containing the original memory_id and numeric score",
        },
        "install_notes": [
            "Candidate package: pip install mem0ai, plus local vector-store dependencies required by upstream mem0 docs.",
            "Use retrieval APIs only; do not score generated answers.",
            "Wire this script only after stable memory IDs are exposed in search results.",
        ],
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--capability", action="store_true", help="emit adapter capability JSON")
    args = parser.parse_args()
    if args.capability:
        print(json.dumps(capability(), indent=2, sort_keys=True))
        return 0
    print(json.dumps(capability(), indent=2, sort_keys=True))
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
