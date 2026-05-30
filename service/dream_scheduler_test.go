package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	memory "github.com/TrebuchetDynamics/goncho/memory"
)

func TestGonchoDreamPublicFacadeSchedulesAndCancelsViaService(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	svc, cleanup := newDreamTestService(t, Config{
		DreamEnabled:     true,
		DreamIdleTimeout: time.Hour,
	})
	defer cleanup()

	seedDreamConclusions(t, svc.db, svc.workspaceID, svc.observer, "user-facade", 50, now.Add(-2*time.Hour))
	created, err := svc.ScheduleDream(ctx, DreamScheduleParams{Peer: "user-facade", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if created.Action != "created" || created.Status != "pending" || created.ID == 0 || created.Evidence.Code != "dream_pending" {
		t.Fatalf("ScheduleDream = %+v, want created pending dream intent", created)
	}

	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "user-facade", Conclusion: "new activity cancels stale dream"}); err != nil {
		t.Fatal(err)
	}
	if got := countDreamsByStatus(t, svc.db, "user-facade", "stale"); got != 1 {
		t.Fatalf("stale dream count = %d, want 1", got)
	}
	if status := dreamStatusByID(t, svc.db, created.ID); status != "stale" {
		t.Fatalf("dream %d status = %q, want stale", created.ID, status)
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

func countDreamsByStatus(t *testing.T, db *sql.DB, peer, status string) int {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM goncho_dreams WHERE observed_peer_id = ? AND status = ?`, peer, status).Scan(&got); err != nil {
		t.Fatalf("count dreams by status: %v", err)
	}
	return got
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
	return contextUnavailableHasCapability(items, capability)
}
