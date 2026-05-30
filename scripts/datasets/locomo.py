#!/usr/bin/env python3
"""Download and convert pinned official LoCoMo into Goncho LOCOMO JSONL."""

import argparse
import json
import re
import urllib.request
from pathlib import Path
from typing import Any

from shared.checksums import sha256

SOURCE_REPO = "snap-research/locomo"
SOURCE_URL = "https://github.com/snap-research/locomo"
SOURCE_REVISION = "3eb6f2c585f5e1699204e3c3bdf7adc5c28cb376"
SOURCE_FILE = "data/locomo10.json"
SOURCE_LICENSE_FILE = "LICENSE.txt"
SOURCE_LICENSE = "Creative Commons Attribution-NonCommercial 4.0 International (CC BY-NC 4.0)"
SOURCE_SHA256 = "79fa87e90f04081343b8c8debecb80a9a6842b76a7aa537dc9fdf651ea698ff4"
LICENSE_SHA256 = "41003d4a74749c0220e33dd415042164b5a1093ed401f36277234f772d22d3d0"

CATEGORY_MAP = {
    1: "single_hop_retrieval",
    2: "temporal_retrieval",
    3: "multi_hop_retrieval",
    4: "open_domain_retrieval",
    5: "adversarial_unanswerable",
}


def download(url: str, out: Path, expected_sha: str) -> None:
    out.parent.mkdir(parents=True, exist_ok=True)
    if not out.exists():
        with urllib.request.urlopen(url, timeout=120) as response:
            out.write_bytes(response.read())
    got = sha256(out)
    if got != expected_sha:
        raise SystemExit(f"checksum mismatch for {out}: got {got}, want {expected_sha}")


def source_raw_url(path: str) -> str:
    return f"https://raw.githubusercontent.com/{SOURCE_REPO}/{SOURCE_REVISION}/{path}"


def session_date(conversation: dict[str, Any], session_no: str) -> str:
    return str(conversation.get(f"session_{session_no}_date_time") or conversation.get(f"session_{session_no}_date") or "")


def evidence_to_memory_id(sample_id: str, evidence: str) -> str:
    safe = evidence.replace(":", "-")
    return f"locomo-{sample_id}-{safe}"


def normalize_evidence_items(raw_items: list[Any]) -> list[str]:
    out: list[str] = []
    seen: set[str] = set()
    for item in raw_items:
        text = str(item)
        for match in re.finditer(r"D\s*(\d+)\s*[:\-]\s*(\d+)", text):
            evidence = f"D{int(match.group(1))}:{int(match.group(2))}"
            if evidence not in seen:
                seen.add(evidence)
                out.append(evidence)
    return out


def qa_category(raw: Any) -> str:
    try:
        return CATEGORY_MAP.get(int(raw), f"category_{raw}")
    except Exception:
        return f"category_{raw}"


def iter_dialog_turns(sample: dict[str, Any]):
    sample_id = str(sample.get("sample_id", "sample"))
    conversation = sample.get("conversation") or {}
    for key in sorted(conversation.keys(), key=session_sort_key):
        match = re.fullmatch(r"session_(\d+)", key)
        if not match:
            continue
        session_no = match.group(1)
        session_id = f"session-{int(session_no):02d}"
        timestamp = session_date(conversation, session_no)
        turns = conversation.get(key) or []
        for fallback_index, turn in enumerate(turns, start=1):
            dia_id = str(turn.get("dia_id") or f"D{session_no}:{fallback_index}")
            turn_index = parse_turn_index(dia_id, fallback_index)
            speaker = str(turn.get("speaker") or "unknown")
            content = str(turn.get("text") or "").strip()
            if not content:
                continue
            yield {
                "memory_id": evidence_to_memory_id(sample_id, dia_id),
                "conversation_id": f"locomo-{sample_id}",
                "session_id": session_id,
                "speaker": speaker,
                "turn_index": turn_index,
                "timestamp": timestamp,
                "content": content,
                "metadata": {
                    "source": "locomo10",
                    "dia_id": dia_id,
                    "sample_id": sample_id,
                    "has_image": bool(turn.get("img_url")),
                },
            }


def session_sort_key(key: str):
    match = re.fullmatch(r"session_(\d+)", key)
    if match:
        return (0, int(match.group(1)))
    return (1, key)


def parse_turn_index(dia_id: str, fallback: int) -> int:
    try:
        return int(dia_id.split(":", 1)[1])
    except Exception:
        return fallback


def convert(raw_path: Path, memories_out: Path, questions_out: Path, metadata_out: Path) -> dict[str, Any]:
    data = json.loads(raw_path.read_text())
    memories_out.parent.mkdir(parents=True, exist_ok=True)
    questions_out.parent.mkdir(parents=True, exist_ok=True)
    memory_ids: set[str] = set()
    memory_count = 0
    question_count = 0
    missing_evidence: list[str] = []

    with memories_out.open("w", encoding="utf-8") as mf:
        for sample in data:
            for row in iter_dialog_turns(sample):
                memory_ids.add(row["memory_id"])
                mf.write(json.dumps(row, ensure_ascii=False) + "\n")
                memory_count += 1

    with questions_out.open("w", encoding="utf-8") as qf:
        for sample in data:
            sample_id = str(sample.get("sample_id", "sample"))
            for idx, qa in enumerate(sample.get("qa") or [], start=1):
                evidence = normalize_evidence_items(qa.get("evidence") or [])
                candidate_gold = [evidence_to_memory_id(sample_id, item) for item in evidence]
                gold = []
                for mid in candidate_gold:
                    if mid not in memory_ids:
                        missing_evidence.append(mid)
                    else:
                        gold.append(mid)
                if not gold:
                    continue
                row = {
                    "question_id": f"locomo-{sample_id}-q-{idx:03d}",
                    "conversation_id": f"locomo-{sample_id}",
                    "question": str(qa.get("question") or "").strip(),
                    "gold_memory_ids": gold,
                    "category": qa_category(qa.get("category")),
                    "answer_hint": str(qa.get("answer") or ""),
                    "metadata": {
                        "source": "locomo10",
                        "sample_id": sample_id,
                        "category_id": qa.get("category"),
                        "evidence": evidence,
                    },
                }
                if not row["question"] or not row["gold_memory_ids"]:
                    continue
                qf.write(json.dumps(row, ensure_ascii=False) + "\n")
                question_count += 1

    meta = {
        "source_repo": SOURCE_REPO,
        "source_url": SOURCE_URL,
        "source_revision": SOURCE_REVISION,
        "source_file": SOURCE_FILE,
        "source_sha256": SOURCE_SHA256,
        "license": SOURCE_LICENSE,
        "license_file": SOURCE_LICENSE_FILE,
        "license_sha256": LICENSE_SHA256,
        "memories": str(memories_out),
        "questions": str(questions_out),
        "memory_count": memory_count,
        "question_count": question_count,
        "missing_evidence_count": len(missing_evidence),
        "missing_evidence_examples": missing_evidence[:20],
        "converted_memories_sha256": sha256(memories_out),
        "converted_questions_sha256": sha256(questions_out),
    }
    metadata_out.write_text(json.dumps(meta, indent=2) + "\n")
    return meta


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--raw-dir", default="data/locomo/raw")
    parser.add_argument("--out-dir", default="data/locomo")
    args = parser.parse_args()

    raw_dir = Path(args.raw_dir)
    out_dir = Path(args.out_dir)
    raw = raw_dir / "locomo10.json"
    license_path = raw_dir / "LICENSE.txt"
    download(source_raw_url(SOURCE_FILE), raw, SOURCE_SHA256)
    download(source_raw_url(SOURCE_LICENSE_FILE), license_path, LICENSE_SHA256)
    meta = convert(raw, out_dir / "memories.jsonl", out_dir / "questions.jsonl", out_dir / "metadata.json")
    print(json.dumps(meta, indent=2))


if __name__ == "__main__":
    main()
