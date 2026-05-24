package goncho

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

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

func TestObservationsPublicFacadeObserveListAndAuditTrail(t *testing.T) {
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
	if got.Observation.ID == "" || got.AuditID == "" || got.Replayed {
		t.Fatalf("Observe result = %+v, want fresh observation and audit IDs", got)
	}

	listed, err := ListObservations(ctx, db, ObservationQuery{WorkspaceID: "workspace-a", Kinds: []ObservationKind{ObservationKindToolResult}})
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if listed.Count != 1 || listed.Observations[0].ID != got.Observation.ID {
		t.Fatalf("listed observations = %+v, want observed result", listed)
	}

	audit, err := AuditTrail(ctx, db, AuditQuery{TargetID: got.Observation.ID})
	if err != nil {
		t.Fatalf("AuditTrail: %v", err)
	}
	if audit.Count != 1 || audit.Events[0].ID != got.AuditID || audit.Events[0].Action != AuditActionObserve || audit.Events[0].TargetType != AuditTargetObservation {
		t.Fatalf("audit result = %+v, want observe event for observation", audit)
	}
}

func TestObservationsPublicFacadeServiceWrappersDefaultWorkspaceOnly(t *testing.T) {
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
