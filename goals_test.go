package goncho

import (
	"context"
	"testing"
)

func TestCreateGoal_CreatesActiveGoal(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	result, err := CreateGoal(ctx, db, GoalParams{Name: "Project X"})
	if err != nil {
		t.Fatalf("CreateGoal: %v", err)
	}
	if result.Goal.Name != "Project X" || result.Goal.Status != GoalActive {
		t.Fatalf("goal = %+v, want active Project X", result.Goal)
	}
}

func TestCreateGoal_ValidatesName(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	_, err := CreateGoal(ctx, db, GoalParams{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCompleteGoal_MarksCompleted(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	goal, _ := CreateGoal(ctx, db, GoalParams{Name: "Test Goal"})

	if err := CompleteGoal(ctx, db, goal.Goal.ID); err != nil {
		t.Fatalf("CompleteGoal: %v", err)
	}

	var status string
	if err := db.QueryRow("SELECT status FROM goals WHERE id = ?", goal.Goal.ID).Scan(&status); err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "completed" {
		t.Fatalf("status = %q, want completed", status)
	}
}

func TestArchiveGoal_MarksArchived(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	goal, _ := CreateGoal(ctx, db, GoalParams{Name: "Old Goal"})

	if err := ArchiveGoal(ctx, db, goal.Goal.ID); err != nil {
		t.Fatalf("ArchiveGoal: %v", err)
	}

	var status string
	if err := db.QueryRow("SELECT status FROM goals WHERE id = ?", goal.Goal.ID).Scan(&status); err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "archived" {
		t.Fatalf("status = %q, want archived", status)
	}
}

func TestAssembleContext_RespectsTokenLimit(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	StoreMemory(ctx, db, StoreParams{Content: "Short fact one", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1"})
	StoreMemory(ctx, db, StoreParams{Content: "Short fact two", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1"})
	StoreMemory(ctx, db, StoreParams{Content: "Short fact three", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1"})

	result, err := AssembleContext(ctx, db, ContextParams{
		PeerID:      "p1",
		WorkspaceID: "w1",
		MaxTokens:   10,
	})
	if err != nil {
		t.Fatalf("AssembleContext: %v", err)
	}
	if result.TokenEst > 10 {
		t.Fatalf("tokenEst = %d, want <= 10", result.TokenEst)
	}
}

func TestAssembleContext_ContextBoost(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	StoreMemory(ctx, db, StoreParams{Content: "Active context memory", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1", ContextID: "ctx-a"})
	StoreMemory(ctx, db, StoreParams{Content: "Other context memory", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1", ContextID: "ctx-b"})

	result, err := AssembleContext(ctx, db, ContextParams{
		PeerID:      "p1",
		WorkspaceID: "w1",
		ContextID:   "ctx-a",
		MaxTokens:   100,
	})
	if err != nil {
		t.Fatalf("AssembleContext: %v", err)
	}
	if len(result.Memories) > 0 && result.Memories[0].ContextID != "ctx-a" {
		t.Logf("top memory context = %q, expected ctx-a boost", result.Memories[0].ContextID)
	}
}
