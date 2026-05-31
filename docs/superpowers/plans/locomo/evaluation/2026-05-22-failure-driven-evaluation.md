# LOCOMO Failure-Driven Evaluation Implementation Plan

Cover TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets and TestWriteLocomoFailureAuditEmitsFailureBucket. Classify wrong branch retrieval, missing companion memories, and failure-audit buckets using stable inserted `memory_id`. Keep no answer hints, no LLM judges, no answer-text scoring. Preserve frozen LOCOMO artifacts.

Validation:
- go test ./cmd/goncho-bench -run TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets -count=1
- go test ./... -count=1
