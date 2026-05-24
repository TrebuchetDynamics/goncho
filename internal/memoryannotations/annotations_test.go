package memoryannotations

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestConclusionFactsExtractsAndDeduplicatesFactKinds(t *testing.T) {
	facts := ConclusionFacts("Alex's owner is Priya. The cache latency is 42 ms. Runtime uses SQLite. Alex's owner is Priya.")

	want := []string{
		"Priya owns Alex",
		"cache latency is 42 ms",
		"Runtime uses SQLite",
	}
	if len(facts) != len(want) {
		t.Fatalf("facts = %#v, want %#v", facts, want)
	}
	for i := range want {
		if facts[i] != want[i] {
			t.Fatalf("facts[%d] = %q, want %q (all facts %#v)", i, facts[i], want[i], facts)
		}
	}
}

func TestStoreAndQueryConclusionFactsByMemoryID(t *testing.T) {
	db := migratedAnnotationTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `INSERT INTO goncho_conclusions(id) VALUES(42)`); err != nil {
		t.Fatalf("insert conclusion: %v", err)
	}
	if err := StoreConclusionFacts(ctx, db, "workspace-a", "", "assistant", "user", 42, []string{"Priya owns Alex", "", "Priya owns Alex"}); err != nil {
		t.Fatalf("StoreConclusionFacts: %v", err)
	}
	byID, err := ConclusionFactsByMemoryID(ctx, db, []int64{42, 42})
	if err != nil {
		t.Fatalf("ConclusionFactsByMemoryID: %v", err)
	}
	facts := byID[42]
	if len(facts) != 1 {
		t.Fatalf("facts for 42 = %#v, want one deduped fact", facts)
	}
	if facts[0].MemorySource != SourceConclusion || facts[0].MemoryID != 42 || facts[0].Value != "Priya owns Alex" || facts[0].Confidence != 0.8 {
		t.Fatalf("fact = %#v, want stored conclusion fact", facts[0])
	}
}

func migratedAnnotationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", t.TempDir()+"/annotations.db")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE goncho_conclusions(id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create conclusions table: %v", err)
	}
	for _, ddl := range DDL {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			t.Fatalf("migrate annotation DDL: %v", err)
		}
	}
	return db
}
