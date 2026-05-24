package queuestatus

import (
	"context"
	"database/sql"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/dreamscheduler"
	memory "github.com/TrebuchetDynamics/goncho/memory"
)

func TestReadZeroStateIsDeterministicObservabilityOnly(t *testing.T) {
	db, cleanup := newQueueStatusTestDB(t)
	defer cleanup()
	defaults := queueStatusTestDefaults()

	first, err := Read(context.Background(), db, defaults)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Read(context.Background(), db, defaults)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("Read returned nondeterministic zero-state:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if first.Status != "degraded" || !first.Degraded {
		t.Fatalf("status = %q degraded=%t, want degraded zero-state", first.Status, first.Degraded)
	}
	if !first.ObservabilityOnly {
		t.Fatal("ObservabilityOnly = false, want true")
	}
	if !stringsContainsAll(first.Message, "zero tracked work units", "observability", "do not wait") {
		t.Fatalf("Message = %q, want zero-state observability warning", first.Message)
	}

	for _, taskType := range TaskTypes {
		counts, ok := first.WorkUnits[taskType]
		if !ok {
			t.Fatalf("WorkUnits missing task type %q: %#v", taskType, first.WorkUnits)
		}
		if counts.CompletedWorkUnits != 0 || counts.InProgressWorkUnits != 0 || counts.PendingWorkUnits != 0 || counts.TotalWorkUnits != 0 {
			t.Fatalf("%s counts = %+v, want deterministic zero-state", taskType, counts)
		}
		if len(counts.Sessions) != 0 {
			t.Fatalf("%s sessions = %+v, want no per-session details before a Goncho task queue exists", taskType, counts.Sessions)
		}
	}
}

func TestOnlyReportsHonchoReasoningWorkTypes(t *testing.T) {
	want := []string{"representation", "summary", "dream"}
	if !slices.Equal(TaskTypes, want) {
		t.Fatalf("TaskTypes = %#v, want %#v", TaskTypes, want)
	}

	status := Zero(queueStatusTestDefaults())
	if len(status.WorkUnits) != len(want) {
		t.Fatalf("WorkUnits len = %d, want only %d Honcho reasoning task types: %#v", len(status.WorkUnits), len(want), status.WorkUnits)
	}
	for _, internalTask := range []string{"webhook", "deletion", "vector_reconciliation", "reconciler"} {
		if _, ok := status.WorkUnits[internalTask]; ok {
			t.Fatalf("WorkUnits included internal infrastructure task %q: %#v", internalTask, status.WorkUnits)
		}
	}
}

func TestDreamStatusReportsEvidenceWithoutWaitingForEmptyQueue(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	db, cleanup := newQueueStatusTestDB(t)
	defer cleanup()
	defaults := queueStatusTestDefaults()

	disabled, err := Read(ctx, db, defaults, dreamscheduler.QueueStatusConfig{DreamEnabled: false, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Dream.Status != "dream_disabled" || !dreamEvidenceHasCode(disabled.Dream.Evidence, "dream_disabled") {
		t.Fatalf("disabled queue status = %+v, want dream_disabled evidence", disabled.Dream)
	}

	insertDreamIntentRow(t, db, dreamIntentSeed{
		WorkspaceID:    defaults.WorkspaceID,
		ObserverPeerID: defaults.ObserverPeerID,
		ObservedPeerID: "user-pending",
		Status:         "pending",
		NewConclusions: 50,
		CreatedAt:      now.Add(-time.Hour).Unix(),
		UpdatedAt:      now.Add(-time.Hour).Unix(),
	})
	insertDreamIntentRow(t, db, dreamIntentSeed{
		WorkspaceID:    defaults.WorkspaceID,
		ObserverPeerID: defaults.ObserverPeerID,
		ObservedPeerID: "user-running",
		Status:         "in_progress",
		NewConclusions: 50,
		CreatedAt:      now.Add(-time.Hour).Unix(),
		UpdatedAt:      now.Add(-time.Hour).Unix(),
	})
	insertDreamIntentRow(t, db, dreamIntentSeed{
		WorkspaceID:      defaults.WorkspaceID,
		ObserverPeerID:   defaults.ObserverPeerID,
		ObservedPeerID:   "user-cooldown",
		Status:           "completed",
		NewConclusions:   50,
		CompletedAt:      now.Add(-2 * time.Hour).Unix(),
		CooldownUntil:    now.Add(6 * time.Hour).Unix(),
		LastConclusionID: 50,
		CreatedAt:        now.Add(-2 * time.Hour).Unix(),
		UpdatedAt:        now.Add(-2 * time.Hour).Unix(),
	})

	status, err := Read(ctx, db, defaults, dreamscheduler.QueueStatusConfig{
		DreamEnabled:     true,
		WorkspaceID:      defaults.WorkspaceID,
		ObserverPeerID:   defaults.ObserverPeerID,
		Now:              now,
		DreamIdleTimeout: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	dreamCounts := status.WorkUnits["dream"]
	if dreamCounts.PendingWorkUnits != 1 || dreamCounts.InProgressWorkUnits != 1 || dreamCounts.CompletedWorkUnits != 1 || dreamCounts.TotalWorkUnits != 3 {
		t.Fatalf("dream work counts = %+v, want 1 pending, 1 in-progress, 1 completed, 3 total", dreamCounts)
	}
	for _, code := range []string{"dream_pending", "dream_in_progress", "dream_cooldown"} {
		if !dreamEvidenceHasCode(status.Dream.Evidence, code) {
			t.Fatalf("dream evidence missing %s: %+v", code, status.Dream.Evidence)
		}
	}

	dropDreamTable(t, db)
	unavailable, err := Read(ctx, db, defaults, dreamscheduler.QueueStatusConfig{DreamEnabled: true, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if unavailable.Dream.Status != "dream_unavailable" || !dreamEvidenceHasCode(unavailable.Dream.Evidence, "dream_unavailable") {
		t.Fatalf("unavailable queue status = %+v, want dream_unavailable evidence", unavailable.Dream)
	}
}

func newQueueStatusTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	return store.DB(), func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}

func queueStatusTestDefaults() Defaults {
	return Defaults{WorkspaceID: "default", ObserverPeerID: "gormes", DreamIdleTimeout: time.Hour}
}

type dreamIntentSeed struct {
	WorkspaceID      string
	ObserverPeerID   string
	ObservedPeerID   string
	Status           string
	NewConclusions   int
	LastConclusionID int64
	CompletedAt      int64
	CooldownUntil    int64
	CreatedAt        int64
	UpdatedAt        int64
}

func insertDreamIntentRow(t *testing.T, db *sql.DB, row dreamIntentSeed) int64 {
	t.Helper()
	if row.Status == "" {
		row.Status = "pending"
	}
	if row.CreatedAt == 0 {
		row.CreatedAt = time.Now().Unix()
	}
	if row.UpdatedAt == 0 {
		row.UpdatedAt = row.CreatedAt
	}
	if row.NewConclusions == 0 {
		row.NewConclusions = 50
	}
	workUnitKey := dreamscheduler.WorkUnitKey(row.WorkspaceID, row.ObserverPeerID, row.ObservedPeerID)
	res, err := db.Exec(`
		INSERT INTO goncho_dreams(
			workspace_id, observer_peer_id, observed_peer_id, work_unit_key, dream_type,
			status, manual, reason, new_conclusions, min_conclusions, last_conclusion_id,
			scheduled_for, completed_at, cooldown_until, idle_until, last_activity_at,
			created_at, updated_at
		)
		VALUES(?, ?, ?, ?, 'consolidation', ?, 0, 'test seed', ?, 50, ?, ?, NULLIF(?, 0), ?, 0, ?, ?, ?)
	`,
		row.WorkspaceID,
		row.ObserverPeerID,
		row.ObservedPeerID,
		workUnitKey,
		row.Status,
		row.NewConclusions,
		row.LastConclusionID,
		row.CreatedAt,
		row.CompletedAt,
		row.CooldownUntil,
		row.CreatedAt,
		row.CreatedAt,
		row.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("insert dream intent: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	return id
}

func dropDreamTable(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`DROP TABLE goncho_dreams`); err != nil {
		t.Fatalf("drop goncho_dreams: %v", err)
	}
}

func dreamEvidenceHasCode(items []dreamscheduler.DreamStatusEvidence, code string) bool {
	return slices.ContainsFunc(items, func(item dreamscheduler.DreamStatusEvidence) bool {
		return item.Code == code
	})
}

func stringsContainsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}
