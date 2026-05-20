---
title: Retrieval Benchmarks
description: Run LongMemEval-style retrieval accuracy checks for Goncho.
---

Goncho includes a local benchmark runner for LongMemEval-style retrieval accuracy checks.

The runner itself does not download datasets. It consumes a local JSONL file, loads memories, runs a selected retrieval system, performs leakage checks, writes optional failure audits, and reports:

- `recall_at_5`
- `recall_at_10`
- `recall_any_at_5`
- `recall_any_at_10`
- `mrr`
- per-question retrieved IDs and first relevant rank
- leakage counts for exact query text and gold IDs in indexed memory

## Scientific Validation Targets

Use the smoke target in normal CI and the full target manually from a clean checkout.

```sh
make bench-longmemeval-s-smoke
```

The smoke target runs the tiny pinned fixture across deterministic baselines:

- `random`
- `bm25`
- `sqlite-fts5`
- `goncho-no-rank`
- `goncho`

For the full pinned LongMemEval-S run:

```sh
make bench-longmemeval-s
```

The full target downloads `xiaowu0162/longmemeval-cleaned` at revision `98d7416c24c778c2fee6e6f3006e7a073259d48f`, verifies SHA256 `d6f21ea9d60a0d56f34a05b609c79c88a451d2ae03597821ea3d5a9678c3a442`, converts the dataset, and writes JSON reports plus failure audits.

## Direct Command

```sh
go run ./cmd/goncho-bench \
  --dataset ./cmd/goncho-bench/testdata/tiny-longmemeval.jsonl \
  --out ./artifacts/tiny-longmemeval-report.json \
  --db ./artifacts/tiny-longmemeval.db \
  --limit 10 \
  --runs 20
```

For the full Go gate:

```sh
go test ./cmd/goncho-bench
go test ./...
```

## JSONL Format

Each line is one JSON object.

```json
{"type":"meta","dataset":"tiny-longmemeval"}
{"type":"memory","id":"mem-auth-owner","peer":"eval-user","session_key":"eval-session","content":"Alice owns the authentication service and reviews JWT middleware changes."}
{"type":"question","id":"q-auth-owner","peer":"eval-user","session_key":"eval-session","query":"Who reviews JWT middleware authentication changes?","relevant_ids":["mem-auth-owner"]}
```

### Memory record

| Field | Required | Meaning |
| --- | --- | --- |
| `type` | yes | Must be `memory`. |
| `id` | yes | Gold memory identifier used for scoring. |
| `peer` | no | Goncho peer id. Defaults to `benchmark-peer`. |
| `session_key` | no | Goncho session key. |
| `content` | yes | Memory text loaded into Goncho. |

### Question record

| Field | Required | Meaning |
| --- | --- | --- |
| `type` | yes | Must be `question`. |
| `id` | yes | Question identifier. |
| `peer` | no | Goncho peer id. Defaults to `benchmark-peer`. |
| `session_key` | no | Goncho session key. |
| `query` | yes | Search query sent to Goncho. |
| `relevant_ids` | yes | Gold memory ids accepted as correct evidence. |

## Report Shape

```json
{
  "system": "goncho",
  "dataset": "tiny-longmemeval",
  "memory_count": 3,
  "question_count": 3,
  "runs": 20,
  "recall_at_5": 1,
  "recall_at_10": 1,
  "recall_any_at_5": 1,
  "recall_any_at_10": 1,
  "mrr": 1,
  "questions": [
    {
      "id": "q-auth-owner",
      "relevant_ids": ["mem-auth-owner"],
      "retrieved_ids": ["mem-auth-owner", "mem-db-owner", "mem-dead-end"],
      "rank": 1
    }
  ]
}
```

The tiny fixture reports the real rank order produced by Goncho's current search path. It now reaches `R@5=1`, `R@10=1`, and `MRR=1` because lexical conclusion ranking puts each tiny gold memory at rank 1. Do not treat the tiny fixture as a benchmark claim. It is a harness smoke test.

## LongMemEval-S Use

To evaluate LongMemEval-S:

1. Obtain the dataset through its official distribution path.
2. Convert it to the JSONL format above.
3. Keep raw benchmark data out of the repository if licensing requires it.
4. Run `cmd/goncho-bench` against the converted local file.
5. Report real `R@5`, `R@10`, and `MRR`; do not copy numbers from other systems.

Example command shape:

```sh
go run ./cmd/goncho-bench \
  --dataset ./benchmarks/longmemeval-s.jsonl \
  --out ./artifacts/longmemeval-s-goncho-report.json \
  --db ./artifacts/longmemeval-s-goncho.db \
  --limit 10 \
  --runs 20
```

## Interpreting Results

| Metric | Meaning |
| --- | --- |
| `recall_at_5` | Fraction of all gold relevant memory IDs retrieved in the top 5. |
| `recall_at_10` | Fraction of all gold relevant memory IDs retrieved in the top 10. |
| `recall_any_at_5` | Whether any gold relevant memory appears in the top 5, averaged across questions. This matches the LongMemEval retrieval table methodology. |
| `recall_any_at_10` | Whether any gold relevant memory appears in the top 10, averaged across questions. This matches the LongMemEval retrieval table methodology. |
| `mrr` | Mean reciprocal rank of the first relevant memory. |

Use this runner to compare Goncho versions, scoring changes, and dataset conversions. Goncho v0.1.x does not claim LongMemEval-S leaderboard performance until the real dataset is run and the report is published.
