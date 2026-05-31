package goncho

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestRecallScoreSignalsAreSingleSourceForWeightedAndDiagnosticVoices(t *testing.T) {
	gotNames := make([]string, 0, len(recallScoreSignals))
	seen := map[string]struct{}{}
	for _, signal := range recallScoreSignals {
		if signal.Name == "" {
			t.Fatalf("recall score signal has empty name: %+v", signal)
		}
		if _, ok := seen[signal.Name]; ok {
			t.Fatalf("duplicate recall score signal %q", signal.Name)
		}
		seen[signal.Name] = struct{}{}
		gotNames = append(gotNames, signal.Name)
	}
	wantNames := []string{"keyword", "semantic", "graph", "fact", "recency", "importance", "scope"}
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("recall score signal names = %v, want %v", gotNames, wantNames)
	}
	for name := range defaultRecallWeights {
		if _, ok := seen[name]; !ok {
			t.Fatalf("default recall weight %q has no shared signal accessor", name)
		}
	}

	score := RecallScore{KeywordScore: 0.11, SemanticScore: 0.12, GraphScore: 0.13, FactScore: 0.14, RecencyScore: 0.15, ImportanceScore: 0.16, ScopeScore: 0.17}
	weights := map[string]float64{"keyword": 1, "semantic": 1, "graph": 1, "fact": 1, "recency": 1, "importance": 1, "scope": 1}
	if got, want := roundRecallFloat(weightedRecallScore(score, weights)), 0.98; got != want {
		t.Fatalf("weighted recall score = %.6f, want %.6f from every shared signal", got, want)
	}

	diagnostics := buildRecallVoiceDiagnostics([]ScoredRecallCandidate{{Score: score}}, nil, RecallScoringConfig{Weights: weights})
	if got := recallVoiceDiagnosticNames(diagnostics); !slices.Equal(got, wantNames) {
		t.Fatalf("diagnostic voice names = %v, want %v", got, wantNames)
	}
}

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

func TestAppendRecallWarningsPreservesDistinctReplayEvidence(t *testing.T) {
	warnings := appendRecallWarnings(nil,
		RecallWarning{Code: RecallWarningSemanticUnavailable, Stage: RecallStageGenerate, Severity: RecallWarningDegraded, Evidence: map[string]string{"generator": "vector", "error": "timeout"}},
		RecallWarning{Code: RecallWarningSemanticUnavailable, Stage: RecallStageGenerate, Severity: RecallWarningDegraded, Evidence: map[string]string{"generator": "graph", "error": "timeout"}},
		RecallWarning{Code: RecallWarningSemanticUnavailable, Stage: RecallStageGenerate, Severity: RecallWarningDegraded, Evidence: map[string]string{"error": "timeout", "generator": "vector"}},
	)

	if len(warnings) != 2 {
		t.Fatalf("warnings = %+v, want two distinct replayable semantic_unavailable warnings", warnings)
	}
	if warnings[0].Evidence["generator"] != "vector" || warnings[1].Evidence["generator"] != "graph" {
		t.Fatalf("warnings = %+v, want stable first-seen evidence order after exact duplicate removal", warnings)
	}
}

func TestRecallPipelineWarningsAreTraceOwned(t *testing.T) {
	engine := newRecallPipelineEngine(staticRecallGenerator{
		warnings: []RecallWarning{{
			Code:     RecallWarningSemanticUnavailable,
			Stage:    RecallStageGenerate,
			Severity: RecallWarningDegraded,
			Message:  "semantic generator unavailable",
			Evidence: map[string]string{"generator": "vector", "error": "timeout"},
		}},
	}, recallPipelineOptions{})

	trace, err := engine.Run(context.Background(), RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "auth", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	trace.Warnings[0].Evidence["generator"] = "corrupted-by-caller"

	again, err := engine.Run(context.Background(), RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "auth", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Warnings[0].Evidence["generator"]; got != "vector" {
		t.Fatalf("warning evidence generator = %q, want engine-owned vector evidence", got)
	}
}

func TestRecallSelectionActionMakesTokenBudgetDecisionReplayable(t *testing.T) {
	candidate := ScoredRecallCandidate{Candidate: RecallCandidate{MemoryID: "mem-action", Content: "one two three"}}
	policy := recallSelectionPolicy{Limit: 2, TokenBudget: 5}

	fits := recallSelectionActionFor(candidate, 2, policy)
	if fits.RejectReason != "" || fits.TokenCost != 3 || fits.Warning.Code != "" {
		t.Fatalf("fits action = %+v, want exact-budget candidate selected without warning", fits)
	}

	over := recallSelectionActionFor(candidate, 3, policy)
	if over.RejectReason != RecallRejectTokenBudget || over.TokenCost != 3 || over.Warning.Code != RecallWarningTokenBudgetTruncated {
		t.Fatalf("over-budget action = %+v, want token-budget rejection with replayable warning", over)
	}
	if !slices.Equal(over.RejectWhy, []string{"used_tokens=3", "candidate_tokens=3", "token_budget=5"}) {
		t.Fatalf("over-budget why = %v, want replayable token accounting", over.RejectWhy)
	}
}

func TestRecallSelectionFlowAccountsForEveryEligibleCandidate(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-too-large-1",
			SourceType: "turn",
			Content:    "auth deployment runbook with many extra words beyond budget",
			ScopeID:    "team",
			CreatedAt:  now,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 1}},
		},
		{
			MemoryID:   "mem-too-large-2",
			SourceType: "turn",
			Content:    "auth rollback checklist with many extra words beyond budget",
			ScopeID:    "team",
			CreatedAt:  now,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.9}},
		},
		{
			MemoryID:   "mem-small",
			SourceType: "turn",
			Content:    "fallback",
			ScopeID:    "team",
			CreatedAt:  now,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.8}},
		},
		{
			MemoryID:   "mem-other-scope",
			SourceType: "turn",
			Content:    "auth other scope",
			ScopeID:    "other",
			CreatedAt:  now,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.7}},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "selection-accounting-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "selection-accounting-test-v1",
			Weights:     map[string]float64{"keyword": 1},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 2,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "auth", ScopeID: "team", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := selectedRecallIDs(trace); !slices.Equal(got, []string{"mem-small"}) {
		t.Fatalf("selected IDs = %v, want small fallback after oversized candidates are rejected", got)
	}
	if got := rejectedRecallIDs(trace); !slices.Equal(got, []string{"mem-other-scope", "mem-too-large-1", "mem-too-large-2"}) {
		t.Fatalf("rejected IDs = %v, want every non-selected candidate accounted for", got)
	}
	if len(trace.Selected)+len(trace.Rejected) != len(trace.Candidates) {
		t.Fatalf("selection accounting mismatch: selected=%d rejected=%d candidates=%d", len(trace.Selected), len(trace.Rejected), len(trace.Candidates))
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

func TestRecallPipelineRejectedOrderPreservesSelectionFlow(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-z-too-large",
			SourceType: "turn",
			Content:    "auth deployment runbook with many extra words beyond budget",
			SessionID:  "sess-large",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 1}},
		},
		{
			MemoryID:   "mem-a-small",
			SourceType: "turn",
			Content:    "fallback",
			SessionID:  "sess-small",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.9}},
		},
		{
			MemoryID:   "mem-b-leftover",
			SourceType: "turn",
			Content:    "second fallback",
			SessionID:  "sess-leftover",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.5,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.8}},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "rejection-flow-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "rejection-flow-test-v1",
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
	if got := selectedRecallIDs(trace); !slices.Equal(got, []string{"mem-a-small"}) {
		t.Fatalf("selected IDs = %v, want fallback selected after oversized best is rejected", got)
	}
	if got := rejectedRecallIDs(trace); !slices.Equal(got, []string{"mem-z-too-large", "mem-b-leftover"}) {
		t.Fatalf("rejected IDs = %v, want selection-flow order: token-budget rejection before leftover", got)
	}
	if trace.Rejected[0].Reason != RecallRejectTokenBudget || trace.Rejected[1].Reason != RecallRejectNotSelected {
		t.Fatalf("rejected = %+v, want token-budget rejection before not-selected leftover", trace.Rejected)
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

func TestRecallCoverageBonusTrimsGraphPathProvenance(t *testing.T) {
	selected := []ScoredRecallCandidate{{Candidate: RecallCandidate{MemoryID: "mem-auth-service"}}}
	candidate := ScoredRecallCandidate{Candidate: RecallCandidate{
		MemoryID:   "mem-auth-owner",
		Provenance: []EvidenceItem{{Kind: "graph", Note: "  mem-auth-service -> owned_by -> mem-auth-owner  "}},
	}}

	if got := recallCoverageBonus(candidate, selected); got != recallGraphCoverageBonus {
		t.Fatalf("coverage bonus = %.6f, want %.6f for whitespace-padded relation path provenance", got, recallGraphCoverageBonus)
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

func TestRecallScopeSelectionInputsExposeScopedEligibility(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	scored := []ScoredRecallCandidate{
		{Candidate: RecallCandidate{MemoryID: "team", ScopeID: "team", CreatedAt: now}},
		{Candidate: RecallCandidate{MemoryID: "global", CreatedAt: now}},
		{Candidate: RecallCandidate{MemoryID: "other", ScopeID: "other", CreatedAt: now}},
	}

	eligible, rejected, warnings := recallScopeSelectionInputs(RecallQuery{ScopeID: "team"}, scored)

	if got := scoredRecallCandidateIDs(eligible); !slices.Equal(got, []string{"team", "global"}) {
		t.Fatalf("eligible IDs = %v, want matching scope plus unscoped/global candidate", got)
	}
	if got := rejectedRecallCandidateIDs(rejected); !slices.Equal(got, []string{"other"}) {
		t.Fatalf("rejected IDs = %v, want only conflicting scope rejected", got)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %+v, want no all-excluded warning while global candidate remains eligible", warnings)
	}
}

func TestRecallSelectionPolicyMakesLimitAndBudgetAssumptionsExplicit(t *testing.T) {
	policy := recallSelectionPolicyFor(RecallQuery{Limit: 0}, RecallScoringConfig{TokenBudget: 120})
	if policy.Limit != 5 || policy.TokenBudget != 120 {
		t.Fatalf("default policy = %+v, want limit default 5 and config token budget", policy)
	}
	policy = recallSelectionPolicyFor(RecallQuery{Limit: 2, MaxTokens: 9}, RecallScoringConfig{TokenBudget: 120})
	if policy.Limit != 2 || policy.TokenBudget != 9 {
		t.Fatalf("override policy = %+v, want query limit and max_tokens override", policy)
	}
	if !policy.FitsTokenBudget(4, 5) || policy.FitsTokenBudget(5, 5) {
		t.Fatalf("budget fit checks for %+v were not boundary-inclusive", policy)
	}
	if got := strings.Join(policy.TokenBudgetRejectionReasons(5, 5), ";"); got != "used_tokens=5;candidate_tokens=5;token_budget=9" {
		t.Fatalf("rejection reasons = %q", got)
	}
	warning := policy.TokenBudgetWarning()
	if warning.Code != RecallWarningTokenBudgetTruncated || warning.Evidence["token_budget"] != "9" {
		t.Fatalf("warning = %+v, want replayable token-budget evidence", warning)
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

func rejectedRecallIDs(trace RecallTrace) []string {
	return rejectedRecallCandidateIDs(trace.Rejected)
}

func scoredRecallCandidateIDs(candidates []ScoredRecallCandidate) []string {
	out := make([]string, 0, len(candidates))
	for _, item := range candidates {
		out = append(out, item.Candidate.MemoryID)
	}
	return out
}

func rejectedRecallCandidateIDs(candidates []RejectedRecallCandidate) []string {
	out := make([]string, 0, len(candidates))
	for _, item := range candidates {
		out = append(out, item.Candidate.MemoryID)
	}
	return out
}

func recallVoiceDiagnosticNames(diagnostics []RecallVoiceDiagnostic) []string {
	out := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, diagnostic.Name)
	}
	return out
}
