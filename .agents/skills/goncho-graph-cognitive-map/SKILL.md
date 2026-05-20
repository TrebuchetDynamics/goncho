---
name: goncho-graph-cognitive-map
description: Use when implementing Goncho graph retrieval, relation extraction, cognitive-map routing, topology, entity links, or multi-hop recall.
---

# Goncho Graph and Cognitive Map

## Goal

Move Goncho beyond similar-text recall into relationship-aware orientation.

## Required TDD Shape

**REQUIRED SUB-SKILL:** Use `goncho-tdd-implementation` first. Every graph slice needs a failing test that proves one observable recall improvement:

- entity extraction creates conservative nodes or relations,
- relation candidates are pending until accepted,
- graph expansion finds a multi-hop fact that lexical search alone would miss,
- cognitive-map branch gating suppresses unrelated memories,
- context packs cite the relation path used.

## Quick Reference

| Need | Prove with |
| --- | --- |
| Extract entities | Conservative nodes/relations are created with evidence IDs |
| Review relation candidates | Candidate state stays pending until accepted |
| Improve recall | Multi-hop graph path finds a fact lexical search misses |
| Gate context | Low-activation or unrelated branches stay out of packs |
| Explain retrieval | Result cites relation path and provenance |

## Minimal Contract Examples

Good test names:

- `TestGraphRecallConnectsOwnerThroughServiceRelation`
- `TestRelationCandidatesRemainPendingBeforeReview`
- `TestCognitiveMapSuppressesLowActivationBranches`
- `TestContextIncludesRelationPathCitation`

## Design Rules

- Start conservative: fewer edges beats hallucinated edges.
- Preserve evidence IDs for every relation.
- Keep graph retrieval budgeted and explainable.
- Relation confidence must affect ranking or review state.
- Do not make graph mandatory for basic local memory.

## Done Criteria

- graph storage has tested read/write behavior,
- at least one search/context path uses graph data,
- relation provenance is visible,
- unrelated branches stay out of context,
- `go test ./...` passes.

## Common Mistakes

| Mistake | Fix |
| --- | --- |
| Building topology structs before recall uses them | Start from a failing recall/context test |
| Extracting speculative relations as truth | Keep candidates pending and evidence-backed |
| Returning graph-expanded memories without explanation | Include relation path citations in the result |
| Making graph required for simple local recall | Keep lexical/local memory path working without graph data |

## Avoid

- extracting speculative entity graphs without review,
- adding topology structs that retrieval never uses,
- returning graph-expanded memories without explanation.
