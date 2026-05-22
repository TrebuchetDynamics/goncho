# LOCOMO External Backend Adapters

The LOCOMO backend harness keeps scoring centralized in Goncho's Go benchmark runner. External adapters may only supply retrieval results with stable memory IDs.

User/operator docs:

- Docs site reference: `docs-site/src/content/docs/reference/retrieval-benchmarks.md`
- Operator runbook: `docs-site/src/content/docs/operators/runbook.md`

## Contract

Adapter input:

- `data/locomo/memories.jsonl`
- `data/locomo/questions.jsonl`

Adapter output JSONL, one comparable row per question:

```json
{
  "backend": "mem0",
  "question_id": "locomo-conv-41-q-001",
  "comparable": true,
  "results": [
    {
      "memory_id": "locomo-conv-41-D1-1",
      "score": 0.123,
      "backend_raw_id": "backend-native-id",
      "metadata": { "memory_id": "locomo-conv-41-D1-1" }
    }
  ]
}
```

If the backend cannot preserve stable IDs, the adapter must fail closed:

```json
{
  "backend": "agentmemory",
  "comparable": false,
  "reason": "not comparable: stable memory IDs unavailable"
}
```

## Non-negotiable scoring rules

- Retrieval only.
- No LLM judge.
- No answer scoring.
- Same LOCOMO converted JSONL.
- Same gold IDs.
- Gold IDs must reference known `memory_id` values from the same conversation as the question.
- Gold IDs must be unique within each LOCOMO question.
- Unique `memory_id` and `question_id` values in the converted LOCOMO fixtures.
- Conversation-scoped backend comparison before stable-ID scoring.
- Stable-ID fan-out must not expand the requested top-K scoring window.
- Centralized Go scoring only.
- No Goncho ranking changes.
- No gold leakage.
- No content-only matching unless collision-safe.

LOCOMO contains duplicate and near-duplicate content, including repeated content across conversations. Stable IDs must come from returned metadata, an external ID field, or another verified collision-safe key.

## Current adapter status

| Backend | Version / source inspected | Comparable | ID strategy | Reason |
| --- | --- | --- | --- | --- |
| Goncho | local Go module | yes | Native `memory_id` mapping in harness | Local deterministic adapter. |
| Goncho no-rank | local Go harness | yes | Native LOCOMO `memory_id` | Local deterministic no-ranking baseline that uses recency order before current Goncho ranking. |
| BM25 | local Go harness | yes | Native LOCOMO `memory_id` | Local deterministic lexical baseline. |
| SQLite FTS5 | local Go SQLite FTS5 | yes | Native LOCOMO `memory_id` column | Local deterministic lexical baseline. |
| agentmemory | `@agentmemory/agentmemory 0.9.20`, PR #583 commit `9b18a80c9d2839b025279978d3f4b5e1f9bc6e74` | yes, standalone fallback | `memory_save.external_id` plus `metadata.memory_id` returned by `memory_smart_search` | Stable IDs work. LOCOMO full score is `0.0` for the standalone InMemoryKV fallback because it uses strict all-term substring matching; this is not the full running agentmemory server. |
| mem0 | Python `3.12.3`; package not installed locally | no | Not executed | `mem0`/`mem0ai` is not installed in this environment; no stable-ID run can be produced. |

## Setup commands

agentmemory candidate setup:

```bash
git clone --branch feature/stable-external-memory-ids https://github.com/XelHaku/agentmemory.git
cd agentmemory
git checkout 9b18a80c9d2839b025279978d3f4b5e1f9bc6e74
npm install --legacy-peer-deps
export AGENTMEMORY_SOURCE_DIR=$PWD
python3 /path/to/goncho/scripts/bench_agentmemory_locomo.py --capability
python3 /path/to/goncho/scripts/bench_agentmemory_locomo.py --smoke
cd /path/to/goncho
AGENTMEMORY_SOURCE_DIR=$AGENTMEMORY_SOURCE_DIR make bench-locomo-backends
```

mem0 candidate setup:

```bash
pip install mem0ai
# configure local vector store/embedder per upstream mem0 docs
python3 scripts/bench_mem0_locomo.py --capability
python3 scripts/bench_mem0_locomo.py --smoke
```

## Smoke fixtures

Both adapter scripts include a duplicate-content smoke check:

```bash
python3 scripts/bench_agentmemory_locomo.py --smoke
python3 scripts/bench_mem0_locomo.py --smoke
```

The smoke fixture includes the same content under different `memory_id` values. Content-only matching is therefore rejected; metadata/external-ID return is required.

## Harness integration

The Go harness can consume external adapter JSONL outputs:

```bash
go run ./cmd/goncho-bench \
  --locomo-memories ./data/locomo/memories.jsonl \
  --locomo-questions ./data/locomo/questions.jsonl \
  --locomo-agentmemory-results ./artifacts/locomo-backends/agentmemory.jsonl \
  --locomo-mem0-results ./artifacts/locomo-backends/mem0.jsonl \
  --locomo-backend-comparison-json-out ./docs/benchmarks/results/locomo-backend-comparison.json \
  --locomo-backend-comparison-failures-out ./docs/benchmarks/failures/locomo-backend-comparison.jsonl \
  --locomo-backend-comparison-md-out ./docs/benchmarks/locomo-backend-comparison.md
```

Make targets run the probes first, then central scoring:

```bash
make bench-locomo-backends-smoke
make bench-locomo-backends
```
