package goncho

import (
	"context"
	"strconv"
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
	factEvidence, ok := recallCandidateFactEvidence(selected.Candidate, "Mira prefers tabs for indentation")
	if !ok {
		t.Fatalf("candidate provenance = %+v, want durable fact annotation provenance", selected.Candidate.Provenance)
	}

	var annotationID int64
	var annotationSource string
	var annotationConfidence float64
	if err := svc.db.QueryRowContext(ctx, `
		SELECT id, source, confidence
		FROM goncho_memory_annotations
		WHERE memory_source = 'conclusion'
		  AND memory_id = ?
		  AND kind = 'fact'
		  AND value = 'Mira prefers tabs for indentation'
	`, answer.ID).Scan(&annotationID, &annotationSource, &annotationConfidence); err != nil {
		t.Fatalf("lookup annotation citation: %v", err)
	}
	if factEvidence.ID != strconv.FormatInt(annotationID, 10) {
		t.Fatalf("fact evidence ID = %q, want durable annotation row id %d", factEvidence.ID, annotationID)
	}
	wantMetadata := map[string]string{
		"memory_source": "conclusion",
		"memory_id":     strconv.FormatInt(answer.ID, 10),
		"source":        annotationSource,
		"confidence":    "0.800",
	}
	for key, want := range wantMetadata {
		if got := factEvidence.Metadata[key]; got != want {
			t.Fatalf("fact evidence metadata[%q] = %q, want %q in %+v", key, got, want, factEvidence.Metadata)
		}
	}
	if annotationConfidence != 0.8 {
		t.Fatalf("annotation confidence = %v, want deterministic extractor confidence", annotationConfidence)
	}
}

func recallCandidateFactEvidence(candidate RecallCandidate, fact string) (EvidenceItem, bool) {
	return evidenceListFindKindSourceScoreNoteContains(candidate.Provenance, "fact", "goncho_memory_annotations", 1, "fact="+fact)
}
