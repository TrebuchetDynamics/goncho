---
name: goncho-graph-cognitive-map
description: Use when implementing Goncho graph retrieval, relation extraction, cognitive-map routing, topology, entity links, or multi-hop recall.
---

# Goncho Graph and Cognitive Map

## Goal

Move Goncho beyond similar-text recall into relationship-aware orientation.

## Required TDD Shape

Use `goncho-tdd-implementation` first. Every graph slice needs a failing test that proves one observable recall improvement:

- entity extraction creates conservative nodes or relations,
- relation candidates are pending until accepted,
- graph expansion finds a multi-hop fact that lexical search alone would miss,
- cognitive-map branch gating suppresses unrelated memories,
- context packs cite the relation path used.

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

## Avoid

- extracting speculative entity graphs without review,
- adding topology structs that retrieval never uses,
- returning graph-expanded memories without explanation.
