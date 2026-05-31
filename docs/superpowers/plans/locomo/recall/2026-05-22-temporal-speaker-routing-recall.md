# LOCOMO Temporal and Speaker Routing Recall Implementation Plan

Start with TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence and TestRecallSpeakerRoutingKeepsWhoSaidWhatInBranch. Handle changed facts, chronology, and who-said-what using stable inserted `memory_id`; superseded evidence remains preserved. Use no answer hints, no LLM judges, no answer-text scoring.

Validation:
- go test . -run TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence -count=1
- go test ./... -count=1
