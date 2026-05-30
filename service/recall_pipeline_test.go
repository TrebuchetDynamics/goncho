package goncho

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
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
		if !traceHasWarning(trace, code) {
			t.Fatalf("warnings = %+v, missing %s", trace.Warnings, code)
		}
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
	if !traceHasWarning(trace, RecallWarningScopeExcludedAllCandidates) {
		t.Fatalf("warnings = %+v, missing scope exclusion warning", trace.Warnings)
	}
	if len(trace.Rejected) != 1 || trace.Rejected[0].Reason != RecallRejectScopeMismatch {
		t.Fatalf("rejected = %+v, want one scope mismatch", trace.Rejected)
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

func traceHasWarning(trace RecallTrace, code string) bool {
	return sliceutil.ContainsFunc(trace.Warnings, func(warning RecallWarning) bool {
		return warning.Code == code
	})
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
