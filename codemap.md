# internal/goncho/

Goncho is the local memory/service substrate extracted from Gormes. It owns the SQLite-backed service facade, recall traces, memory-tool compatibility, and the new raw evidence lane.

## Responsibility

Expose a Go-native memory facade for agents and hosts: profile cards, conclusions, session context, recall traces, local memory tools, dynamic agent/webhook helpers, and observation/audit capture.

## Design

Core APIs are package-level functions plus thin `Service` wrappers. `RunMigrations`, `Observe`, `ListObservations`, and `AuditTrail` create a claims/evidence foundation without feeding raw observations into recall yet. Observations live in `goncho_observations`; observe provenance lives in `goncho_audit_events`.

## Flow

Host or service code writes scoped events through `Observe`; Goncho redacts, truncates, checksums, stores the observation, and writes an audit row in one transaction. Existing search/context flows continue to read conclusions, turns, summaries, peer cards, and recall traces.

## Integration

Gormes and other hosts should call the Go API or service wrappers after running `RunMigrations`. The local `memory` and `session` packages provide extraction-safe compatibility for SQLite setup, memory V1 fixtures, FTS-backed turn search, and in-memory session metadata tests.
