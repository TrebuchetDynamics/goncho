# Graph-Assisted LOCOMO Multi-Hop Recall Implementation Plan

Use superpowers:executing-plans. Start with TestGraphRecallConnectsOwnerThroughServiceRelation. graph-expanded candidates must carry `EvidenceItem{Kind: "graph"` and stable inserted `memory_id`. Keep no answer hints, no LLM judges, no answer-text scoring. Include coverage-aware selection and relation path provenance.

Validation:
- go test . -run TestGraphRecallConnectsOwnerThroughServiceRelation -count=1
- go test ./... -count=1
