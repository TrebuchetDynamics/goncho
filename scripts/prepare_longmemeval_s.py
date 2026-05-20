#!/usr/bin/env python3
"""Download and convert pinned LongMemEval-S into Goncho benchmark JSONL."""

import argparse
import hashlib
import json
from pathlib import Path

DATASET_REPO = "xiaowu0162/longmemeval-cleaned"
DATASET_REVISION = "98d7416c24c778c2fee6e6f3006e7a073259d48f"
DATASET_FILE = "longmemeval_s_cleaned.json"
DATASET_SHA256 = "d6f21ea9d60a0d56f34a05b609c79c88a451d2ae03597821ea3d5a9678c3a442"


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def download_raw(out_dir: Path) -> Path:
    from huggingface_hub import hf_hub_download

    out_dir.mkdir(parents=True, exist_ok=True)
    path = Path(
        hf_hub_download(
            repo_id=DATASET_REPO,
            filename=DATASET_FILE,
            repo_type="dataset",
            revision=DATASET_REVISION,
            local_dir=str(out_dir),
        )
    )
    got = sha256(path)
    if got != DATASET_SHA256:
        raise SystemExit(f"checksum mismatch for {path}: got {got}, want {DATASET_SHA256}")
    return path


def convert(raw_path: Path, out_path: Path) -> tuple[int, int]:
    data = json.loads(raw_path.read_text())
    out_path.parent.mkdir(parents=True, exist_ok=True)
    memory_count = 0
    with out_path.open("w", encoding="utf-8") as f:
        f.write(json.dumps({"type": "meta", "dataset": "longmemeval-s-cleaned"}) + "\n")
        for item in data:
            qid = item["question_id"]
            peer = "longmemeval:" + qid
            for sid, session in zip(item["haystack_session_ids"], item["haystack_sessions"]):
                parts = []
                for msg in session:
                    parts.append(f"{msg.get('role', '')}: {msg.get('content', '')}")
                f.write(
                    json.dumps(
                        {
                            "type": "memory",
                            "id": sid,
                            "peer": peer,
                            "content": "\n".join(parts),
                        },
                        ensure_ascii=False,
                    )
                    + "\n"
                )
                memory_count += 1
            f.write(
                json.dumps(
                    {
                        "type": "question",
                        "id": qid,
                        "peer": peer,
                        "query": item["question"],
                        "relevant_ids": item["answer_session_ids"],
                    },
                    ensure_ascii=False,
                )
                + "\n"
            )
    return len(data), memory_count


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--raw-dir", default="artifacts/longmemeval/raw")
    parser.add_argument("--out", default="artifacts/longmemeval/longmemeval-s-goncho.jsonl")
    args = parser.parse_args()

    raw = download_raw(Path(args.raw_dir))
    questions, memories = convert(raw, Path(args.out))
    print(json.dumps({
        "repo": DATASET_REPO,
        "revision": DATASET_REVISION,
        "sha256": DATASET_SHA256,
        "raw": str(raw),
        "converted": args.out,
        "questions": questions,
        "memories": memories,
    }, indent=2))


if __name__ == "__main__":
    main()
