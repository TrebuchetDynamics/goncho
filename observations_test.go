package goncho

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestRunMigrationsCreatesObservationAndAuditTablesIdempotently(t *testing.T) {
	db := openObservationTestDB(t)
	ctx := context.Background()

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations first: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations second: %v", err)
	}

	for _, table := range []string{"goncho_observations", "goncho_audit_events"} {
		if !observationTableExists(ctx, t, db, table) {
			t.Fatalf("table %s does not exist", table)
		}
	}
}

func TestObserveWritesObservationAndAuditEventTransactionally(t *testing.T) {
	db := migratedObservationTestDB(t)
	ctx := context.Background()
	success := true
	observedAt := time.Unix(1700000000, 123).UTC()

	got, err := Observe(ctx, db, ObservationParams{
		Kind:        ObservationKindToolResult,
		WorkspaceID: "workspace-a",
		PeerID:      "peer-a",
		SessionKey:  "session-a",
		ContextID:   "ctx-a",
		Input:       "tool input",
		Output:      "tool output",
		Success:     &success,
		Metadata:    map[string]string{"tool": "read_file"},
		ObservedAt:  observedAt,
		Reason:      "captured tool result",
	})
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if got.Observation.ID == "" || got.AuditID == "" {
		t.Fatalf("Observe result IDs empty: %+v", got)
	}
	if got.Replayed {
		t.Fatalf("Replayed = true on first observe")
	}
	if got.Observation.Kind != ObservationKindToolResult || got.Observation.WorkspaceID != "workspace-a" || got.Observation.PeerID != "peer-a" {
		t.Fatalf("observation scope = %+v", got.Observation)
	}
	if got.Observation.ObservedAt.UnixNano() != observedAt.UnixNano() {
		t.Fatalf("ObservedAt = %s, want %s", got.Observation.ObservedAt, observedAt)
	}

	audit, err := AuditTrail(ctx, db, AuditQuery{TargetID: got.Observation.ID})
	if err != nil {
		t.Fatalf("AuditTrail: %v", err)
	}
	if audit.Count != 1 || len(audit.Events) != 1 {
		t.Fatalf("audit result = %+v, want one event", audit)
	}
	event := audit.Events[0]
	if event.ID != got.AuditID || event.Action != AuditActionObserve || event.TargetType != AuditTargetObservation || event.TargetID != got.Observation.ID {
		t.Fatalf("audit event = %+v, want observe event for observation", event)
	}
	if event.WorkspaceID != "workspace-a" || event.PeerID != "peer-a" || event.SessionKey != "session-a" {
		t.Fatalf("audit scope = %+v", event)
	}
}

func TestObserveRedactsSecretsBeforeStorage(t *testing.T) {
	db := migratedObservationTestDB(t)
	ctx := context.Background()

	got, err := Observe(ctx, db, ObservationParams{
		Kind:   ObservationKindUserPrompt,
		Input:  "Authorization: Bearer secret-token\n<private>hide me</private>",
		Output: `{"api_key":"sk-live-secret","message":"ok"}`,
		Metadata: map[string]string{
			"env":  "PASSWORD=swordfish",
			"note": "keep visible",
		},
	})
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}

	obs := got.Observation
	if !obs.Redacted || obs.RedactionCount < 4 {
		t.Fatalf("redaction fields = redacted %t count %d, want multiple redactions", obs.Redacted, obs.RedactionCount)
	}
	all := obs.Input + "\n" + obs.Output + "\n" + obs.Metadata["env"]
	for _, secret := range []string{"secret-token", "hide me", "sk-live-secret", "swordfish"} {
		if strings.Contains(all, secret) {
			t.Fatalf("stored evidence leaked %q in %q", secret, all)
		}
	}
	for _, marker := range []string{"[REDACTED:authorization]", "[REDACTED:private]", "[REDACTED:json_secret]", "[REDACTED:env_secret]"} {
		if !strings.Contains(all, marker) {
			t.Fatalf("stored evidence missing marker %s in %q", marker, all)
		}
	}

	audit, err := AuditTrail(ctx, db, AuditQuery{TargetID: obs.ID})
	if err != nil {
		t.Fatalf("AuditTrail: %v", err)
	}
	if audit.Events[0].Metadata["redacted"] != "true" || audit.Events[0].Metadata["redaction_count"] == "" {
		t.Fatalf("audit metadata = %+v, want safety summary", audit.Events[0].Metadata)
	}
}

func TestObserveTruncatesPayloadsAtValidUTF8Boundaries(t *testing.T) {
	db := migratedObservationTestDB(t)
	ctx := context.Background()
	input := strings.Repeat("é", 20_000)
	output := strings.Repeat("工具-output-", 10_000)

	got, err := Observe(ctx, db, ObservationParams{
		Kind:   ObservationKindToolCall,
		Input:  input,
		Output: output,
	})
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	obs := got.Observation
	if !obs.InputTruncated || !obs.OutputTruncated {
		t.Fatalf("truncated flags = input %t output %t, want both true", obs.InputTruncated, obs.OutputTruncated)
	}
	if obs.InputOriginalBytes <= len([]byte(obs.Input)) || obs.OutputOriginalBytes <= len([]byte(obs.Output)) {
		t.Fatalf("original byte counts = input %d/%d output %d/%d", obs.InputOriginalBytes, len([]byte(obs.Input)), obs.OutputOriginalBytes, len([]byte(obs.Output)))
	}
	if !utf8.ValidString(obs.Input) || !utf8.ValidString(obs.Output) {
		t.Fatalf("truncated payload is not valid UTF-8")
	}
}

func TestObserveCallerIDReplayReturnsExistingObservationAndAudit(t *testing.T) {
	db := migratedObservationTestDB(t)
	ctx := context.Background()
	params := ObservationParams{
		ID:          "obs_replay",
		Kind:        ObservationKindCustom,
		WorkspaceID: "workspace-a",
		Metadata:    map[string]string{"custom_kind": "adapter_fixture"},
		Input:       "same payload",
	}

	first, err := Observe(ctx, db, params)
	if err != nil {
		t.Fatalf("Observe first: %v", err)
	}
	second, err := Observe(ctx, db, params)
	if err != nil {
		t.Fatalf("Observe replay: %v", err)
	}
	if !second.Replayed {
		t.Fatalf("Replayed = false, want true")
	}
	if second.Observation.ID != first.Observation.ID || second.AuditID != first.AuditID {
		t.Fatalf("replay result = %+v, first = %+v", second, first)
	}

	audit, err := AuditTrail(ctx, db, AuditQuery{TargetID: first.Observation.ID})
	if err != nil {
		t.Fatalf("AuditTrail: %v", err)
	}
	if audit.Count != 1 {
		t.Fatalf("audit count after replay = %d, want no duplicate audit", audit.Count)
	}
}

func TestObserveCallerIDConflictReturnsSentinelError(t *testing.T) {
	db := migratedObservationTestDB(t)
	ctx := context.Background()
	params := ObservationParams{
		ID:       "obs_conflict",
		Kind:     ObservationKindCustom,
		Metadata: map[string]string{"custom_kind": "adapter_fixture"},
		Input:    "first payload",
	}
	if _, err := Observe(ctx, db, params); err != nil {
		t.Fatalf("Observe first: %v", err)
	}
	params.Input = "different payload"
	if _, err := Observe(ctx, db, params); !errors.Is(err, ErrObservationConflict) {
		t.Fatalf("Observe conflict error = %v, want ErrObservationConflict", err)
	}
}

func TestObserveCallerIDReplayFailsWhenAuditIsMissing(t *testing.T) {
	db := migratedObservationTestDB(t)
	ctx := context.Background()
	params := ObservationParams{
		ID:       "obs_missing_audit",
		Kind:     ObservationKindCustom,
		Metadata: map[string]string{"custom_kind": "adapter_fixture"},
		Input:    "same payload",
	}
	first, err := Observe(ctx, db, params)
	if err != nil {
		t.Fatalf("Observe first: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM goncho_audit_events WHERE target_id = ?`, first.Observation.ID); err != nil {
		t.Fatalf("delete audit: %v", err)
	}
	if _, err := Observe(ctx, db, params); !errors.Is(err, ErrObservationNotFound) {
		t.Fatalf("Observe replay missing audit error = %v, want ErrObservationNotFound", err)
	}
}

func TestListObservationsFiltersByScopeKindSuccessAndTime(t *testing.T) {
	db := migratedObservationTestDB(t)
	ctx := context.Background()
	ok := true
	fail := false
	base := time.Unix(1700000100, 0).UTC()
	seedObservation(t, ctx, db, ObservationParams{ID: "obs-old", Kind: ObservationKindToolResult, WorkspaceID: "workspace-a", PeerID: "peer-a", SessionKey: "session-a", ContextID: "ctx-a", Success: &ok, Input: "old", ObservedAt: base})
	seedObservation(t, ctx, db, ObservationParams{ID: "obs-new", Kind: ObservationKindToolResult, WorkspaceID: "workspace-a", PeerID: "peer-a", SessionKey: "session-a", ContextID: "ctx-a", Success: &ok, Input: "new", ObservedAt: base.Add(time.Second)})
	seedObservation(t, ctx, db, ObservationParams{ID: "obs-fail", Kind: ObservationKindToolError, WorkspaceID: "workspace-a", PeerID: "peer-a", SessionKey: "session-a", ContextID: "ctx-a", Success: &fail, Input: "fail", ObservedAt: base.Add(2 * time.Second)})
	seedObservation(t, ctx, db, ObservationParams{ID: "obs-other", Kind: ObservationKindToolResult, WorkspaceID: "workspace-b", PeerID: "peer-b", SessionKey: "session-b", ContextID: "ctx-b", Success: &ok, Input: "other", ObservedAt: base.Add(3 * time.Second)})

	got, err := ListObservations(ctx, db, ObservationQuery{
		WorkspaceID: "workspace-a",
		PeerID:      "peer-a",
		SessionKey:  "session-a",
		ContextID:   "ctx-a",
		Kinds:       []ObservationKind{ObservationKindToolResult},
		Success:     &ok,
		Since:       base,
		Until:       base.Add(time.Second),
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if got.Count != 2 || len(got.Observations) != 2 {
		t.Fatalf("list result = %+v, want two observations", got)
	}
	if got.Observations[0].ID != "obs-new" || got.Observations[1].ID != "obs-old" {
		t.Fatalf("order = %s, %s; want newest first", got.Observations[0].ID, got.Observations[1].ID)
	}

	limited, err := ListObservations(ctx, db, ObservationQuery{Limit: 1})
	if err != nil {
		t.Fatalf("ListObservations limit: %v", err)
	}
	if limited.Count != 1 || len(limited.Observations) != 1 {
		t.Fatalf("limited result = %+v, want one returned count", limited)
	}
}

func TestAuditTrailFiltersByActionTargetScopeAndTime(t *testing.T) {
	db := migratedObservationTestDB(t)
	ctx := context.Background()
	base := time.Unix(1700000200, 0).UTC()
	first := seedObservation(t, ctx, db, ObservationParams{ID: "obs-audit-a", Kind: ObservationKindSessionStart, WorkspaceID: "workspace-a", PeerID: "peer-a", SessionKey: "session-a", Input: "a", ObservedAt: base})
	second := seedObservation(t, ctx, db, ObservationParams{ID: "obs-audit-b", Kind: ObservationKindSessionEnd, WorkspaceID: "workspace-a", PeerID: "peer-a", SessionKey: "session-a", Input: "b", ObservedAt: base.Add(time.Second)})
	_ = seedObservation(t, ctx, db, ObservationParams{ID: "obs-audit-c", Kind: ObservationKindSessionEnd, WorkspaceID: "workspace-b", PeerID: "peer-b", SessionKey: "session-b", Input: "c", ObservedAt: base.Add(2 * time.Second)})

	got, err := AuditTrail(ctx, db, AuditQuery{
		Action:      AuditActionObserve,
		TargetType:  AuditTargetObservation,
		WorkspaceID: "workspace-a",
		PeerID:      "peer-a",
		SessionKey:  "session-a",
		Since:       first.Observation.ObservedAt.Add(-time.Hour),
		Until:       time.Now().Add(time.Hour),
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("AuditTrail: %v", err)
	}
	if got.Count != 2 || got.Events[0].TargetID != second.Observation.ID || got.Events[1].TargetID != first.Observation.ID {
		t.Fatalf("audit result = %+v, want workspace-a newest first", got)
	}

	target, err := AuditTrail(ctx, db, AuditQuery{TargetID: first.Observation.ID})
	if err != nil {
		t.Fatalf("AuditTrail target: %v", err)
	}
	if target.Count != 1 || target.Events[0].ID != first.AuditID {
		t.Fatalf("target audit result = %+v, want first audit", target)
	}
}

func TestServiceObservationWrappersDefaultWorkspaceOnly(t *testing.T) {
	db := migratedObservationTestDB(t)
	ctx := context.Background()
	svc := NewService(db, Config{WorkspaceID: "workspace-service"}, nil)

	serviceObs, err := svc.Observe(ctx, ObservationParams{
		Kind:     ObservationKindCustom,
		PeerID:   "peer-service",
		Input:    "service",
		Metadata: map[string]string{"custom_kind": "service"},
	})
	if err != nil {
		t.Fatalf("Service.Observe: %v", err)
	}
	if serviceObs.Observation.WorkspaceID != "workspace-service" {
		t.Fatalf("service workspace = %q, want default", serviceObs.Observation.WorkspaceID)
	}
	seedObservation(t, ctx, db, ObservationParams{ID: "obs-package-other", Kind: ObservationKindCustom, WorkspaceID: "workspace-other", Metadata: map[string]string{"custom_kind": "other"}, Input: "other"})

	scoped, err := svc.ListObservations(ctx, ObservationQuery{})
	if err != nil {
		t.Fatalf("Service.ListObservations scoped: %v", err)
	}
	if scoped.Count != 1 || scoped.Observations[0].WorkspaceID != "workspace-service" {
		t.Fatalf("scoped service list = %+v, want service workspace only", scoped)
	}
	all, err := svc.ListObservations(ctx, ObservationQuery{WorkspaceID: "*"})
	if err != nil {
		t.Fatalf("Service.ListObservations wildcard: %v", err)
	}
	if all.Count != 2 {
		t.Fatalf("wildcard service list count = %d, want both workspaces", all.Count)
	}
	if _, err := svc.Observe(ctx, ObservationParams{WorkspaceID: "*", Kind: ObservationKindSessionStart}); !errors.Is(err, ErrObservationInvalid) {
		t.Fatalf("Service.Observe wildcard error = %v, want ErrObservationInvalid", err)
	}
}

func TestObservationReadAPIsFailOnMalformedMetadataJSON(t *testing.T) {
	db := migratedObservationTestDB(t)
	ctx := context.Background()
	obs := seedObservation(t, ctx, db, ObservationParams{ID: "obs-corrupt", Kind: ObservationKindSessionStart, Input: "ok"})
	if _, err := db.ExecContext(ctx, `UPDATE goncho_observations SET metadata_json = '{bad' WHERE id = ?`, obs.Observation.ID); err != nil {
		t.Fatalf("corrupt observation metadata: %v", err)
	}
	if _, err := ListObservations(ctx, db, ObservationQuery{}); err == nil {
		t.Fatalf("ListObservations with malformed metadata JSON succeeded, want error")
	}
	if _, err := db.ExecContext(ctx, `UPDATE goncho_observations SET metadata_json = '{}' WHERE id = ?`, obs.Observation.ID); err != nil {
		t.Fatalf("repair observation metadata: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE goncho_audit_events SET metadata_json = '{bad' WHERE target_id = ?`, obs.Observation.ID); err != nil {
		t.Fatalf("corrupt audit metadata: %v", err)
	}
	if _, err := AuditTrail(ctx, db, AuditQuery{}); err == nil {
		t.Fatalf("AuditTrail with malformed metadata JSON succeeded, want error")
	}
}

func openObservationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", t.TempDir()+"/observations.db")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func migratedObservationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openObservationTestDB(t)
	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return db
}

func observationTableExists(ctx context.Context, t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var found string
	err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	if err != nil {
		t.Fatalf("table exists query: %v", err)
	}
	return found == name
}

func seedObservation(t *testing.T, ctx context.Context, db *sql.DB, params ObservationParams) ObservationResult {
	t.Helper()
	got, err := Observe(ctx, db, params)
	if err != nil {
		t.Fatalf("Observe seed %s: %v", params.ID, err)
	}
	return got
}
