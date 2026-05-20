LONGMEMEVAL_REVISION := 98d7416c24c778c2fee6e6f3006e7a073259d48f
LONGMEMEVAL_SHA256 := d6f21ea9d60a0d56f34a05b609c79c88a451d2ae03597821ea3d5a9678c3a442
LONGMEMEVAL_DATE := $(shell date -u +%Y-%m-%d)
BENCH_SYSTEMS := random bm25 sqlite-fts5 goncho-no-rank goncho
LONGMEMEVAL_RUNS ?= 20

.PHONY: bench-longmemeval-s-smoke bench-longmemeval-s prepare-longmemeval-s

bench-longmemeval-s-smoke:
	@mkdir -p artifacts/bench-smoke docs/benchmarks/results docs/benchmarks/failures
	@for system in $(BENCH_SYSTEMS); do \
		go run ./cmd/goncho-bench \
			--dataset ./cmd/goncho-bench/testdata/tiny-longmemeval.jsonl \
			--out ./docs/benchmarks/results/longmemeval-s-smoke-$$system.json \
			--failures ./docs/benchmarks/failures/longmemeval-s-smoke-$$system.jsonl \
			--db ./artifacts/bench-smoke/$$system.db \
			--system $$system \
			--dataset-revision smoke-fixture \
			--dataset-sha256 smoke-fixture \
			--limit 10 \
			--runs 2; \
	done

prepare-longmemeval-s:
	python3 ./scripts/prepare_longmemeval_s.py \
		--raw-dir ./artifacts/longmemeval/raw \
		--out ./artifacts/longmemeval/longmemeval-s-goncho.jsonl

bench-longmemeval-s: prepare-longmemeval-s
	@mkdir -p artifacts/longmemeval docs/benchmarks/results docs/benchmarks/failures
	@for system in $(BENCH_SYSTEMS); do \
		go run ./cmd/goncho-bench \
			--dataset ./artifacts/longmemeval/longmemeval-s-goncho.jsonl \
			--out ./docs/benchmarks/results/longmemeval-s-$(LONGMEMEVAL_DATE)-$$system.json \
			--failures ./docs/benchmarks/failures/longmemeval-s-$(LONGMEMEVAL_DATE)-$$system.jsonl \
			--db ./artifacts/longmemeval/$$system.db \
			--system $$system \
			--dataset-revision $(LONGMEMEVAL_REVISION) \
			--dataset-sha256 $(LONGMEMEVAL_SHA256) \
			--limit 10 \
			--runs $(LONGMEMEVAL_RUNS); \
	done
