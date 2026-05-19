package goncho

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/TrebuchetDynamics/goncho/internal/schema"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := schema.RunMigrations(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestStoreMemory_InsertsAndRetrieves(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	result, err := StoreMemory(ctx, db, StoreParams{
		Content:     "User prefers SQLite over Postgres",
		Kind:        KindPreference,
		PeerID:      "telegram:123",
		WorkspaceID: "ws-1",
		Scope:       ScopePrivate,
		ContextID:   "project-x",
		Importance:  0.9,
	})
	if err != nil {
		t.Fatalf("StoreMemory: %v", err)
	}
	if result.Memory.ID == "" || result.Memory.Content != "User prefers SQLite over Postgres" {
		t.Fatalf("memory = %+v, want stored content", result.Memory)
	}
	if result.Memory.Kind != KindPreference || result.Memory.Importance != 0.9 {
		t.Fatalf("kind/importance = %v/%v, want preference/0.9", result.Memory.Kind, result.Memory.Importance)
	}
	if len(result.Memory.Checksum) != 64 {
		t.Fatalf("checksum len = %d, want 64", len(result.Memory.Checksum))
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM memories WHERE id = ?", result.Memory.ID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestStoreMemory_ValidatesContent(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	_, err := StoreMemory(ctx, db, StoreParams{Content: "", Kind: KindFact})
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestStoreMemory_ValidatesKind(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	_, err := StoreMemory(ctx, db, StoreParams{Content: "test", Kind: ""})
	if err == nil {
		t.Fatal("expected error for empty kind")
	}
}

func TestStoreMemory_ExtractsRelations(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	result, err := StoreMemory(ctx, db, StoreParams{
		Content: "Alice prefers PostgreSQL and uses Docker for deployment",
		Kind:    KindFact,
	})
	if err != nil {
		t.Fatalf("StoreMemory: %v", err)
	}

	var relCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM memory_relations WHERE source_id = ?", result.Memory.ID).Scan(&relCount); err != nil {
		t.Fatalf("count relations: %v", err)
	}
	if relCount == 0 {
		t.Fatal("expected relations to be extracted")
	}
}

func TestUpdateMemory_AppendsAndSupersedes(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	store, err := StoreMemory(ctx, db, StoreParams{
		Content: "User prefers SQLite",
		Kind:    KindPreference,
		PeerID:  "telegram:123",
	})
	if err != nil {
		t.Fatalf("StoreMemory: %v", err)
	}

	result, err := UpdateMemory(ctx, db, UpdateParams{
		ID:      store.Memory.ID,
		Content: "User now prefers Postgres for production",
		Reason:  "changed mind",
	})
	if err != nil {
		t.Fatalf("UpdateMemory: %v", err)
	}
	if result.Memory.Content != "User now prefers Postgres for production" {
		t.Fatalf("new content = %q, want Postgres preference", result.Memory.Content)
	}
	if result.Supersede.ID != store.Memory.ID {
		t.Fatalf("supersede id = %q, want %q", result.Supersede.ID, store.Memory.ID)
	}
	if result.Supersede.ValidUntil.IsZero() {
		t.Fatal("old memory should have valid_until set")
	}

	var activeCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM memories WHERE id = ? AND valid_until IS NULL", store.Memory.ID).Scan(&activeCount); err != nil {
		t.Fatalf("count: %v", err)
	}
	if activeCount != 0 {
		t.Fatal("old memory should be superseded (valid_until set)")
	}
}

func TestUpdateMemory_RejectsNotFound(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	_, err := UpdateMemory(ctx, db, UpdateParams{
		ID:      "nonexistent",
		Content: "test",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent memory")
	}
}

func TestForgetMemory_SoftDeletes(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	store, err := StoreMemory(ctx, db, StoreParams{
		Content: "Temporary fact",
		Kind:    KindFact,
	})
	if err != nil {
		t.Fatalf("StoreMemory: %v", err)
	}

	if err := ForgetMemory(ctx, db, store.Memory.ID, ForgetParams{Reason: "expired"}); err != nil {
		t.Fatalf("ForgetMemory: %v", err)
	}

	var validUntil sql.NullInt64
	if err := db.QueryRow("SELECT valid_until FROM memories WHERE id = ?", store.Memory.ID).Scan(&validUntil); err != nil {
		t.Fatalf("query: %v", err)
	}
	if !validUntil.Valid {
		t.Fatal("expected valid_until to be set after forget")
	}
}

func TestForgetMemory_RejectsNotFound(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	err := ForgetMemory(ctx, db, "nonexistent", ForgetParams{})
	if err == nil {
		t.Fatal("expected error for nonexistent memory")
	}
}

func TestMemory_IsExpired(t *testing.T) {
	now := time.Now()

	m := Memory{ValidUntil: now.Add(-time.Hour)}
	if !m.IsExpired(now) {
		t.Fatal("expected expired")
	}

	m = Memory{ValidUntil: now.Add(time.Hour)}
	if m.IsExpired(now) {
		t.Fatal("expected not expired")
	}

	m = Memory{}
	if m.IsExpired(now) {
		t.Fatal("expected not expired (zero valid_until)")
	}
}

func TestFTSIndex_Populated(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	_, err := StoreMemory(ctx, db, StoreParams{
		Content: "The quick brown fox jumps over the lazy dog",
		Kind:    KindFact,
	})
	if err != nil {
		t.Fatalf("StoreMemory: %v", err)
	}

	var ftsCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM memory_fts").Scan(&ftsCount); err != nil {
		t.Fatalf("fts count: %v", err)
	}
	if ftsCount != 1 {
		t.Fatalf("fts count = %d, want 1", ftsCount)
	}
}

func TestStoreMemory_Defaults(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	result, err := StoreMemory(ctx, db, StoreParams{
		Content: "test",
		Kind:    KindFact,
	})
	if err != nil {
		t.Fatalf("StoreMemory: %v", err)
	}
	if result.Memory.Scope != ScopePrivate {
		t.Fatalf("scope = %q, want %q", result.Memory.Scope, ScopePrivate)
	}
	if result.Memory.Importance != 0.5 {
		t.Fatalf("importance = %v, want 0.5", result.Memory.Importance)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
