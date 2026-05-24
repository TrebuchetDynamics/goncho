package goncho

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"testing"
	"time"
)

func TestRecallExpandsOwnerThroughDurableKGRelation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	uses, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Billing API uses LedgerDB.",
		SessionKey: "sess-annotation-graph",
	})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Owner of LedgerDB is Mira.",
		SessionKey: "sess-annotation-graph",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Who is responsible for storage used by Billing API? responsible storage used Billing API responsible storage used Billing API. This checklist repeats the retrieval words but names no owner.",
		SessionKey: "sess-annotation-graph",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 300 WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?, ?)
	`, decoy.ID, uses.ID, owner.ID, decoy.ID, uses.ID, owner.ID); err != nil {
		t.Fatalf("force lexical decoy recency: %v", err)
	}

	usesFactID := lookupAnnotationID(t, svc, uses.ID, "Billing API uses LedgerDB")
	ownerFactID := lookupAnnotationID(t, svc, owner.ID, "Mira owns LedgerDB")

	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: "annotation-graph-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "annotation-graph-test-v1",
			Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 180,
		},
		now: func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        "team",
		Query:       "Who is responsible for storage used by Billing API?",
		SessionKey:  "sess-annotation-graph",
		ScopeID:     MemoryScopeWorkspace,
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}

	ownerMemoryID := strconv.FormatInt(owner.ID, 10)
	selected := selectedRecallIDs(trace)
	if !slices.Contains(selected, ownerMemoryID) {
		t.Fatalf("selected IDs = %v candidates=%+v rejected=%+v, want graph-expanded owner %s", selected, trace.Candidates, trace.Rejected, ownerMemoryID)
	}
	ownerCandidate, ok := selectedRecallCandidate(trace, ownerMemoryID)
	if !ok {
		t.Fatalf("selected = %+v, want owner candidate", trace.Selected)
	}
	evidenceID := fmt.Sprintf("annotation:%d->annotation:%d", usesFactID, ownerFactID)
	if !candidateHasGraphProvenance(ownerCandidate, evidenceID) {
		t.Fatalf("owner provenance = %+v, want graph evidence %s", ownerCandidate.Provenance, evidenceID)
	}
	wantNote := fmt.Sprintf("%d -> uses -> LedgerDB -> owned_by -> %d", uses.ID, owner.ID)
	if !candidateHasGraphNote(ownerCandidate, wantNote) {
		t.Fatalf("owner provenance = %+v, want relation path %q", ownerCandidate.Provenance, wantNote)
	}
}

func lookupAnnotationID(t *testing.T, svc *Service, memoryID int64, value string) int64 {
	t.Helper()
	var id int64
	if err := svc.db.QueryRowContext(context.Background(), `
		SELECT id
		FROM goncho_memory_annotations
		WHERE memory_source = 'conclusion'
		  AND memory_id = ?
		  AND kind = 'fact'
		  AND value = ?
	`, memoryID, value).Scan(&id); err != nil {
		t.Fatalf("lookup annotation %q for memory %d: %v", value, memoryID, err)
	}
	return id
}
