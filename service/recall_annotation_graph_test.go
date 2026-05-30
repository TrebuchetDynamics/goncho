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
	if !recallCandidateHasGraphProvenance(ownerCandidate, evidenceID) {
		t.Fatalf("owner provenance = %+v, want graph evidence %s", ownerCandidate.Provenance, evidenceID)
	}
	wantNote := fmt.Sprintf("%d -> uses -> LedgerDB -> owned_by -> %d", uses.ID, owner.ID)
	if !recallCandidateHasGraphNote(ownerCandidate, wantNote) {
		t.Fatalf("owner provenance = %+v, want relation path %q", ownerCandidate.Provenance, wantNote)
	}
}

func TestRecallExpandsVersionThroughMultiHopDurableKGRelation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	uses, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Billing API uses LedgerDB.",
		SessionKey: "sess-annotation-graph-version",
	})
	if err != nil {
		t.Fatal(err)
	}
	runs, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: LedgerDB runs on PostgreSQL.",
		SessionKey: "sess-annotation-graph-version",
	})
	if err != nil {
		t.Fatal(err)
	}
	version, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: PostgreSQL version is 14.2.",
		SessionKey: "sess-annotation-graph-version",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "What version is used by Billing API storage? version used Billing API storage version used Billing API storage. This checklist repeats the retrieval words but names no database version.",
		SessionKey: "sess-annotation-graph-version",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 400 WHEN ? THEN 300 WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?, ?, ?)
	`, decoy.ID, uses.ID, runs.ID, version.ID, decoy.ID, uses.ID, runs.ID, version.ID); err != nil {
		t.Fatalf("force lexical decoy recency: %v", err)
	}

	usesFactID := lookupAnnotationID(t, svc, uses.ID, "Billing API uses LedgerDB")
	runsFactID := lookupAnnotationID(t, svc, runs.ID, "LedgerDB runs on PostgreSQL")
	versionFactID := lookupAnnotationID(t, svc, version.ID, "PostgreSQL version is 14.2")

	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: "annotation-graph-version-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "annotation-graph-version-test-v1",
			Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 220,
		},
		now: func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        "team",
		Query:       "What version is used by Billing API storage?",
		SessionKey:  "sess-annotation-graph-version",
		ScopeID:     MemoryScopeWorkspace,
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}

	versionMemoryID := strconv.FormatInt(version.ID, 10)
	selected := selectedRecallIDs(trace)
	if !slices.Contains(selected, versionMemoryID) {
		t.Fatalf("selected IDs = %v candidates=%+v rejected=%+v, want graph-expanded version %s", selected, trace.Candidates, trace.Rejected, versionMemoryID)
	}
	versionCandidate, ok := selectedRecallCandidate(trace, versionMemoryID)
	if !ok {
		t.Fatalf("selected = %+v, want version candidate", trace.Selected)
	}
	evidenceID := fmt.Sprintf("annotation:%d->annotation:%d->annotation:%d", usesFactID, runsFactID, versionFactID)
	if !recallCandidateHasGraphProvenance(versionCandidate, evidenceID) {
		t.Fatalf("version provenance = %+v, want graph evidence %s", versionCandidate.Provenance, evidenceID)
	}
	wantNote := fmt.Sprintf("%d -> uses -> LedgerDB -> runs on -> PostgreSQL -> version -> %d", uses.ID, version.ID)
	if !recallCandidateHasGraphNote(versionCandidate, wantNote) {
		t.Fatalf("version provenance = %+v, want relation path %q", versionCandidate.Provenance, wantNote)
	}
}

func TestRecallExpandsTimelineThroughOwnerRelation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	owner, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Owner of Orion is Mira.",
		SessionKey: "sess-annotation-graph-timeline",
	})
	if err != nil {
		t.Fatal(err)
	}
	timeline, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Orion deadline is 2026-06-01.",
		SessionKey: "sess-annotation-graph-timeline",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "When is the deadline for Mira's owned project? deadline Mira owned project deadline Mira owned project. This checklist repeats the retrieval words but names no date.",
		SessionKey: "sess-annotation-graph-timeline",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 300 WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?, ?)
	`, decoy.ID, owner.ID, timeline.ID, decoy.ID, owner.ID, timeline.ID); err != nil {
		t.Fatalf("force lexical decoy recency: %v", err)
	}

	ownerFactID := lookupAnnotationID(t, svc, owner.ID, "Mira owns Orion")
	timelineFactID := lookupAnnotationID(t, svc, timeline.ID, "Orion occurs on 2026-06-01")

	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: "annotation-graph-timeline-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "annotation-graph-timeline-test-v1",
			Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 200,
		},
		now: func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        "team",
		Query:       "When is the deadline for Mira's owned project?",
		SessionKey:  "sess-annotation-graph-timeline",
		ScopeID:     MemoryScopeWorkspace,
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}

	timelineMemoryID := strconv.FormatInt(timeline.ID, 10)
	selected := selectedRecallIDs(trace)
	if !slices.Contains(selected, timelineMemoryID) {
		t.Fatalf("selected IDs = %v candidates=%+v rejected=%+v, want graph-expanded timeline %s", selected, trace.Candidates, trace.Rejected, timelineMemoryID)
	}
	timelineCandidate, ok := selectedRecallCandidate(trace, timelineMemoryID)
	if !ok {
		t.Fatalf("selected = %+v, want timeline candidate", trace.Selected)
	}
	evidenceID := fmt.Sprintf("annotation:%d->annotation:%d", ownerFactID, timelineFactID)
	if !recallCandidateHasGraphProvenance(timelineCandidate, evidenceID) {
		t.Fatalf("timeline provenance = %+v, want graph evidence %s", timelineCandidate.Provenance, evidenceID)
	}
	wantNote := fmt.Sprintf("%d -> owned_entity -> Orion -> timeline -> %d", owner.ID, timeline.ID)
	if !recallCandidateHasGraphNote(timelineCandidate, wantNote) {
		t.Fatalf("timeline provenance = %+v, want relation path %q", timelineCandidate.Provenance, wantNote)
	}
}

func TestRecallExpandsMetricThroughDurableKGRelation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	uses, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Billing API uses VectorDB.",
		SessionKey: "sess-annotation-graph-metric",
	})
	if err != nil {
		t.Fatal(err)
	}
	metric, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: VectorDB latency is 250ms.",
		SessionKey: "sess-annotation-graph-metric",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "How fast is the storage used by Billing API? fast storage used Billing API latency fast storage used Billing API. This checklist repeats the retrieval words but names no metric.",
		SessionKey: "sess-annotation-graph-metric",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 300 WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?, ?)
	`, decoy.ID, uses.ID, metric.ID, decoy.ID, uses.ID, metric.ID); err != nil {
		t.Fatalf("force lexical decoy recency: %v", err)
	}

	usesFactID := lookupAnnotationID(t, svc, uses.ID, "Billing API uses VectorDB")
	metricFactID := lookupAnnotationID(t, svc, metric.ID, "VectorDB latency is 250ms")

	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: "annotation-graph-metric-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "annotation-graph-metric-test-v1",
			Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 200,
		},
		now: func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        "team",
		Query:       "How fast is the storage used by Billing API?",
		SessionKey:  "sess-annotation-graph-metric",
		ScopeID:     MemoryScopeWorkspace,
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}

	metricMemoryID := strconv.FormatInt(metric.ID, 10)
	selected := selectedRecallIDs(trace)
	if !slices.Contains(selected, metricMemoryID) {
		t.Fatalf("selected IDs = %v candidates=%+v rejected=%+v, want graph-expanded metric %s", selected, trace.Candidates, trace.Rejected, metricMemoryID)
	}
	metricCandidate, ok := selectedRecallCandidate(trace, metricMemoryID)
	if !ok {
		t.Fatalf("selected = %+v, want metric candidate", trace.Selected)
	}
	evidenceID := fmt.Sprintf("annotation:%d->annotation:%d", usesFactID, metricFactID)
	if !recallCandidateHasGraphProvenance(metricCandidate, evidenceID) {
		t.Fatalf("metric provenance = %+v, want graph evidence %s", metricCandidate.Provenance, evidenceID)
	}
	wantNote := fmt.Sprintf("%d -> uses -> VectorDB -> metric -> %d", uses.ID, metric.ID)
	if !recallCandidateHasGraphNote(metricCandidate, wantNote) {
		t.Fatalf("metric provenance = %+v, want relation path %q", metricCandidate.Provenance, wantNote)
	}
}

func TestRecallExpandsLocationThroughDurableKGRelation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	uses, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Billing API uses VectorDB.",
		SessionKey: "sess-annotation-graph-location",
	})
	if err != nil {
		t.Fatal(err)
	}
	location, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: VectorDB location is us-east-1.",
		SessionKey: "sess-annotation-graph-location",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Where is the storage used by Billing API? where storage used Billing API location where storage used Billing API. This checklist repeats the retrieval words but names no location.",
		SessionKey: "sess-annotation-graph-location",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 300 WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?, ?)
	`, decoy.ID, uses.ID, location.ID, decoy.ID, uses.ID, location.ID); err != nil {
		t.Fatalf("force lexical decoy recency: %v", err)
	}

	usesFactID := lookupAnnotationID(t, svc, uses.ID, "Billing API uses VectorDB")
	locationFactID := lookupAnnotationID(t, svc, location.ID, "VectorDB is located at us-east-1")

	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: "annotation-graph-location-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "annotation-graph-location-test-v1",
			Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 200,
		},
		now: func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        "team",
		Query:       "Where is the storage used by Billing API?",
		SessionKey:  "sess-annotation-graph-location",
		ScopeID:     MemoryScopeWorkspace,
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}

	locationMemoryID := strconv.FormatInt(location.ID, 10)
	selected := selectedRecallIDs(trace)
	if !slices.Contains(selected, locationMemoryID) {
		t.Fatalf("selected IDs = %v candidates=%+v rejected=%+v, want graph-expanded location %s", selected, trace.Candidates, trace.Rejected, locationMemoryID)
	}
	locationCandidate, ok := selectedRecallCandidate(trace, locationMemoryID)
	if !ok {
		t.Fatalf("selected = %+v, want location candidate", trace.Selected)
	}
	evidenceID := fmt.Sprintf("annotation:%d->annotation:%d", usesFactID, locationFactID)
	if !recallCandidateHasGraphProvenance(locationCandidate, evidenceID) {
		t.Fatalf("location provenance = %+v, want graph evidence %s", locationCandidate.Provenance, evidenceID)
	}
	wantNote := fmt.Sprintf("%d -> uses -> VectorDB -> location -> %d", uses.ID, location.ID)
	if !recallCandidateHasGraphNote(locationCandidate, wantNote) {
		t.Fatalf("location provenance = %+v, want relation path %q", locationCandidate.Provenance, wantNote)
	}
}

func TestRecallExpandsPreferenceThroughDurableKGRelation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	uses, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Billing API uses VectorDB.",
		SessionKey: "sess-annotation-graph-preference",
	})
	if err != nil {
		t.Fatal(err)
	}
	preference, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: VectorDB's indentation preference is tabs.",
		SessionKey: "sess-annotation-graph-preference",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "What indentation does the storage used by Billing API prefer? indentation storage used Billing API prefer indentation storage used Billing API. This checklist repeats the retrieval words but names no preference value.",
		SessionKey: "sess-annotation-graph-preference",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 300 WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?, ?)
	`, decoy.ID, uses.ID, preference.ID, decoy.ID, uses.ID, preference.ID); err != nil {
		t.Fatalf("force lexical decoy recency: %v", err)
	}

	usesFactID := lookupAnnotationID(t, svc, uses.ID, "Billing API uses VectorDB")
	preferenceFactID := lookupAnnotationID(t, svc, preference.ID, "VectorDB prefers tabs for indentation")

	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: "annotation-graph-preference-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "annotation-graph-preference-test-v1",
			Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 200,
		},
		now: func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        "team",
		Query:       "What indentation does the storage used by Billing API prefer?",
		SessionKey:  "sess-annotation-graph-preference",
		ScopeID:     MemoryScopeWorkspace,
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}

	preferenceMemoryID := strconv.FormatInt(preference.ID, 10)
	selected := selectedRecallIDs(trace)
	if !slices.Contains(selected, preferenceMemoryID) {
		t.Fatalf("selected IDs = %v candidates=%+v rejected=%+v, want graph-expanded preference %s", selected, trace.Candidates, trace.Rejected, preferenceMemoryID)
	}
	preferenceCandidate, ok := selectedRecallCandidate(trace, preferenceMemoryID)
	if !ok {
		t.Fatalf("selected = %+v, want preference candidate", trace.Selected)
	}
	evidenceID := fmt.Sprintf("annotation:%d->annotation:%d", usesFactID, preferenceFactID)
	if !recallCandidateHasGraphProvenance(preferenceCandidate, evidenceID) {
		t.Fatalf("preference provenance = %+v, want graph evidence %s", preferenceCandidate.Provenance, evidenceID)
	}
	wantNote := fmt.Sprintf("%d -> uses -> VectorDB -> preference -> %d", uses.ID, preference.ID)
	if !recallCandidateHasGraphNote(preferenceCandidate, wantNote) {
		t.Fatalf("preference provenance = %+v, want relation path %q", preferenceCandidate.Provenance, wantNote)
	}
}

func TestRecallExpandsInstructionThroughDurableKGRelation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	uses, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Billing API uses Exporter.",
		SessionKey: "sess-annotation-graph-instruction",
	})
	if err != nil {
		t.Fatal(err)
	}
	instruction, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Exporter's rule is always encrypt snapshots.",
		SessionKey: "sess-annotation-graph-instruction",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "What rule did the storage used by Billing API give about snapshots? rule storage used Billing API snapshots rule storage used Billing API snapshots. This checklist repeats the retrieval words but names no rule.",
		SessionKey: "sess-annotation-graph-instruction",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 300 WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?, ?)
	`, decoy.ID, uses.ID, instruction.ID, decoy.ID, uses.ID, instruction.ID); err != nil {
		t.Fatalf("force lexical decoy recency: %v", err)
	}

	usesFactID := lookupAnnotationID(t, svc, uses.ID, "Billing API uses Exporter")
	instructionFactID := lookupAnnotationID(t, svc, instruction.ID, "Exporter instructed always encrypt snapshots")

	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: "annotation-graph-instruction-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "annotation-graph-instruction-test-v1",
			Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 200,
		},
		now: func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        "team",
		Query:       "What rule did the storage used by Billing API give about snapshots?",
		SessionKey:  "sess-annotation-graph-instruction",
		ScopeID:     MemoryScopeWorkspace,
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}

	instructionMemoryID := strconv.FormatInt(instruction.ID, 10)
	selected := selectedRecallIDs(trace)
	if !slices.Contains(selected, instructionMemoryID) {
		t.Fatalf("selected IDs = %v candidates=%+v rejected=%+v, want graph-expanded instruction %s", selected, trace.Candidates, trace.Rejected, instructionMemoryID)
	}
	instructionCandidate, ok := selectedRecallCandidate(trace, instructionMemoryID)
	if !ok {
		t.Fatalf("selected = %+v, want instruction candidate", trace.Selected)
	}
	evidenceID := fmt.Sprintf("annotation:%d->annotation:%d", usesFactID, instructionFactID)
	if !recallCandidateHasGraphProvenance(instructionCandidate, evidenceID) {
		t.Fatalf("instruction provenance = %+v, want graph evidence %s", instructionCandidate.Provenance, evidenceID)
	}
	wantNote := fmt.Sprintf("%d -> uses -> Exporter -> instruction -> %d", uses.ID, instruction.ID)
	if !recallCandidateHasGraphNote(instructionCandidate, wantNote) {
		t.Fatalf("instruction provenance = %+v, want relation path %q", instructionCandidate.Provenance, wantNote)
	}
}

func TestRecallExpandsSequenceThroughDurableKGRelation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	uses, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Billing API uses VectorDB migration.",
		SessionKey: "sess-annotation-graph-sequence",
	})
	if err != nil {
		t.Fatal(err)
	}
	sequence, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: VectorDB migration: first freeze writes, then run migration, finally enable readers.",
		SessionKey: "sess-annotation-graph-sequence",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "What is the order of the migration used by Billing API? order migration used Billing API order migration used Billing API. This checklist repeats the retrieval words but names no steps.",
		SessionKey: "sess-annotation-graph-sequence",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 300 WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?, ?)
	`, decoy.ID, uses.ID, sequence.ID, decoy.ID, uses.ID, sequence.ID); err != nil {
		t.Fatalf("force lexical decoy recency: %v", err)
	}

	usesFactID := lookupAnnotationID(t, svc, uses.ID, "Billing API uses VectorDB migration")
	sequenceFactID := lookupAnnotationID(t, svc, sequence.ID, "VectorDB migration is first freeze writes, then run migration, finally enable readers")

	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: "annotation-graph-sequence-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "annotation-graph-sequence-test-v1",
			Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 220,
		},
		now: func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        "team",
		Query:       "What is the order of the migration used by Billing API?",
		SessionKey:  "sess-annotation-graph-sequence",
		ScopeID:     MemoryScopeWorkspace,
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}

	sequenceMemoryID := strconv.FormatInt(sequence.ID, 10)
	selected := selectedRecallIDs(trace)
	if !slices.Contains(selected, sequenceMemoryID) {
		t.Fatalf("selected IDs = %v candidates=%+v rejected=%+v, want graph-expanded sequence %s", selected, trace.Candidates, trace.Rejected, sequenceMemoryID)
	}
	sequenceCandidate, ok := selectedRecallCandidate(trace, sequenceMemoryID)
	if !ok {
		t.Fatalf("selected = %+v, want sequence candidate", trace.Selected)
	}
	evidenceID := fmt.Sprintf("annotation:%d->annotation:%d", usesFactID, sequenceFactID)
	if !recallCandidateHasGraphProvenance(sequenceCandidate, evidenceID) {
		t.Fatalf("sequence provenance = %+v, want graph evidence %s", sequenceCandidate.Provenance, evidenceID)
	}
	wantNote := fmt.Sprintf("%d -> uses -> VectorDB migration -> sequence -> %d", uses.ID, sequence.ID)
	if !recallCandidateHasGraphNote(sequenceCandidate, wantNote) {
		t.Fatalf("sequence provenance = %+v, want relation path %q", sequenceCandidate.Provenance, wantNote)
	}
}

func TestRecallExpandsDecisionThroughDurableKGRelation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	uses, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Billing API uses VectorDB snapshots.",
		SessionKey: "sess-annotation-graph-decision",
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: We decided to keep VectorDB snapshots encrypted.",
		SessionKey: "sess-annotation-graph-decision",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "What did we decide about the storage used by Billing API? decision storage used Billing API decision storage used Billing API. This checklist repeats the retrieval words but names no decision.",
		SessionKey: "sess-annotation-graph-decision",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 300 WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?, ?)
	`, decoy.ID, uses.ID, decision.ID, decoy.ID, uses.ID, decision.ID); err != nil {
		t.Fatalf("force lexical decoy recency: %v", err)
	}

	usesFactID := lookupAnnotationID(t, svc, uses.ID, "Billing API uses VectorDB snapshots")
	decisionFactID := lookupAnnotationID(t, svc, decision.ID, "user decided to keep VectorDB snapshots encrypted")

	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: "annotation-graph-decision-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "annotation-graph-decision-test-v1",
			Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 220,
		},
		now: func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        "team",
		Query:       "What did we decide about the storage used by Billing API?",
		SessionKey:  "sess-annotation-graph-decision",
		ScopeID:     MemoryScopeWorkspace,
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}

	decisionMemoryID := strconv.FormatInt(decision.ID, 10)
	selected := selectedRecallIDs(trace)
	if !slices.Contains(selected, decisionMemoryID) {
		t.Fatalf("selected IDs = %v candidates=%+v rejected=%+v, want graph-expanded decision %s", selected, trace.Candidates, trace.Rejected, decisionMemoryID)
	}
	decisionCandidate, ok := selectedRecallCandidate(trace, decisionMemoryID)
	if !ok {
		t.Fatalf("selected = %+v, want decision candidate", trace.Selected)
	}
	evidenceID := fmt.Sprintf("annotation:%d->annotation:%d", usesFactID, decisionFactID)
	if !recallCandidateHasGraphProvenance(decisionCandidate, evidenceID) {
		t.Fatalf("decision provenance = %+v, want graph evidence %s", decisionCandidate.Provenance, evidenceID)
	}
	wantNote := fmt.Sprintf("%d -> uses -> VectorDB snapshots -> decision -> %d", uses.ID, decision.ID)
	if !recallCandidateHasGraphNote(decisionCandidate, wantNote) {
		t.Fatalf("decision provenance = %+v, want relation path %q", decisionCandidate.Provenance, wantNote)
	}
}

func TestRecallExpandsNegationThroughDurableKGRelation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	uses, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Billing API uses VectorDB snapshots.",
		SessionKey: "sess-annotation-graph-negation",
	})
	if err != nil {
		t.Fatal(err)
	}
	negation, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: We never auto-delete VectorDB snapshots.",
		SessionKey: "sess-annotation-graph-negation",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Did we ever auto-delete the storage used by Billing API? auto-delete storage used Billing API auto-delete storage used Billing API. This checklist repeats the retrieval words but names no denial.",
		SessionKey: "sess-annotation-graph-negation",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 300 WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?, ?)
	`, decoy.ID, uses.ID, negation.ID, decoy.ID, uses.ID, negation.ID); err != nil {
		t.Fatalf("force lexical decoy recency: %v", err)
	}

	usesFactID := lookupAnnotationID(t, svc, uses.ID, "Billing API uses VectorDB snapshots")
	negationFactID := lookupAnnotationID(t, svc, negation.ID, "user never auto-delete VectorDB snapshots")

	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: "annotation-graph-negation-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "annotation-graph-negation-test-v1",
			Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 220,
		},
		now: func() time.Time { return time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC) },
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        "team",
		Query:       "Did we ever auto-delete the storage used by Billing API?",
		SessionKey:  "sess-annotation-graph-negation",
		ScopeID:     MemoryScopeWorkspace,
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}

	negationMemoryID := strconv.FormatInt(negation.ID, 10)
	selected := selectedRecallIDs(trace)
	if !slices.Contains(selected, negationMemoryID) {
		t.Fatalf("selected IDs = %v candidates=%+v rejected=%+v, want graph-expanded negation %s", selected, trace.Candidates, trace.Rejected, negationMemoryID)
	}
	negationCandidate, ok := selectedRecallCandidate(trace, negationMemoryID)
	if !ok {
		t.Fatalf("selected = %+v, want negation candidate", trace.Selected)
	}
	evidenceID := fmt.Sprintf("annotation:%d->annotation:%d", usesFactID, negationFactID)
	if !recallCandidateHasGraphProvenance(negationCandidate, evidenceID) {
		t.Fatalf("negation provenance = %+v, want graph evidence %s", negationCandidate.Provenance, evidenceID)
	}
	wantNote := fmt.Sprintf("%d -> uses -> VectorDB snapshots -> negation -> %d", uses.ID, negation.ID)
	if !recallCandidateHasGraphNote(negationCandidate, wantNote) {
		t.Fatalf("negation provenance = %+v, want relation path %q", negationCandidate.Provenance, wantNote)
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
