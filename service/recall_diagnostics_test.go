package goncho

import (
	"strings"
	"testing"
)

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
