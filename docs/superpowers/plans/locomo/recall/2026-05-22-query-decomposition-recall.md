# LOCOMO Query Decomposition Recall Implementation Plan

Start with TestRecallQueryDecompositionRetrievesEachSubQuestionFact. split multi-part questions, then merge and deduplicate by stable `memory_id`. Preserve stable inserted `memory_id` identity and use no answer hints, no LLM judges, no answer-text scoring.

Validation:
- go test . -run TestRecallQueryDecompositionRetrievesEachSubQuestionFact -count=1
- go test ./... -count=1
