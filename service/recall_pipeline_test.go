package goncho

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestRecallPipelineWarningsAndTokenBudget(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	config := RecallScoringConfig{
		Version:       "warnings-v1",
		Weights:       map[string]float64{"keyword": 1},
		RRFK:          60,
		MMRLambda:     1,
		DiversityKeys: []string{"session_id"},
		TokenBudget:   9,
	}
	engine := newRecallPipelineEngine(staticRecallGenerator{
		candidates: []RecallCandidate{
			{
				MemoryID:   "mem-a",
				SourceType: "turn",
				Content:    "short auth fact",
				SessionID:  "sess-a",
				ScopeID:    "team",
				CreatedAt:  now,
				Importance: 0.5,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 1}},
			},
			{
				MemoryID:   "mem-b",
				SourceType: "turn",
				Content:    "this candidate is too long for the configured budget",
				SessionID:  "sess-b",
				ScopeID:    "team",
				CreatedAt:  now,
				Importance: 0.5,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.9}},
			},
		},
		warnings: []RecallWarning{
			{Code: RecallWarningSemanticUnavailable, Stage: RecallStageGenerate, Severity: RecallWarningDegraded, Message: "semantic generator unavailable"},
			{Code: RecallWarningGraphDisabled, Stage: RecallStageGenerate, Severity: RecallWarningInfo, Message: "graph generator disabled"},
			{Code: RecallWarningStaleEmbeddingIndex, Stage: RecallStageGenerate, Severity: RecallWarningDegraded, Message: "embedding index stale"},
			{Code: RecallWarningFTSUnavailable, Stage: RecallStageGenerate, Severity: RecallWarningDegraded, Message: "fts table missing"},
		},
	}, recallPipelineOptions{
		pipelineVersion: "test-pipeline",
		scoringConfig:   config,
		now:             func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "auth",
		ScopeID:     "team",
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-a"}) {
		t.Fatalf("selected IDs = %v, want only mem-a within budget", selectedRecallIDs(trace))
	}
	for _, code := range []string{
		RecallWarningSemanticUnavailable,
		RecallWarningGraphDisabled,
		RecallWarningStaleEmbeddingIndex,
		RecallWarningFTSUnavailable,
		RecallWarningTokenBudgetTruncated,
	} {
		if !recallTraceHasWarning(trace, code) {
			t.Fatalf("warnings = %+v, missing %s", trace.Warnings, code)
		}
	}
}

func TestRecallPipelineTokenBudgetSkipsOversizedBestCandidate(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-too-large",
			SourceType: "turn",
			Content:    "auth deployment runbook with many extra words beyond budget",
			SessionID:  "sess-large",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 1}},
		},
		{
			MemoryID:   "mem-small",
			SourceType: "turn",
			Content:    "fallback",
			SessionID:  "sess-small",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.1}},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "budget-skip-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "budget-skip-test-v1",
			Weights:     map[string]float64{"keyword": 1},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 3,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "auth",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-small"}) {
		t.Fatalf("selected IDs = %v, want oversized best candidate skipped and smaller candidate selected", selectedRecallIDs(trace))
	}
	if len(trace.Rejected) != 1 || trace.Rejected[0].Candidate.MemoryID != "mem-too-large" || trace.Rejected[0].Reason != RecallRejectTokenBudget {
		t.Fatalf("rejected = %+v, want oversized candidate rejected by token budget", trace.Rejected)
	}
}

func TestRecallPipelineSingleNameSpeakerTargetMatchesFullSpeakerIdentity(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-juan-perez",
			SourceType: "turn",
			Content:    "Juan Perez explained the deployment rollback window.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.50},
				{Kind: "speaker", Source: "Juan Perez"},
			},
		},
		{
			MemoryID:   "mem-maria",
			SourceType: "turn",
			Content:    "Maria repeated deployment rollback notes from Juan.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.51},
				{Kind: "speaker", Source: "Maria"},
			},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "single-name-speaker-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "single-name-speaker-test-v1",
			Weights:     map[string]float64{"keyword": 1},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 80,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "what did Juan say about deployments?",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-juan-perez"}) {
		t.Fatalf("selected IDs = %v, want single-name target to match full speaker identity", selectedRecallIDs(trace))
	}
	if !strings.Contains(strings.Join(trace.Selected[0].Score.WhySelected, "\n"), "speaker_adjustment=0.120000") {
		t.Fatalf("why selected = %v, want speaker adjustment evidence", trace.Selected[0].Score.WhySelected)
	}
}

func TestRecallPipelineMultiWordSpeakerTargetDoesNotMatchSameFirstName(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-juan-lopez",
			SourceType: "turn",
			Content:    "Deployment rollback window changed today.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.20},
				{Kind: "speaker", Source: "Juan Lopez"},
			},
		},
		{
			MemoryID:   "mem-juan-perez",
			SourceType: "turn",
			Content:    "Requested speaker gave the lower-keyword deployment note.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.10},
				{Kind: "speaker", Source: "Juan Perez"},
			},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "multi-word-speaker-disambiguation-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "multi-word-speaker-disambiguation-test-v1",
			Weights:     map[string]float64{"keyword": 1},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 80,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "what did Juan Perez say about deployments?",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-juan-perez"}) {
		t.Fatalf("selected IDs = %v, want full multi-word speaker target to disambiguate same first-name speakers", selectedRecallIDs(trace))
	}
	if !strings.Contains(strings.Join(trace.Selected[0].Score.WhySelected, "\n"), "speaker_adjustment=0.120000") {
		t.Fatalf("why selected = %v, want speaker adjustment evidence for exact multi-word speaker target", trace.Selected[0].Score.WhySelected)
	}
}

func TestRecallPipelineMultiWordSpeakerTargetGetsSpeakerBonus(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-juan-perez",
			SourceType: "turn",
			Content:    "Deployment owner is Juan Perez.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.50},
				{Kind: "speaker", Source: "Juan Perez"},
			},
		},
		{
			MemoryID:   "mem-maria",
			SourceType: "turn",
			Content:    "Deployment owner is Maria.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.51},
				{Kind: "speaker", Source: "Maria"},
			},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "multi-word-speaker-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "multi-word-speaker-test-v1",
			Weights:     map[string]float64{"keyword": 1},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 80,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "what did Juan Perez say about deployments?",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-juan-perez"}) {
		t.Fatalf("selected IDs = %v, want multi-word speaker target to receive speaker bonus", selectedRecallIDs(trace))
	}
	if !strings.Contains(strings.Join(trace.Selected[0].Score.WhySelected, "\n"), "speaker_adjustment=0.120000") {
		t.Fatalf("why selected = %v, want speaker adjustment evidence", trace.Selected[0].Score.WhySelected)
	}
}

func TestRecallPipelineScopeWarningWhenAllCandidatesExcluded(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-other",
			SourceType: "turn",
			Content:    "other scope memory",
			SessionID:  "sess-other",
			ScopeID:    "other",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 1}},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "test-pipeline",
		scoringConfig: RecallScoringConfig{
			Version:     "scope-v1",
			Weights:     map[string]float64{"keyword": 1},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 100,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "auth",
		ScopeID:     "team",
		Limit:       5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.Selected) != 0 {
		t.Fatalf("selected = %+v, want no cross-scope candidates", trace.Selected)
	}
	if !recallTraceHasWarning(trace, RecallWarningScopeExcludedAllCandidates) {
		t.Fatalf("warnings = %+v, missing scope exclusion warning", trace.Warnings)
	}
	if len(trace.Rejected) != 1 || trace.Rejected[0].Reason != RecallRejectScopeMismatch {
		t.Fatalf("rejected = %+v, want one scope mismatch", trace.Rejected)
	}
}

func TestRecallPipelineDiversityKeyMemoryIDPenalizesDuplicateMemories(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-duplicate",
			Content:    "Auth deploys use the blue environment.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.8,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 1.0}},
		},
		{
			MemoryID:   "mem-duplicate",
			Content:    "Auth deploys use blue environment duplicate wording.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.8,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.99}},
		},
		{
			MemoryID:   "mem-distinct",
			Content:    "Billing deploys use the green environment.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.8,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.98}},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "memory-diversity-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:       "memory-diversity-test-v1",
			Weights:       map[string]float64{"keyword": 1},
			RRFK:          60,
			MMRLambda:     0.01,
			DiversityKeys: []string{"memory_id"},
			TokenBudget:   120,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "auth",
		ScopeID:     "team",
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-duplicate", "mem-distinct"}) {
		t.Fatalf("selected IDs = %v, want duplicate memory_id penalized before second selection", selectedRecallIDs(trace))
	}
}

func TestRecallPipelineCoverageAwareSelectionKeepsGraphCompanion(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-auth-service",
			Content:    "Authentication service handles login flows.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.8,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 1.0}},
		},
		{
			MemoryID:   "mem-auth-service-dup",
			Content:    "Authentication service handles login flows and session refresh.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.8,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.99}},
		},
		{
			MemoryID:   "mem-auth-owner",
			Content:    "Mira owns component A-17.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.8,
			Provenance: []EvidenceItem{{Kind: "graph", Source: "mem-auth-service", Score: 0.98, Note: "mem-auth-service -> owned_by -> mem-auth-owner"}},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "coverage-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:       "coverage-test-v1",
			Weights:       map[string]float64{"keyword": 0.45, "graph": 0.45, "scope": 0.10},
			RRFK:          60,
			MMRLambda:     0.70,
			DiversityKeys: []string{"memory_id"},
			TokenBudget:   120,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "authentication owner",
		ScopeID:     "team",
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-auth-service", "mem-auth-owner"}) {
		t.Fatalf("selected IDs = %v, want coverage-aware selection", selectedRecallIDs(trace))
	}
}

func TestRecallPipelineSelectedReasonsReportAdjustedFinalScore(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{{
		MemoryID:  "mem-current",
		Content:   "The current owner is Mira.",
		ScopeID:   "team",
		CreatedAt: now,
		Provenance: []EvidenceItem{
			{Kind: "keyword", Score: 1},
			{Kind: "temporal", Note: "current_fact valid_now"},
		},
	}}}, recallPipelineOptions{
		pipelineVersion: "adjusted-score-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "adjusted-score-test-v1",
			Weights:     map[string]float64{"keyword": 1},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 100,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "current owner",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.Selected) != 1 {
		t.Fatalf("selected = %+v, want one current-truth candidate", trace.Selected)
	}
	selected := trace.Selected[0]
	wantReason := fmt.Sprintf("final_score=%.6f", selected.Score.FinalScore)
	if !strings.Contains(strings.Join(selected.Score.WhySelected, ";"), wantReason) {
		t.Fatalf("selected score = %.6f why = %+v, want reason %q after selection adjustments", selected.Score.FinalScore, selected.Score.WhySelected, wantReason)
	}
}

func TestRecallPipelineRRFIgnoresAbsentSignalScores(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-b",
			Content:    "candidate without semantic evidence",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0,
		},
		{
			MemoryID:   "mem-a",
			Content:    "another candidate without semantic evidence",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0,
		},
	}}, recallPipelineOptions{
		pipelineVersion: "absent-signal-rrf-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "absent-signal-rrf-test-v1",
			Weights:     map[string]float64{"semantic": 1},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 100,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "semantic only",
		ScopeID:     "team",
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range trace.Candidates {
		if item.Score.SemanticScore != 0 || item.Score.RRFScore != 0 || item.Score.FinalScore != 0 {
			t.Fatalf("candidate %s score = %+v, want absent semantic signal to contribute no score", item.Candidate.MemoryID, item.Score)
		}
	}
}

func TestRecallPipelineCopiesScoringConfig(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	config := RecallScoringConfig{
		Version:       "copy-v1",
		Weights:       map[string]float64{"keyword": 1},
		RRFK:          60,
		MMRLambda:     1,
		DiversityKeys: []string{"session_id"},
		TokenBudget:   100,
	}
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{{
		MemoryID:   "mem-a",
		SourceType: "turn",
		Content:    "auth fact",
		SessionID:  "sess-a",
		ScopeID:    "team",
		CreatedAt:  now,
		Importance: 0.5,
		Provenance: []EvidenceItem{{Kind: "keyword", Score: 1}},
	}}}, recallPipelineOptions{
		pipelineVersion: "test-pipeline",
		scoringConfig:   config,
		now:             func() time.Time { return now },
	})
	config.Weights["keyword"] = 0
	config.DiversityKeys[0] = "scope_id"

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "auth",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if trace.ScoringConfig.Weights["keyword"] != 1 || !slices.Equal(trace.ScoringConfig.DiversityKeys, []string{"session_id"}) {
		t.Fatalf("trace scoring config = %+v, want engine-owned copy", trace.ScoringConfig)
	}
	trace.ScoringConfig.Weights["keyword"] = 0
	trace.ScoringConfig.DiversityKeys[0] = "scope_id"
	again, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "auth",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.ScoringConfig.Weights["keyword"] != 1 || !slices.Equal(again.ScoringConfig.DiversityKeys, []string{"session_id"}) {
		t.Fatalf("next trace scoring config = %+v, want fresh copy", again.ScoringConfig)
	}
}

type staticRecallGenerator struct {
	candidates []RecallCandidate
	warnings   []RecallWarning
}

func (g staticRecallGenerator) Generate(ctx context.Context, q RecallQuery) ([]RecallCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]RecallCandidate, len(g.candidates))
	copy(out, g.candidates)
	return out, nil
}

func (g staticRecallGenerator) RecallWarnings() []RecallWarning {
	out := make([]RecallWarning, len(g.warnings))
	copy(out, g.warnings)
	return out
}
