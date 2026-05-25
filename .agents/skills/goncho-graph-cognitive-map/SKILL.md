---
name: goncho-graph-cognitive-map
description: Implement Goncho graph and cognitive-map recall. Use when working on relation extraction, entity links, graph-expanded retrieval, topology, branch routing, or multi-hop recall provenance.
---

# Goncho Graph and Cognitive Map

## Quick start

Load `goncho-tdd-implementation`, then start from a failing recall/context test that graph data makes pass.

## Workflow

1. **Pick one graph behavior**
   - Conservative entity/relation extraction.
   - Pending relation candidates and review acceptance.
   - Multi-hop recall that lexical search misses.
   - Cognitive-map branch gating.
   - Relation-path provenance in results/context.
2. **Write the contract test**
   - Good names: `TestGraphRecallConnectsOwnerThroughServiceRelation`, `TestRelationCandidatesRemainPendingBeforeReview`, `TestCognitiveMapSuppressesLowActivationBranches`, `TestContextIncludesRelationPathCitation`.
3. **Implement minimally**
   - Store evidence IDs for every node/relation.
   - Keep speculative relations pending until accepted.
   - Budget graph expansion and keep lexical/local recall working without graph data.
4. **Verify**
   - Run the narrow graph test, relevant recall/context tests, then `go test ./...`.

## Design rules

- Fewer edges beats hallucinated topology.
- Relation confidence must affect review state, ranking, inclusion, or warning behavior.
- Graph-expanded results must explain the relation path used.
- Graph must enhance local recall, not become required for basic memory.

## Skill contract

### Entry protocol
- Trivial: answer architecture questions using current files/tests.
- Medium ambiguity: propose one observable recall improvement and ask only which path should be optimized first.
- High ambiguity/risk: stop before wholesale graph schema rewrites or benchmark artifact regeneration.

### Topology check
- State/ownership: entity store, relation candidates, accepted relations, recall pipeline, context output.
- Feedback/validation: one lexical-miss or branch-gating test plus provenance assertions.
- Blast radius: ranking, context budget, retrieval latency, review queues, and adapter compatibility.
- Timing/ordering: extraction before review, acceptance before use as truth, replay/import determinism.

### Verification gate
Done requires tested graph read/write, at least one search/context path using graph data, visible relation provenance, unrelated-branch suppression, and `go test ./...` pass or blocker output.

### Red lines
- Do not promote speculative extracted relations as truth.
- Do not add topology structs unused by retrieval/context/review.
- Do not return graph-expanded memories without path citations.
- Do not tune against benchmark gold IDs or regenerate frozen full-run artifacts unless explicitly requested.

### Output contract
End with: graph behavior covered, test names, provenance shape, validation commands, and recall/latency risks.
