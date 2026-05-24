package dreamscheduler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	memory "github.com/TrebuchetDynamics/goncho/memory"
)

func TestSchedulerRequiresThresholdCooldownAndIdle(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	db, cleanup := newDreamSchedulerTestDB(t)
	defer cleanup()
	cfg := dreamSchedulerTestConfig(db, time.Hour)

	seedDreamConclusions(t, db, cfg.WorkspaceID, cfg.ObserverPeerID, "user-threshold", 49, now.Add(-2*time.Hour))
	got, err := Schedule(ctx, cfg, DreamScheduleParams{Peer: "user-threshold", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != "rejected" || got.Evidence.Code != "dream_threshold" || got.NewConclusions != 49 {
		t.Fatalf("threshold result = %+v, want rejected dream_threshold with 49 new conclusions", got)
	}

	seedDreamConclusions(t, db, cfg.WorkspaceID, cfg.ObserverPeerID, "user-idle", 49, now.Add(-2*time.Hour))
	seedDreamConclusions(t, db, cfg.WorkspaceID, cfg.ObserverPeerID, "user-idle", 1, now.Add(-30*time.Minute))
	got, err = Schedule(ctx, cfg, DreamScheduleParams{Peer: "user-idle", Now: now})
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

	seedDreamConclusions(t, db, cfg.WorkspaceID, cfg.ObserverPeerID, "user-ready", 50, now.Add(-2*time.Hour))
	got, err = Schedule(ctx, cfg, DreamScheduleParams{Peer: "user-ready", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != "created" || got.Status != "pending" || got.ID == 0 {
		t.Fatalf("eligible result = %+v, want created pending dream intent", got)
	}

	seedDreamConclusions(t, db, cfg.WorkspaceID, cfg.ObserverPeerID, "user-cooldown", 100, now.Add(-2*time.Hour))
	insertDreamIntentRow(t, db, dreamIntentSeed{
		WorkspaceID:      cfg.WorkspaceID,
		ObserverPeerID:   cfg.ObserverPeerID,
		ObservedPeerID:   "user-cooldown",
		Status:           "completed",
		LastConclusionID: 50,
		NewConclusions:   50,
		CompletedAt:      now.Add(-7 * time.Hour).Unix(),
		CooldownUntil:    now.Add(time.Hour).Unix(),
		CreatedAt:        now.Add(-7 * time.Hour).Unix(),
		UpdatedAt:        now.Add(-7 * time.Hour).Unix(),
	})
	got, err = Schedule(ctx, cfg, DreamScheduleParams{Peer: "user-cooldown", Now: now})
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

func TestSchedulerDedupesActiveIntentAndStalesPendingOnNewActivity(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	db, cleanup := newDreamSchedulerTestDB(t)
	defer cleanup()
	cfg := dreamSchedulerTestConfig(db, time.Hour)

	seedDreamConclusions(t, db, cfg.WorkspaceID, cfg.ObserverPeerID, "user-active", 50, now.Add(-2*time.Hour))
	created, err := Schedule(ctx, cfg, DreamScheduleParams{Peer: "user-active", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	reused, err := Schedule(ctx, cfg, DreamScheduleParams{Peer: "user-active", Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if reused.Action != "reused" || reused.ID != created.ID || reused.Evidence.Code != "dream_pending" {
		t.Fatalf("second schedule = %+v, want reused pending dream %d", reused, created.ID)
	}
	if got := countDreamsByStatus(t, db, "user-active", "pending") + countDreamsByStatus(t, db, "user-active", "in_progress"); got != 1 {
		t.Fatalf("active dream count = %d, want 1", got)
	}

	setDreamStatus(t, db, created.ID, "in_progress", now.Add(2*time.Minute).Unix())
	reused, err = Schedule(ctx, cfg, DreamScheduleParams{Peer: "user-active", Now: now.Add(3 * time.Minute), Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	if reused.Action != "reused" || reused.ID != created.ID || reused.Evidence.Code != "dream_in_progress" {
		t.Fatalf("manual during in-progress = %+v, want reused dream_in_progress", reused)
	}

	seedDreamConclusions(t, db, cfg.WorkspaceID, cfg.ObserverPeerID, "user-stale", 50, now.Add(-2*time.Hour))
	pending, err := Schedule(ctx, cfg, DreamScheduleParams{Peer: "user-stale", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CancelPendingForObserved(ctx, db, cfg.WorkspaceID, "user-stale", now.Add(time.Minute).Unix(), "new_activity"); err != nil {
		t.Fatal(err)
	}
	if got := countDreamsByStatus(t, db, "user-stale", "stale"); got != 1 {
		t.Fatalf("stale dream count = %d, want 1", got)
	}
	if got := countDreamsByStatus(t, db, "user-stale", "pending"); got != 0 {
		t.Fatalf("pending dream count after new activity = %d, want 0", got)
	}
	if got := countDreamsForPeer(t, db, "user-stale"); got != 1 {
		t.Fatalf("dream history count = %d, want stale history preserved", got)
	}
	if status := dreamStatusByID(t, db, pending.ID); status != "stale" {
		t.Fatalf("dream %d status = %q, want stale", pending.ID, status)
	}
}

func TestManualScheduleReportsCreatedReusedAndRejected(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	db, cleanup := newDreamSchedulerTestDB(t)
	defer cleanup()
	cfg := dreamSchedulerTestConfig(db, 0)

	created, err := Schedule(ctx, cfg, DreamScheduleParams{Peer: "user-manual", Now: now, Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	if created.Action != "created" || created.ID == 0 || created.Evidence.Code != "dream_pending" {
		t.Fatalf("manual created = %+v, want created pending evidence", created)
	}

	reused, err := Schedule(ctx, cfg, DreamScheduleParams{Peer: "user-manual", Now: now.Add(time.Minute), Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	if reused.Action != "reused" || reused.ID != created.ID || reused.Evidence.Code != "dream_pending" {
		t.Fatalf("manual reused = %+v, want reused pending dream %d", reused, created.ID)
	}

	disabledCfg := cfg
	disabledCfg.DreamEnabled = false
	rejected, err := Schedule(ctx, disabledCfg, DreamScheduleParams{Peer: "user-manual", Now: now, Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Action != "rejected" || rejected.Evidence.Code != "dream_disabled" {
		t.Fatalf("manual disabled = %+v, want rejected dream_disabled", rejected)
	}

	dropDreamTable(t, db)
	rejected, err = Schedule(ctx, cfg, DreamScheduleParams{Peer: "user-manual", Now: now, Manual: true})
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Action != "rejected" || rejected.Evidence.Code != "dream_unavailable" {
		t.Fatalf("manual unavailable = %+v, want rejected dream_unavailable", rejected)
	}
}

func newDreamSchedulerTestDB(t *testing.T) (*sql.DB, func()) {
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

func dreamSchedulerTestConfig(db *sql.DB, idle time.Duration) ScheduleConfig {
	return ScheduleConfig{
		DB:             db,
		WorkspaceID:    "default",
		ObserverPeerID: "gormes",
		DreamEnabled:   true,
		IdleTimeout:    idle,
		MinConclusions: DefaultMinConclusions,
		Cooldown:       DefaultCooldown,
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
	workUnitKey := WorkUnitKey(row.WorkspaceID, row.ObserverPeerID, row.ObservedPeerID)
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
