# Goncho Benchmark Roadmap

Goncho is evaluated as a long-term memory retrieval system for agents, not as a generic vector store.

LongMemEval-S proved the first layer: deterministic retrieval sanity on long conversational haystacks. The next work should progressively test harder forms of memory: evolving facts, temporal state, scale, noise, standard IR credibility, and real agent utility.

## Benchmark progression

| Phase | Benchmark | Purpose | Why it matters for Goncho | Status |
| --- | --- | --- | --- | --- |
| 1 | LongMemEval-S | Long conversational retrieval | Proves ID-based recall over many sessions without LLM judgment. | First scientific pass done. |
| 2 | LOCOMO | Conversational long-term memory | Tests long conversations, evolving facts, temporal recall, contradictions, and relationship changes. | Candidate-generation milestone and stable-ID backend comparison frozen. |
| 3 | InfiniteBench / RULER | Scale and buried-fact stress | Tests whether retrieval survives very large memory, distractors, and long-context pressure. | Planned. |
| 4 | BABILong | Controlled synthetic reasoning | Tests temporal/entity tracking and consistency under known-answer synthetic tasks. | Planned. |
| 5 | BEIR | Standard IR credibility | Compares Goncho against classic retrieval systems beyond agent-memory-specific tasks. | Planned. |
| 6 | Real-world agent replay | Actual agent utility | Tests whether memory helps on real coding sessions, user preferences, rejected approaches, and repeated mistakes. | Planned; most important long term. |

## Phase 2: LOCOMO

LOCOMO is the best next benchmark because it is closer to real agent memory than plain retrieval.

Milestone: LOCOMO exposed a candidate-generation weakness in Goncho, not primarily a ranking-philosophy weakness. After widening lexical pre-rank candidates, Goncho recall_any@5 improved `0.5247 -> 0.6014`, recall_any@10 improved `0.5873 -> 0.6791`, MRR improved `0.4104 -> 0.4690`, and BM25-win `missing_candidate` failures dropped `164 -> 2`. LongMemEval-S stayed stable at recall_any@5 `0.968`, recall_any@10 `0.980`, MRR `0.9135`.

The next LOCOMO step should not be immediate tuning. The candidate-generation result and stable-ID backend comparison are frozen for historical comparability. Future LOCOMO work should preserve those artifacts while either making more external backends comparable or adding contradiction/staleness audits on top of the same harness.

LOCOMO improvement priorities:

- Use multi-hop graph expansion to connect entities, events, relationships, and evidence IDs that lexical matching alone cannot bridge.
- Add query decomposition so multi-part questions retrieve each required fact before final ranking.
- Add coverage-aware ranking so top results include complementary gold memories instead of near-duplicate hits.
- Improve temporal and speaker routing so changed facts, chronology, and who-said-what are ranked in the right conversation branch.
- Drive changes from failure-audit buckets such as missing candidates, rank-too-low candidates, wrong branch retrieval, and missing companion memories.
- Target: raise multi-hop recall_any@10 above `50%` and raise multi-hop strict_recall@10 above `25%` without answer hints, benchmark-specific hacks, or LLM judges.

LOCOMO implementation gate:

- Recommendations are not approval to change retrieval behavior.
- Write an approved design or plan before production retrieval changes.
- Start implementation with a focused failing recall test, for example `TestGraphRecallConnectsOwnerThroughServiceRelation`, before adding graph storage, relation extraction, or reranking code.
- Preserve frozen LOCOMO artifacts until a new date-stamped full run is intentionally generated.
- Do not tune against LOCOMO gold IDs, answer text, or benchmark-specific hacks; score only stable inserted memory IDs.

First graph-assisted implementation slice delivered: `TestGraphRecallConnectsOwnerThroughServiceRelation` proves graph-expanded multi-hop recall can retrieve a stable-ID companion memory with relation path provenance before any LOCOMO full-run artifact is regenerated.

Coverage-aware graph companion selection delivered: `TestRecallPipelineCoverageAwareSelectionKeepsGraphCompanion` proves selection prefers relation-path companion memories over near-duplicate lexical hits without regenerating LOCOMO full-run artifacts.

Query-decomposition recall slice delivered: `TestRecallQueryDecompositionRetrievesEachSubQuestionFact` proves multi-part questions can split into subqueries, retrieve each required stable-ID fact, and merge results before scoring without regenerating LOCOMO full-run artifacts.

Temporal current-truth routing slice delivered: `TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence` proves current facts can outrank superseded evidence while preserving superseded candidates and stable-ID memories without regenerating LOCOMO full-run artifacts.

Speaker who-said-what routing slice delivered: `TestRecallSpeakerRoutingKeepsWhoSaidWhatInBranch` proves explicit speaker provenance can select the right who-said-what branches while preserving stable-ID memories without regenerating LOCOMO full-run artifacts.

Failure-driven evaluation slice delivered: `TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets` and `TestWriteLocomoFailureAuditEmitsFailureBucket` prove wrong branch retrieval and missing companion memories can be separated into failure-audit buckets while preserving stable-ID memories without regenerating LOCOMO full-run artifacts.

LOCOMO answer-ready closeout delivered: the current evidence chain supports a plain answer to how to improve Goncho. Keep the frozen baseline metrics as the guardrail, target multi-hop recall_any@10 above `50%` and multi-hop strict_recall@10 above `25%`, continue hybrid candidate generation plus graph expansion, query decomposition, coverage-aware selection, temporal routing, speaker routing, and failure-bucket audits, and read backend-comparison `failure_buckets` beside rank categories. The delivered loop proves graph companions, query decomposition, temporal routing, speaker routing, failure-audit buckets, backend-comparison `failure_buckets`, and public docs guards without answer hints, LLM judges, answer-text scoring, or frozen artifact regeneration. Remaining benchmark backlog: choose an approved retrieval slice, then generate a new date-stamped full LOCOMO run only when the change is ready to compare against the frozen evidence.

It should test:

- long conversations,
- temporal memory,
- multi-session recall,
- evolving facts,
- contradictions over time,
- relationship and event changes.

Goncho-specific questions:

- Does Goncho preserve older truth while surfacing current truth?
- Does it handle changed preferences and changed relationships?
- Can it answer from memory without leaking stale facts as current facts?
- Can review/staleness warnings explain uncertainty?

Expected outputs:

- JSON result report,
- markdown summary generated from JSON,
- failure audit JSONL,
- contradiction/staleness audit where applicable,
- latency/RSS metrics.

## Phase 3: InfiniteBench and RULER

These benchmarks stress scale.

They should test:

- retrieval with huge memory pools,
- buried facts,
- distractors,
- structured retrieval,
- memory growth degradation,
- context budget pressure.

Goncho-specific measurements:

- recall as memory count grows,
- latency as memory count grows,
- RSS as memory count grows,
- degradation curves under added noise,
- token-budget pass rate.

## Phase 4: BABILong

BABILong is synthetic and controlled. It is not realistic enough alone, but it is useful scientifically.

It should test:

- temporal reasoning,
- entity tracking,
- simple multi-hop consistency,
- repeated facts under distractors.

Goncho-specific measurements:

- exact answer evidence recall,
- relation-chain retrieval,
- temporal ordering errors,
- false positives under similar entity names.

## Phase 5: BEIR

BEIR is not agent-memory-specific, but it is important for credibility.

It should compare Goncho against:

- random,
- BM25,
- SQLite FTS5,
- vector-only,
- hybrid BM25+vector where available.

Goncho-specific question:

- Can Goncho remain credible as an information-retrieval system while adding agent-memory semantics like scope, lifecycle, review, and trust warnings?

## Phase 6: Real-world replay benchmark

This is the most important eventual benchmark.

Synthetic benchmarks do not prove agent utility. Goncho should eventually replay real sessions such as:

- real coding tasks,
- real chat preferences,
- rejected approaches,
- stale code paths,
- recurring mistakes,
- user corrections,
- handoffs and compactions.

Example checks:

- “Three days ago the user rejected Redis. Did Goncho remember that?”
- “The file moved after memory was written. Did Goncho verify live state before trusting it?”
- “The agent repeated a failed Docker fix before. Did Goncho warn?”
- “A prompt-injection import entered memory. Did Goncho quarantine it?”

Outputs:

- replay fixture,
- memory state before task,
- retrieved context,
- action taken,
- expected memory behavior,
- failure reason if behavior differs.

## Metrics every serious benchmark should report

Do not report only “recall.” Report:

- recall@K,
- recall_any@K when the benchmark uses any-gold-session scoring,
- MRR,
- NDCG where applicable,
- latency min/p50/p95/max,
- RSS / peak memory,
- database size,
- memory count and total token estimate,
- degradation as distractors increase,
- stale-memory warning rate,
- contradiction handling accuracy,
- leakage counts,
- failure categories.

## External-backend comparison rule

The benchmark harness must stay more trustworthy than any backend.

For agentmemory, mem0, Goncho, BM25, SQLite FTS5, or future systems:

- adapters stay isolated,
- scoring stays centralized,
- every backend uses the same JSONL,
- every backend uses the same memory IDs,
- every backend reports the same metrics,
- every backend gets the same leakage checks,
- every backend gets the same failure taxonomy,
- no adapter may alter scoring semantics.

This turns the harness into persistent-agent-memory evaluation infrastructure, not just a Goncho-specific benchmark script.

## Required scientific controls

Every benchmark should include:

1. Pinned dataset source and revision.
2. Raw artifact checksum.
3. Conversion script.
4. Converted artifact checksum when practical.
5. Deterministic scoring by evidence ID, not LLM judgment, unless explicitly running an answer-generation benchmark.
6. Baselines: random, BM25, SQLite FTS5, Goncho without current ranking, Goncho current.
7. Leakage checks:
   - query text not accidentally stored as memory,
   - gold IDs not present in retrievable content,
   - answer labels not indexed unless the benchmark intentionally includes them.
8. Failure audit:
   - query,
   - expected memory ID,
   - retrieved top 10,
   - rank of correct item if present,
   - likely miss reason.
9. One-command clean-room reproduction where licensing permits.
10. CI-safe smoke target with tiny pinned fixtures.

BEAM smoke slice delivered: `make bench-beam-smoke` runs the pinned raw HuggingFace-style fixture at `cmd/goncho-bench/testdata/beam-smoke/hf-beam-smoke.jsonl`, emits Mnemosyne-compatible results/summary/paired outcomes, and compares against the checked-in `mnemosyne-smoke` paired baseline with checksum diagnostics and a bootstrap-gated superiority verdict.

BEAM failure-audit slice delivered: `goncho-bench --beam-service-failures-out beam_failures.jsonl` writes one JSONL row per failed service-backed BEAM question, including query, expected memory IDs, retrieved top 10, rank of the first expected ID, likely failure mode, provenance/context/token-budget gates, and warning codes.

BEAM leakage-control slice delivered: `beam_e2e_results.json.metadata.diagnostics.leakage` reports question-text, stable-ID, ideal-answer-text, and rubric-label leakage examples for service-backed BEAM runs; `--fail-on-leakage` rejects blocking question/stable-ID/rubric contamination before scoring.

BEAM judge-export slice delivered: `goncho-bench --beam-service-judge-requests-out beam_judge_requests.jsonl` exports one JSONL row per service-backed BEAM question with selected recall context, answer prompts that exclude ideal-answer/rubric metadata, and separate judge prompts carrying the preserved BEAM ideal answer and rubric.

BEAM judged-artifact import slice delivered: `goncho-bench --beam-service-judgments-in beam_judge_results.jsonl` merges external official-style answer/judge rows into Mnemosyne-compatible results, summary, and paired outcomes while keeping recall provenance and judge-source diagnostics.

BEAM judgment-completeness gate delivered: judged imports fail by default on missing or unmatched rows before comparable artifacts are written; `--beam-service-allow-partial-judgments` is available only for diagnostic partial runs that keep missing/unmatched counts in metadata.

BEAM nested-judgment import delivered: `--beam-service-judgments-in` accepts nested Mnemosyne-compatible `beam_e2e_results.json` files as well as flat JSONL rows, inheriting conversation/scale identity from result groups before applying the same completeness gate.

BEAM Mnemosyne-qid import delivered: nested judged rows can match by exact source qid or by conversation/scale/ability/question when Mnemosyne emits `conversation_id:qN` qids that differ from Goncho's converted BEAM qids, while still reporting missing/unmatched rows through the strict completeness gate.

BEAM result-to-paired import delivered: `--beam-paired-results-in` converts nested Mnemosyne-compatible `beam_e2e_results.json` files into append-only `paired_outcomes.jsonl` rows, and paired comparison now joins exact qids first then exact conversation/scale/ability/question so real Mnemosyne qids can compare against Goncho source qids without rewrite scripts.

## Framing

The public framing should remain:

> Goncho is being evaluated as a long-term memory retrieval system for agents, not just a vector store.

That means retrieval scores are only one layer. Goncho also needs to prove scope isolation, trust preservation, stale-memory behavior, contradiction handling, negative memory, and real agent utility.
