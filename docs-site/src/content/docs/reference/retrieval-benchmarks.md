---
title: Retrieval Benchmarks
description: Run LongMemEval-style retrieval accuracy checks for Goncho.
---

Goncho includes a local benchmark runner for LongMemEval-style retrieval accuracy checks.

The runner does not download datasets. It consumes a local JSONL file, loads memories into a temporary Goncho SQLite store, runs Goncho search for each question, and reports:

- `recall_at_5`
- `recall_at_10`
- `mrr`
- per-question retrieved IDs and first relevant rank

## Command

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
| `recall_at_5` | Fraction of gold relevant memory IDs retrieved in the top 5. |
| `recall_at_10` | Fraction of gold relevant memory IDs retrieved in the top 10. |
| `mrr` | Mean reciprocal rank of the first relevant memory. |

Use this runner to compare Goncho versions, scoring changes, and dataset conversions. Goncho v0.1.x does not claim LongMemEval-S leaderboard performance until the real dataset is run and the report is published.
