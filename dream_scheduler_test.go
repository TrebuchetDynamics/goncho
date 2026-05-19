package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	memory "github.com/TrebuchetDynamics/goncho/memory"
)

func TestGonchoDreamSchedulerRequiresThresholdCooldownAndIdle(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	svc, cleanup := newDreamTestService(t, Config{
		DreamEnabled:     true,
		DreamIdleTimeout: time.Hour,
	})
	defer cleanup()

	seedDreamConclusions(t, svc.db, svc.workspaceID, svc.observer, "user-threshold", 49, now.Add(-2*time.Hour))
	got, err := svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-threshold", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != "rejected" || got.Evidence.Code != "dream_threshold" || got.NewConclusions != 49 {
		t.Fatalf("threshold result = %+v, want rejected dream_threshold with 49 new conclusions", got)
	}

	seedDreamConclusions(t, svc.db, svc.workspaceID, svc.observer, "user-idle", 49, now.Add(-2*time.Hour))
	seedDreamConclusions(t, svc.db, svc.workspaceID, svc.observer, "user-idle", 1, now.Add(-30*time.Minute))
	got, err = svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-idle", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != "rejected" || got.Evidence.Code != "dream_idle" {
		t.Fatalf("idle result = %+v, want rejected dream_idle", got)
	}
	wantIdleUntil := now.Add(30 * time.Minute).Unix()
	if got.Evidence.IdleUntil != wantIdleUntil {
		t.Fatalf("IdleUntil = %d, want %d", got.Evidence.IdleUntil, wantIdleUntil)
	}

	seedDreamConclusions(t, svc.db, svc.workspaceID, svc.observer, "user-ready", 50, now.Add(-2*time.Hour))
	got, err = svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-ready", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != "created" || got.Status != "pending" || got.ID == 0 {
		t.Fatalf("eligible result = %+v, want created pending dream intent", got)
	}

	seedDreamConclusions(t, svc.db, svc.workspaceID, svc.observer, "user-cooldown", 100, now.Add(-2*time.Hour))
	insertDreamIntentRow(t, svc.db, dreamIntentSeed{
		WorkspaceID:      svc.workspaceID,
		ObserverPeerID:   svc.observer,
		ObservedPeerID:   "user-cooldown",
		Status:           "completed",
		LastConclusionID: 50,
		NewConclusions:   50,
		CompletedAt:      now.Add(-7 * time.Hour).Unix(),
		CooldownUntil:    now.Add(time.Hour).Unix(),
		CreatedAt:        now.Add(-7 * time.Hour).Unix(),
		UpdatedAt:        now.Add(-7 * time.Hour).Unix(),
	})
	got, err = svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-cooldown", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != "rejected" || got.Evidence.Code != "dream_cooldown" {
		t.Fatalf("cooldown result = %+v, want rejected dream_cooldown", got)
	}
	if got.Evidence.CooldownUntil != now.Add(time.Hour).Unix() {
		t.Fatalf("CooldownUntil = %d, want %d", got.Evidence.CooldownUntil, now.Add(time.Hour).Unix())
	}
}

func TestGonchoDreamSchedulerDedupesActiveIntentAndStalesPendingOnNewActivity(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	svc, cleanup := newDreamTestService(t, Config{
		DreamEnabled:     true,
		DreamIdleTimeout: time.Hour,
	})
	defer cleanup()

	seedDreamConclusions(t, svc.db, svc.workspaceID, svc.observer, "user-active", 50, now.Add(-2*time.Hour))
	created, err := svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-active", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	reused, err := svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-active", Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if reused.Action != "reused" || reused.ID != created.ID || reused.Evidence.Code != "dream_pending" {
		t.Fatalf("second schedule = %+v, want reused pending dream %d", reused, created.ID)
	}
	if got := countDreamsByStatus(t, svc.db, "user-active", "pending") + countDreamsByStatus(t, svc.db, "user-active", "in_progress"); got != 1 {
		t.Fatalf("active dream count = %d, want 1", got)
	}

	setDreamStatus(t, svc.db, created.ID, "in_progress", now.Add(2*time.Minute).Unix())
	reused, err = svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-active", Now: now.Add(3 * time.Minute), Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	if reused.Action != "reused" || reused.ID != created.ID || reused.Evidence.Code != "dream_in_progress" {
		t.Fatalf("manual during in-progress = %+v, want reused dream_in_progress", reused)
	}

	seedDreamConclusions(t, svc.db, svc.workspaceID, svc.observer, "user-stale", 50, now.Add(-2*time.Hour))
	pending, err := svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-stale", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "user-stale", Conclusion: "new activity cancels stale dream"}); err != nil {
		t.Fatal(err)
	}
	if got := countDreamsByStatus(t, svc.db, "user-stale", "stale"); got != 1 {
		t.Fatalf("stale dream count = %d, want 1", got)
	}
	if got := countDreamsByStatus(t, svc.db, "user-stale", "pending"); got != 0 {
		t.Fatalf("pending dream count after new activity = %d, want 0", got)
	}
	if got := countDreamsForPeer(t, svc.db, "user-stale"); got != 1 {
		t.Fatalf("dream history count = %d, want stale history preserved", got)
	}
	if status := dreamStatusByID(t, svc.db, pending.ID); status != "stale" {
		t.Fatalf("dream %d status = %q, want stale", pending.ID, status)
	}
}

func TestGonchoDreamManualScheduleReportsCreatedReusedAndRejected(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	svc, cleanup := newDreamTestService(t, Config{DreamEnabled: true})
	defer cleanup()

	created, err := svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-manual", Now: now, Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	if created.Action != "created" || created.ID == 0 || created.Evidence.Code != "dream_pending" {
		t.Fatalf("manual created = %+v, want created pending evidence", created)
	}

	reused, err := svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-manual", Now: now.Add(time.Minute), Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	if reused.Action != "reused" || reused.ID != created.ID || reused.Evidence.Code != "dream_pending" {
		t.Fatalf("manual reused = %+v, want reused pending dream %d", reused, created.ID)
	}

	disabled, disabledCleanup := newDreamTestService(t, Config{DreamEnabled: false})
	defer disabledCleanup()
	rejected, err := disabled.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-manual", Now: now, Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Action != "rejected" || rejected.Evidence.Code != "dream_disabled" {
		t.Fatalf("manual disabled = %+v, want rejected dream_disabled", rejected)
	}

	dropDreamTable(t, svc.db)
	rejected, err = svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-manual", Now: now, Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Action != "rejected" || rejected.Evidence.Code != "dream_unavailable" {
		t.Fatalf("manual unavailable = %+v, want rejected dream_unavailable", rejected)
	}
}

func TestGonchoDreamContextReportsDisabledAndUnavailableEvidence(t *testing.T) {
	ctx := context.Background()
	disabled, disabledCleanup := newDreamTestService(t, Config{DreamEnabled: false})
	defer disabledCleanup()
	includeDreamStatus := true

	got, err := disabled.Context(ctx, ContextParams{Peer: "user-context", IncludeDreamStatus: &includeDreamStatus})
	if err != nil {
		t.Fatal(err)
	}
	if !contextHasCapability(got.Unavailable, "dream_disabled") {
		t.Fatalf("Context unavailable = %+v, want dream_disabled evidence", got.Unavailable)
	}

	enabled, enabledCleanup := newDreamTestService(t, Config{DreamEnabled: true})
	defer enabledCleanup()
	dropDreamTable(t, enabled.db)
	got, err = enabled.Context(ctx, ContextParams{Peer: "user-context", IncludeDreamStatus: &includeDreamStatus})
	if err != nil {
		t.Fatal(err)
	}
	if !contextHasCapability(got.Unavailable, "dream_unavailable") {
		t.Fatalf("Context unavailable = %+v, want dream_unavailable evidence", got.Unavailable)
	}
}

func TestGonchoDreamQueueStatusReportsDreamEvidenceWithoutWaitingForEmptyQueue(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	svc, cleanup := newDreamTestService(t, Config{DreamEnabled: true})
	defer cleanup()

	disabled, err := ReadQueueStatus(ctx, svc.db, QueueStatusConfig{DreamEnabled: false, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Dream.Status != "dream_disabled" || !dreamEvidenceHasCode(disabled.Dream.Evidence, "dream_disabled") {
		t.Fatalf("disabled queue status = %+v, want dream_disabled evidence", disabled.Dream)
	}

	insertDreamIntentRow(t, svc.db, dreamIntentSeed{
		WorkspaceID:    svc.workspaceID,
		ObserverPeerID: svc.observer,
		ObservedPeerID: "user-pending",
		Status:         "pending",
		NewConclusions: 50,
		CreatedAt:      now.Add(-time.Hour).Unix(),
		UpdatedAt:      now.Add(-time.Hour).Unix(),
	})
	insertDreamIntentRow(t, svc.db, dreamIntentSeed{
		WorkspaceID:    svc.workspaceID,
		ObserverPeerID: svc.observer,
		ObservedPeerID: "user-running",
		Status:         "in_progress",
		NewConclusions: 50,
		CreatedAt:      now.Add(-time.Hour).Unix(),
		UpdatedAt:      now.Add(-time.Hour).Unix(),
	})
	insertDreamIntentRow(t, svc.db, dreamIntentSeed{
		WorkspaceID:      svc.workspaceID,
		ObserverPeerID:   svc.observer,
		ObservedPeerID:   "user-cooldown",
		Status:           "completed",
		NewConclusions:   50,
		CompletedAt:      now.Add(-2 * time.Hour).Unix(),
		CooldownUntil:    now.Add(6 * time.Hour).Unix(),
		LastConclusionID: 50,
		CreatedAt:        now.Add(-2 * time.Hour).Unix(),
		UpdatedAt:        now.Add(-2 * time.Hour).Unix(),
	})

	status, err := ReadQueueStatus(ctx, svc.db, QueueStatusConfig{
		DreamEnabled:     true,
		WorkspaceID:      svc.workspaceID,
		ObserverPeerID:   svc.observer,
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

	dropDreamTable(t, svc.db)
	unavailable, err := ReadQueueStatus(ctx, svc.db, QueueStatusConfig{DreamEnabled: true, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if unavailable.Dream.Status != "dream_unavailable" || !dreamEvidenceHasCode(unavailable.Dream.Evidence, "dream_unavailable") {
		t.Fatalf("unavailable queue status = %+v, want dream_unavailable evidence", unavailable.Dream)
	}
}

func newDreamTestService(t *testing.T, cfg Config) (*Service, func()) {
	t.Helper()

	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	cfg.WorkspaceID = "default"
	cfg.ObserverPeerID = "gormes"
	svc := NewService(store.DB(), cfg, nil)
	return svc, func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}

func seedDreamConclusions(t *testing.T, db *sql.DB, workspaceID, observer, peer string, count int, createdAt time.Time) {
	t.Helper()
	for i := 0; i < count; i++ {
		_, err := db.Exec(`
			INSERT INTO goncho_conclusions(
				workspace_id, observer_peer_id, peer_id, session_key, content,
				kind, status, source, idempotency_key, evidence_json, created_at, updated_at
			)
			VALUES(?, ?, ?, NULL, ?, 'manual', 'processed', 'test', ?, '[]', ?, ?)
		`,
			workspaceID,
			observer,
			peer,
			"dream conclusion",
			strings.Join([]string{peer, createdAt.Format(time.RFC3339Nano), time.Now().Format(time.RFC3339Nano), fmt.Sprint(i)}, ":"),
			createdAt.Add(time.Duration(i)*time.Second).Unix(),
			createdAt.Add(time.Duration(i)*time.Second).Unix(),
		)
		if err != nil {
			t.Fatalf("seed conclusion %d: %v", i, err)
		}
	}
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
	workUnitKey := "dream:consolidation:" + row.WorkspaceID + ":" + row.ObserverPeerID + ":" + row.ObservedPeerID
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

func countDreamsByStatus(t *testing.T, db *sql.DB, peer, status string) int {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM goncho_dreams WHERE observed_peer_id = ? AND status = ?`, peer, status).Scan(&got); err != nil {
		t.Fatalf("count dreams by status: %v", err)
	}
	return got
}

func countDreamsForPeer(t *testing.T, db *sql.DB, peer string) int {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM goncho_dreams WHERE observed_peer_id = ?`, peer).Scan(&got); err != nil {
		t.Fatalf("count dreams for peer: %v", err)
	}
	return got
}

func setDreamStatus(t *testing.T, db *sql.DB, id int64, status string, updatedAt int64) {
	t.Helper()
	if _, err := db.Exec(`UPDATE goncho_dreams SET status = ?, updated_at = ?, started_at = ? WHERE id = ?`, status, updatedAt, updatedAt, id); err != nil {
		t.Fatalf("set dream status: %v", err)
	}
}

func dreamStatusByID(t *testing.T, db *sql.DB, id int64) string {
	t.Helper()
	var status string
	if err := db.QueryRow(`SELECT status FROM goncho_dreams WHERE id = ?`, id).Scan(&status); err != nil {
		t.Fatalf("dream status by id: %v", err)
	}
	return status
}

func dropDreamTable(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`DROP TABLE goncho_dreams`); err != nil {
		t.Fatalf("drop goncho_dreams: %v", err)
	}
}

func contextHasCapability(items []ContextUnavailableEvidence, capability string) bool {
	return slices.ContainsFunc(items, func(item ContextUnavailableEvidence) bool {
		return item.Capability == capability
	})
}

func dreamEvidenceHasCode(items []DreamStatusEvidence, code string) bool {
	return slices.ContainsFunc(items, func(item DreamStatusEvidence) bool {
		return item.Code == code
	})
}
