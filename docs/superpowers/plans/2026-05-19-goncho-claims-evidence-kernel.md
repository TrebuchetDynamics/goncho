# Goncho Claims/Evidence Kernel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Goncho's raw evidence lane: migrations, observation capture, observation listing, observe audit trail, service wrappers, and tests.

**Architecture:** Keep observations separate from turns, conclusions, recall, and current memory tools. Package-level APIs own the core behavior; `Service` wrappers add only the configured workspace default. Evidence is sanitized before storage and append-only with retry-safe idempotency.

**Tech Stack:** Go, `database/sql`, SQLite via existing repo driver, table-driven unit tests, stable JSON golden helpers for existing proof fixtures.

---

## File Structure

- Modify: `docs/superpowers/specs/2026-05-19-goncho-claims-evidence-kernel-design.md` to include late accepted decisions.
- Create: `docs/superpowers/plans/2026-05-19-goncho-claims-evidence-kernel.md` for this execution plan.
- Create: `test_golden_helpers_test.go` for Goncho-local stable JSON marshal/compare helpers.
- Modify: `proof_matrix_test.go` and `recall_benchmark_test.go` to remove the forbidden `gormes-agent/internal/transcript` import.
- Create: `observations_test.go` for TDD coverage of migrations, observe, redaction, truncation, idempotency, listing, service wrappers, and corrupt-row failures.
- Create: `migrations.go` for `RunMigrations`.
- Create: `observations.go` for observation types, validation, redaction, truncation, checksum, `Observe`, `ListObservations`, and service wrappers.
- Create: `audit.go` for audit types, audit write helper, `AuditTrail`, and service wrapper.
- Optionally modify: `codemap.md` only if it exists and needs a concise note.

## Task 1: Patch Existing Test Harness

**Files:**
- Create: `test_golden_helpers_test.go`
- Modify: `proof_matrix_test.go`
- Modify: `recall_benchmark_test.go`

- [ ] **Step 1: Write local golden helper**

Add test-only helpers:

```go
type gonchoJSONDiff struct {
	Path    string
	Message string
}

func (d gonchoJSONDiff) Error() string
func marshalStableJSON(t any) ([]byte, error)
func compareGoldenJSON(wantRaw, gotRaw []byte) error
func normalizeGoldenJSON(raw []byte) (any, error)
func firstJSONDiff(path string, want, got any) *gonchoJSONDiff
```

`marshalStableJSON` should use `json.MarshalIndent` plus a trailing newline. `compareGoldenJSON` should decode both documents into `any`, compare with `reflect.DeepEqual`, and return a `gonchoJSONDiff` pointing at the first mismatched path.

- [ ] **Step 2: Replace forbidden imports**

In `proof_matrix_test.go`, remove:

```go
"github.com/TrebuchetDynamics/gormes-agent/internal/transcript"
```

Replace `transcript.MarshalStableJSON`, `transcript.CompareGoldenJSON`, and `transcript.JSONDiff` with `marshalStableJSON`, `compareGoldenJSON`, and `gonchoJSONDiff`.

Repeat the same replacement in `recall_benchmark_test.go`.

- [ ] **Step 3: Verify harness compiles far enough to expose current package errors**

Run: `go test . -run 'TestGonchoProofMatrixFullLocalReportFixture|TestRecallBenchmarkReportFixture' -count=1`

Expected: no failure mentioning `gormes-agent/internal/transcript`. Other existing package failures may remain and should be reported separately if unrelated.

## Task 2: Red Tests for Migrations and Basic Observe

**Files:**
- Create: `observations_test.go`
- Create later: `migrations.go`, `observations.go`, `audit.go`

- [ ] **Step 1: Write failing migration/observe tests**

Add tests using `sql.Open("sqlite", ":memory:")` and `_ "github.com/ncruces/go-sqlite3/driver"`:

```go
func TestRunMigrationsCreatesObservationAndAuditTablesIdempotently(t *testing.T)
func TestObserveWritesObservationAndAuditEventTransactionally(t *testing.T)
```

The tests should assert both tables exist, `RunMigrations` can be called twice, `Observe` returns a non-empty observation ID and audit ID, and `AuditTrail` returns an `observe` event for the observation.

- [ ] **Step 2: Run red**

Run: `go test . -run 'TestRunMigrationsCreatesObservationAndAuditTablesIdempotently|TestObserveWritesObservationAndAuditEventTransactionally' -count=1`

Expected: compile failure for missing `RunMigrations`, `Observe`, observation types, and audit types.

- [ ] **Step 3: Implement minimal green**

Add DDL and the smallest valid `Observe`/`AuditTrail` implementation to pass these two tests. Use a transaction for observe + audit insert.

- [ ] **Step 4: Run green**

Run the same targeted command.

Expected: both tests pass.

## Task 3: Red Tests for Safety and Idempotency

**Files:**
- Modify: `observations_test.go`
- Modify: `observations.go`
- Modify: `audit.go`

- [ ] **Step 1: Write failing safety/idempotency tests**

Add tests:

```go
func TestObserveRedactsSecretsBeforeStorage(t *testing.T)
func TestObserveTruncatesPayloadsAtValidUTF8Boundaries(t *testing.T)
func TestObserveCallerIDReplayReturnsExistingObservationAndAudit(t *testing.T)
func TestObserveCallerIDConflictReturnsSentinelError(t *testing.T)
func TestObserveCallerIDReplayFailsWhenAuditIsMissing(t *testing.T)
```

Assert stable markers such as `[REDACTED:authorization]`, `[REDACTED:env_secret]`, and `[REDACTED:private]`; assert original secret fragments are absent. Conflict should satisfy `errors.Is(err, ErrObservationConflict)`. Missing audit should satisfy `errors.Is(err, ErrObservationNotFound)`.

- [ ] **Step 2: Run red**

Run: `go test . -run 'TestObserve(Redacts|Truncates|CallerID)' -count=1`

Expected: failing assertions or missing symbols.

- [ ] **Step 3: Implement safety/idempotency**

Add validation, UTF-8 coercion, redaction, truncation, deterministic metadata JSON, canonical checksum, ID generation, replay, conflict detection, and sentinel errors.

- [ ] **Step 4: Run green**

Run the same targeted command.

Expected: all safety/idempotency tests pass.

## Task 4: Red Tests for Query APIs and Service Wrappers

**Files:**
- Modify: `observations_test.go`
- Modify: `observations.go`
- Modify: `audit.go`

- [ ] **Step 1: Write failing query tests**

Add tests:

```go
func TestListObservationsFiltersByScopeKindSuccessAndTime(t *testing.T)
func TestAuditTrailFiltersByActionTargetScopeAndTime(t *testing.T)
func TestServiceObservationWrappersDefaultWorkspaceOnly(t *testing.T)
func TestObservationReadAPIsFailOnMalformedMetadataJSON(t *testing.T)
```

Assert newest-first ordering, default/max limit behavior, `WorkspaceID: "*"` wildcard behavior only on service query wrappers, and `Service.Observe` rejecting wildcard workspace with `ErrObservationInvalid`.

- [ ] **Step 2: Run red**

Run: `go test . -run 'Test(ListObservations|AuditTrail|ServiceObservation|ObservationRead)' -count=1`

Expected: failing assertions or missing query behavior.

- [ ] **Step 3: Implement query behavior and wrappers**

Add filter builders that validate enums before SQL, use exact-match semantics, apply inclusive time bounds, return non-nil slices, and fail on corrupt metadata JSON.

- [ ] **Step 4: Run green**

Run the same targeted command.

Expected: all query/service tests pass.

## Task 5: Full Slice Verification

**Files:**
- Format touched Go files.
- Optionally modify: `codemap.md` if present.

- [ ] **Step 1: Format**

Run: `gofmt -w test_golden_helpers_test.go proof_matrix_test.go recall_benchmark_test.go observations_test.go migrations.go observations.go audit.go`

- [ ] **Step 2: Run focused tests**

Run: `go test . -run 'Test(RunMigrations|Observe|ListObservations|AuditTrail|ServiceObservation|ObservationRead)' -count=1`

Expected: observation/audit slice tests pass.

- [ ] **Step 3: Run full package signal**

Run: `go test . -count=1`

Expected: either pass, or report unrelated pre-existing failures with exact output. Do not claim full suite passes unless this command exits 0.

- [ ] **Step 4: Inspect diff**

Run: `git diff -- docs/superpowers/specs/2026-05-19-goncho-claims-evidence-kernel-design.md docs/superpowers/plans/2026-05-19-goncho-claims-evidence-kernel.md test_golden_helpers_test.go proof_matrix_test.go recall_benchmark_test.go observations_test.go migrations.go observations.go audit.go`

Expected: diff is limited to the accepted slice and the harness fix.

## Self-Review

- Spec coverage: the plan covers migration, raw evidence capture, redaction, truncation, idempotency, audit trail, query filters, service wrappers, harness unblock, and verification.
- Placeholder scan: no task uses TBD/TODO/fill-in language.
- Type consistency: public names match the accepted spec: `RunMigrations`, `Observe`, `ListObservations`, `AuditTrail`, `ObservationParams`, `ObservationQuery`, `AuditQuery`, `PeerID`, `SessionKey`, `ContextID`, `ObservedAt`, and the four sentinel errors.
