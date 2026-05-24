# internal/goncho/

Goncho is the local memory/service substrate extracted from Gormes. It owns the SQLite-backed service facade, recall traces, memory-tool compatibility, and the new raw evidence lane.

## Responsibility

Expose a Go-native memory facade for agents and hosts: profile cards, conclusions, session context, recall traces, local memory tools, dynamic agent/webhook helpers, memory tier/ACL policy, observation/audit capture, review-item governance, skill-learning proposal governance, local dream work-intent scheduling, and queue-status diagnostics.

## Design

Core APIs are package-level functions plus thin `Service` wrappers. `RunMigrations`, `Observe`, `ListObservations`, `AuditTrail`, `CreateReviewItem`, `ListReviewItems`, `ResolveReviewItem`, `SubmitSkillLearningProposal`, `GetSkillLearningProposal`, `ListSkillLearningProposals`, `ScheduleDream`, and `ReadQueueStatus` create a claims/evidence/review/learning-governance/work-intent/diagnostics foundation without feeding raw observations into recall yet. `internal/memorypolicy` owns memory tier normalization, hierarchy, default source-tier mapping, ACL SQL, and explicit grant checks; `internal/observationlog` owns observation/audit storage; `internal/reviewlog` owns review-item storage and review-required context evidence; `internal/skillproposals` owns skill-learning proposal storage and review state transitions; `internal/dreamscheduler` owns dream scheduler eligibility, dedupe, cancellation, and work-intent evidence; `internal/queuestatus` owns queue-status read-model counts and diagnostics evidence.

## Flow

Host or service code writes scoped events through `Observe`; Goncho redacts, truncates, checksums, stores the observation, and writes an audit row in one transaction. Existing search/context flows continue to read conclusions, turns, summaries, peer cards, and recall traces.

## Integration

Gormes and other hosts should call the Go API or service wrappers after running `RunMigrations`. The local `memory` and `session` packages provide extraction-safe compatibility for SQLite setup, memory V1 fixtures, FTS-backed turn search, and in-memory session metadata tests. The public `agentmemory` package is a source-pinned architecture mirror/port matrix for rohitg00/agentmemory commit `355124141625ccc0d740ae08ddaaf77fe2c165ae`, mapping its pipeline, tiers, retrieval streams, hooks, and 53 MCP tools onto delivered, partial, adapter-owned, deferred, or excluded Goncho seams; it also exposes agentmemory-compatible executable aliases for save, smart search, recall, and profile backed by the local Goncho service.
