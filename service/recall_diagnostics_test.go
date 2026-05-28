package goncho

import (
	"strings"
	"testing"
)

func TestRecallVoiceDiagnosticsReportsPerVoiceStatsForAllScoredCandidates(t *testing.T) {
	trace := RecallTrace{
		TraceID:         "trace-voice",
		PipelineVersion: "test-pipeline",
		Query:           RecallQuery{WorkspaceID: "default", Peer: "user", Query: "auth"},
		ScoringConfig: RecallScoringConfig{
			Version: "voice-v1",
			Weights: map[string]float64{"keyword": 0.5, "semantic": 0.3, "fact": 0.2},
			RRFK:    60,
		},
		VoiceDiagnostics: []RecallVoiceDiagnostic{
			{Name: "keyword", Enabled: true, Weight: 0.5, CandidatesWith: 2, MaxScore: 1.0, MinScore: 0.5, AvgScore: 0.75, SelectedCount: 1},
			{Name: "semantic", Enabled: true, Weight: 0.3, CandidatesWith: 1, MaxScore: 0.8, MinScore: 0.0, AvgScore: 0.4, SelectedCount: 1},
			{Name: "graph", Enabled: false, Weight: 0.0, CandidatesWith: 0, MaxScore: 0.0, MinScore: 0.0, AvgScore: 0.0, SelectedCount: 0},
			{Name: "fact", Enabled: true, Weight: 0.2, CandidatesWith: 1, MaxScore: 0.9, MinScore: 0.0, AvgScore: 0.45, SelectedCount: 0},
			{Name: "recency", Enabled: false, Weight: 0.0, CandidatesWith: 0, MaxScore: 0.0, MinScore: 0.0, AvgScore: 0.0, SelectedCount: 0},
			{Name: "importance", Enabled: false, Weight: 0.0, CandidatesWith: 0, MaxScore: 0.0, MinScore: 0.0, AvgScore: 0.0, SelectedCount: 0},
			{Name: "scope", Enabled: false, Weight: 0.0, CandidatesWith: 0, MaxScore: 0.0, MinScore: 0.0, AvgScore: 0.0, SelectedCount: 0},
		},
		Candidates: []ScoredRecallCandidate{
			{Candidate: RecallCandidate{MemoryID: "mem-a"}},
			{Candidate: RecallCandidate{MemoryID: "mem-b"}},
		},
		Selected: []ScoredRecallCandidate{
			{Candidate: RecallCandidate{MemoryID: "mem-a"}, Score: RecallScore{KeywordScore: 1.0, SemanticScore: 0.8}},
		},
	}

	if len(trace.VoiceDiagnostics) != 7 {
		t.Fatalf("voice diagnostics count = %d, want 7", len(trace.VoiceDiagnostics))
	}

	// Verify keyword voice (highest weight, should be populated).
	kw := trace.VoiceDiagnostics[0]
	if kw.Name != "keyword" || !kw.Enabled || kw.Weight != 0.5 || kw.CandidatesWith != 2 || kw.MaxScore != 1.0 || kw.MinScore != 0.5 || kw.AvgScore != 0.75 || kw.SelectedCount != 1 {
		t.Fatalf("keyword voice = %+v", kw)
	}

	// Verify graph voice (disabled).
	graph := trace.VoiceDiagnostics[2]
	if graph.Name != "graph" || graph.Enabled || graph.Weight != 0.0 || graph.SelectedCount != 0 {
		t.Fatalf("graph voice = %+v, want disabled with zero weight", graph)
	}

	// Verify fact voice (enabled but didn't match selected).
	fact := trace.VoiceDiagnostics[3]
	if fact.Name != "fact" || !fact.Enabled || fact.SelectedCount != 0 {
		t.Fatalf("fact voice = %+v, want enabled with zero selected", fact)
	}

	// Verify voice diagnostics survive JSON round-trip.
	raw, err := trace.StableJSON()
	if err != nil {
		t.Fatalf("StableJSON: %v", err)
	}
	if !strings.Contains(string(raw), "voice_diagnostics") {
		t.Fatalf("StableJSON missing voice_diagnostics: %s", raw)
	}
}

func TestBuildRecallVoiceDiagnosticsComputesStatsFromScoredAndSelected(t *testing.T) {
	config := RecallScoringConfig{
		Version: "test",
		Weights: map[string]float64{
			"keyword":    0.3,
			"semantic":   0.2,
			"graph":      0.0,
			"fact":       0.15,
			"recency":    0.1,
			"importance": 0.05,
			"scope":      0.05,
		},
	}
	scored := []ScoredRecallCandidate{
		{Candidate: RecallCandidate{MemoryID: "a"}, Score: RecallScore{KeywordScore: 1.0, SemanticScore: 0.5, GraphScore: 0.0, FactScore: 0.8, RecencyScore: 0.3, ImportanceScore: 0.7, ScopeScore: 0.2}},
		{Candidate: RecallCandidate{MemoryID: "b"}, Score: RecallScore{KeywordScore: 0.5, SemanticScore: 0.0, GraphScore: 0.0, FactScore: 0.0, RecencyScore: 0.6, ImportanceScore: 0.0, ScopeScore: 0.9}},
	}
	selected := []ScoredRecallCandidate{scored[0]} // only candidate "a" selected

	diags := buildRecallVoiceDiagnostics(scored, selected, config)

	if len(diags) != 7 {
		t.Fatalf("diags count = %d, want 7", len(diags))
	}

	// keyword: enabled, both candidates have scores
	kw := diags[0]
	if kw.Name != "keyword" || !kw.Enabled || kw.Weight != 0.3 || kw.CandidatesWith != 2 || kw.MaxScore != 1.0 || kw.MinScore != 0.5 || kw.AvgScore != 0.75 || kw.SelectedCount != 1 {
		t.Fatalf("keyword = %+v", kw)
	}

	// semantic: enabled, one candidate has score
	sem := diags[1]
	if sem.Name != "semantic" || !sem.Enabled || sem.Weight != 0.2 || sem.CandidatesWith != 1 || sem.MaxScore != 0.5 || sem.MinScore != 0.0 || sem.AvgScore != 0.25 || sem.SelectedCount != 1 {
		t.Fatalf("semantic = %+v", sem)
	}

	// graph: disabled (weight=0)
	gr := diags[2]
	if gr.Name != "graph" || gr.Enabled || gr.Weight != 0.0 || gr.CandidatesWith != 0 {
		t.Fatalf("graph = %+v, want disabled", gr)
	}

	// scope: enabled, selected candidate "a" has scope 0.2
	sc := diags[6]
	if sc.Name != "scope" || !sc.Enabled || sc.Weight != 0.05 || sc.MaxScore != 0.9 || sc.MinScore != 0.2 || sc.SelectedCount != 1 {
		t.Fatalf("scope = %+v, want selected_count=1 (selected mem-a has score=0.2)", sc)
	}
}

func TestRecallDiagnosticsReportExplainsSelectedRejectedAndWarnings(t *testing.T) {
	trace := RecallTrace{
		TraceID:         "trace-123",
		PipelineVersion: "test-pipeline",
		Query:           RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "auth", ScopeID: "team"},
		ScoringConfig:   RecallScoringConfig{Version: "diag-v1", Weights: map[string]float64{"keyword": 1}, RRFK: 60, MMRLambda: 0.7, TokenBudget: 64},
		Candidates: []ScoredRecallCandidate{
			{Candidate: RecallCandidate{MemoryID: "mem-a"}},
			{Candidate: RecallCandidate{MemoryID: "mem-b"}},
		},
		Selected: []ScoredRecallCandidate{{
			Candidate: RecallCandidate{MemoryID: "mem-a", SourceType: "conclusion", Content: "JWT auth uses jose middleware.", SessionID: "sess-a", ScopeID: "team"},
			Score: RecallScore{
				KeywordScore: 0.9,
				FinalScore:   0.8,
				WhySelected:  []string{"final_score=0.800000", "scoring_config=diag-v1"},
			},
		}},
		Rejected: []RejectedRecallCandidate{{
			Candidate:   RecallCandidate{MemoryID: "mem-b", SourceType: "turn", Content: "other scope", SessionID: "sess-b", ScopeID: "other"},
			Score:       RecallScore{KeywordScore: 0.2, SemanticScore: 0.4, FinalScore: 0.5},
			Reason:      RecallRejectScopeMismatch,
			WhyRejected: []string{"candidate_scope=other", "query_scope=team"},
		}},
		Warnings: []RecallWarning{{
			Code:     RecallWarningSemanticUnavailable,
			Stage:    RecallStageGenerate,
			Severity: RecallWarningDegraded,
			Message:  "semantic generator unavailable",
		}},
	}

	report := BuildRecallDiagnostics(trace)
	if report.Status != "degraded" {
		t.Fatalf("Status = %q, want degraded", report.Status)
	}
	if report.TraceID != "trace-123" || report.ScoringConfig.Version != "diag-v1" || report.CandidateCount != 2 {
		t.Fatalf("report header = %+v", report)
	}
	if len(report.Selected) != 1 || report.Selected[0].MemoryID != "mem-a" || report.Selected[0].FinalScore != 0.8 {
		t.Fatalf("selected = %+v", report.Selected)
	}
	if !strings.Contains(strings.Join(report.Selected[0].WhySelected, " "), "scoring_config=diag-v1") {
		t.Fatalf("selected reasons = %+v", report.Selected[0].WhySelected)
	}
	if len(report.Rejected) != 1 || report.Rejected[0].Reason != RecallRejectScopeMismatch {
		t.Fatalf("rejected = %+v", report.Rejected)
	}
	if len(report.Warnings) != 1 || report.Warnings[0].Code != RecallWarningSemanticUnavailable {
		t.Fatalf("warnings = %+v", report.Warnings)
	}

	text := FormatRecallDiagnosticsReport(report)
	for _, want := range []string{
		"Goncho recall diagnostics",
		"status: degraded",
		"trace_id: trace-123",
		"scoring_config: diag-v1",
		"selected candidates",
		"mem-a",
		"why: final_score=0.800000; scoring_config=diag-v1",
		"rejected candidates",
		"mem-b",
		"reason=scope_mismatch",
		"scores: keyword=0.200000 semantic=0.400000",
		"warnings",
		"semantic_unavailable",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("formatted diagnostics missing %q:\n%s", want, text)
		}
	}
}
