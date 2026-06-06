# Goncho Memory Evaluation

This context names Goncho's memory-retrieval evaluation concepts so benchmark work can distinguish generic retrieval quality from memory-system capabilities.

## Language

**LOCOMO**:
A long-conversation memory benchmark used here to evaluate whether Goncho retrieves stable evidence from conversation-scoped memories across temporal, speaker, and multi-hop questions.
_Avoid_: Generic search benchmark

**LongMemEval-S**:
A long-memory retrieval benchmark used here as a preservation guardrail while changing Goncho retrieval behavior.
_Avoid_: langv5

**Candidate Generation**:
The stage that admits memories into the set that can later be ranked or selected as evidence.
_Avoid_: Ranking, reranking

**Host-Provided Semantic Candidate Stream**:
A semantic retrieval input supplied by a host or benchmark adapter and fused into Goncho recall as provenance-bearing candidates without making embedding models part of Goncho core.
_Avoid_: Bundled embedding runtime, vector database as truth

**Companion Memory**:
A memory that is not sufficient alone but is needed alongside another retrieved memory to answer a multi-hop or context-dependent question.
_Avoid_: Duplicate memory, related hit

**Multi-Hop Retrieval**:
Retrieval where answering a question requires connecting two or more memories through an entity, event, speaker, temporal, or relationship link.
_Avoid_: Open-domain retrieval

**LOCOMO Retrieval Quality Slice**:
A benchmark-driven implementation slice that targets LOCOMO recall and failure-bucket gaps before broadening capture, integration, or product surface area.
_Avoid_: Integration polish, product UX slice

**Memory Impact Audit**:
A pre-action recall product that gathers stale code claims, negative memories, related action items, and prior failures for the files or symbols an agent is about to edit.
_Avoid_: Generic code graph impact analysis, PR review

**Recall Pipeline**:
Goncho's auditable retrieval path that scores, selects, and explains memory candidates with provenance. It is distinct from flat Search, which returns simple result rows.
_Avoid_: Search, default search

**Pipeline Benchmark System**:
A benchmark variant that evaluates projected Recall Pipeline output without redefining the existing Goncho Search baseline.
_Avoid_: Replacing goncho baseline, silent search change

**Recall Ranking Profile**:
A benchmark-oriented Recall Pipeline configuration that evaluates top-K evidence ranking separately from default host recall behavior.
_Avoid_: Default Service.Recall behavior

**Paired Delta Audit**:
A per-question comparison between two benchmark systems that classifies wins, losses, shared hits, shared misses, and rank regressions before changing retrieval behavior.
_Avoid_: Aggregate-only comparison, blind tuning

**Selection Loss**:
A recall failure where a gold memory was admitted as a candidate but was not selected into the benchmark top-K output.
_Avoid_: Candidate generation failure

**Rank-Low Candidate**:
A recall failure where a gold memory is present in the candidate set but scored too low to reach the selected top-K.
_Avoid_: Selection loss

## Example dialogue

Dev: “Should we tune the reranker to improve LOCOMO?”
Domain expert: “Only after candidate generation admits the missing evidence. A reranker cannot choose a memory that never became a candidate.”

Dev: “Should LongMemEval-S improve too?”
Domain expert: “Not necessarily. LongMemEval-S is the guardrail: LOCOMO changes should preserve its current strong result.”

Dev: “The failure says missing companion memory. Is that just a bad top-10 rank?”
Domain expert: “No. A companion memory is part of an evidence chain; the issue is whether the chain is present, not merely whether one isolated hit ranked higher.”

Dev: “Should Goncho ship an embedding model to improve LOCOMO?”
Domain expert: “Not in core. Use a host-provided semantic candidate stream so semantic recall is measurable without turning vectors into Goncho’s source of truth.”

Dev: “Can we borrow GitNexus impact analysis?”
Domain expert: “Borrow the guardrail shape, not the code graph analyzer. In Goncho, a Memory Impact Audit checks remembered evidence before action; it does not compute a full static blast radius.”
