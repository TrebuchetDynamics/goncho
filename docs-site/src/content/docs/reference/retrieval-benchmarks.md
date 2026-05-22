---
title: Retrieval Benchmarks
description: Run deterministic LongMemEval and LOCOMO retrieval accuracy checks for Goncho.
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

Use `make ecosystem-smoke` when checking public module resolution, package docs, external importability, and the checkout-local benchmark CLI together.

For benchmark-only validation, use the install smoke to prove the benchmark CLI builds from the current checkout, the smoke target in normal CI, and the full target manually from a clean checkout.

```sh
make ecosystem-smoke
make install-smoke
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

Use this runner to compare Goncho versions, scoring changes, and dataset conversions.

## LOCOMO Candidate-Generation Milestone

LOCOMO exposed a candidate-generation weakness in Goncho. After widening lexical pre-rank candidates, BM25-win `missing_candidate` failures dropped from `164` to `2`, and Goncho now essentially matches BM25 on full LOCOMO retrieval while preserving LongMemEval-S performance.

| Metric | Before | After |
| --- | ---: | ---: |
| LOCOMO Goncho recall_any@5 | `0.5247` | `0.6014` |
| LOCOMO Goncho recall_any@10 | `0.5873` | `0.6791` |
| LOCOMO Goncho MRR | `0.4104` | `0.4690` |
| BM25-win `missing_candidate` failures | `164` | `2` |

This milestone used no LLM judge, no answer scoring, no benchmark-specific gold-ID hack, and no ranking change. LongMemEval-S remained stable at recall_any@5 `0.968`, recall_any@10 `0.980`, MRR `0.9135`.

## LOCOMO External Backend Comparison

Use these targets to compare Goncho against other retrieval backends under the same LOCOMO scoring harness:

```sh
make bench-locomo-backends-smoke
make bench-locomo-backends
```

The harness compares:

- Goncho
- BM25
- SQLite FTS5
- agentmemory
- mem0

The benchmark harness is more trusted than any backend. It keeps adapters isolated and scoring centralized so every backend uses the same JSONL, same gold IDs, same metrics, same leakage checks, and same failure taxonomy.

Outputs:

| File | Purpose |
| --- | --- |
| `docs/benchmarks/results/locomo-backend-comparison.json` | Machine-readable backend comparison report. |
| `docs/benchmarks/locomo-backend-comparison.md` | Human-readable backend comparison report. |
| `docs/benchmarks/failures/locomo-backend-comparison.jsonl` | Failure audit and not-comparable evidence. |
| `docs/benchmarks/external-backend-adapters.md` | Adapter contract, setup notes, and current comparability status. |

External adapters must return stable inserted `memory_id` values. Content-only matching is not accepted because LOCOMO contains duplicate and near-duplicate memories.

Current status:

| Backend | Comparable | Notes |
| --- | --- | --- |
| Goncho | yes | Local deterministic adapter. |
| BM25 | yes | Local lexical baseline. |
| SQLite FTS5 | yes | Local SQLite FTS baseline. |
| agentmemory | yes, PR standalone fallback | PR #583 commit `9b18a80c9d2839b025279978d3f4b5e1f9bc6e74` preserves stable IDs through `external_id`/metadata. LOCOMO full scored `0.0000` in standalone InMemoryKV fallback mode; this is not the full running agentmemory server. |
| mem0 | no | `mem0`/`mem0ai` is not installed in this environment; no stable-ID run was produced. |

Probe commands:

```sh
AGENTMEMORY_SOURCE_DIR=/path/to/agentmemory-pr583 python3 scripts/bench_agentmemory_locomo.py --capability
AGENTMEMORY_SOURCE_DIR=/path/to/agentmemory-pr583 python3 scripts/bench_agentmemory_locomo.py --smoke
python3 scripts/bench_mem0_locomo.py --capability
python3 scripts/bench_mem0_locomo.py --smoke
```

If a backend cannot preserve stable IDs, keep it marked `not comparable` and document the exact reason. Do not score generated answers, use an LLM judge, or map results by content unless a collision audit proves the mapping is safe.

Next experiments are tracked in the [Benchmark Roadmap](/roadmap/benchmark-roadmap/). Preserve the frozen LOCOMO backend-comparison artifacts before adding contradiction/staleness audits, making more external backends comparable, or moving on to InfiniteBench, RULER, BABILong, BEIR, and real-world agent replay.
