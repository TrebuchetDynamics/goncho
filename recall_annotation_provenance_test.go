package goncho

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRecallCandidatesIncludeDurableFactAnnotationProvenance(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Mira's indentation preference is tabs.",
		SessionKey: "sess-recall-annotation",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "What indentation does Mira prefer? indentation prefer indentation prefer indentation prefer. This checklist repeats the retrieval words but does not answer it.",
		SessionKey: "sess-recall-annotation",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?)
	`, decoy.ID, answer.ID, decoy.ID, answer.ID); err != nil {
		t.Fatalf("force decoy recency: %v", err)
	}

	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: "annotation-provenance-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "annotation-provenance-test-v1",
			Weights:     map[string]float64{"keyword": 0.35, "fact": 0.55, "scope": 0.10},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 120,
		},
		now: func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        "team",
		Query:       "What indentation does Mira prefer?",
		SessionKey:  "sess-recall-annotation",
		ScopeID:     MemoryScopeWorkspace,
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.Candidates) < 2 {
		t.Fatalf("candidates = %+v, want annotated answer and lexical decoy before selection", trace.Candidates)
	}
	if len(trace.Selected) != 1 {
		t.Fatalf("selected = %+v candidates=%+v rejected=%+v, want one annotated candidate", trace.Selected, trace.Candidates, trace.Rejected)
	}
	selected := trace.Selected[0]
	if selected.Candidate.MemoryID != strconv.FormatInt(answer.ID, 10) {
		t.Fatalf("selected memory = %+v, want durable annotated conclusion %d", selected.Candidate, answer.ID)
	}
	if selected.Score.FactScore != 1 {
		t.Fatalf("fact score = %v, want durable annotation to feed recall scoring", selected.Score.FactScore)
	}
	if !recallCandidateHasFactAnnotation(selected.Candidate, "Mira prefers tabs for indentation") {
		t.Fatalf("candidate provenance = %+v, want durable fact annotation provenance", selected.Candidate.Provenance)
	}
}

func recallCandidateHasFactAnnotation(candidate RecallCandidate, fact string) bool {
	for _, item := range candidate.Provenance {
		if item.Kind != "fact" || item.Source != "goncho_memory_annotations" || item.Score != 1 {
			continue
		}
		if strings.Contains(item.Note, "fact="+fact) {
			return true
		}
	}
	return false
}
