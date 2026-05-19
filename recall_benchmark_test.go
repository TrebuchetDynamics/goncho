package goncho

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecallBenchmarkCorpusReportFixture(t *testing.T) {
	cases := []RecallBenchmarkCase{
		{
			ID:              "stable-trace-auth",
			Trace:           loadRecallBenchmarkTrace(t, filepath.Join("testdata", "recall_trace", "stable_trace.golden.json")),
			RelevantIDs:     []string{"mem-auth", "mem-rate"},
			ContextContains: []string{"JWT auth uses jose middleware."},
			Latency:         12 * time.Millisecond,
		},
		{
			ID:              "golden-turn-preference",
			Trace:           recallBenchmarkGoldenTurnTrace(),
			RelevantIDs:     []string{"golden-pref"},
			ContextContains: []string{"evidence-first reports", "RecallTrace"},
			Latency:         25 * time.Millisecond,
		},
	}

	report := EvaluateRecallBenchmark(cases)
	if report.Service != "goncho" || report.CorpusVersion != RecallBenchmarkCorpusVersion {
		t.Fatalf("report metadata = %+v", report)
	}
	if report.CaseCount != 2 {
		t.Fatalf("case count = %d, want 2", report.CaseCount)
	}
	if report.RecallAt5 != 1 || report.RecallAt10 != 1 {
		t.Fatalf("recall = @5 %.3f @10 %.3f, want 1.0", report.RecallAt5, report.RecallAt10)
	}
	if report.ContextHitRate != 1 || report.TokenBudgetPassRate != 1 {
		t.Fatalf("context/token rates = %.3f/%.3f, want 1.0", report.ContextHitRate, report.TokenBudgetPassRate)
	}
	if report.Latency.P50MS != 12 || report.Latency.P95MS != 25 {
		t.Fatalf("latency = %+v, want p50=12 p95=25", report.Latency)
	}
	if len(report.Cases) != 2 || report.Cases[0].CandidateMemoryIDs[0] != "mem-auth" || report.Cases[1].SelectedMemoryIDs[0] != "golden-pref" {
		t.Fatalf("case summaries = %+v", report.Cases)
	}
	assertRecallBenchmarkReportFixture(t, report)
}

func TestRecallBenchmarkCorpusWarningsAreCodeFirst(t *testing.T) {
	report := EvaluateRecallBenchmark([]RecallBenchmarkCase{{
		ID: "broken-case",
		Trace: RecallTrace{
			TraceID:         "trace-broken",
			PipelineVersion: "test",
			Query:           RecallQuery{WorkspaceID: "default", Peer: "user", Query: "missing"},
			ScoringConfig:   RecallScoringConfig{Version: "test-v1"},
		},
	}})
	if report.WarningCount != 1 || len(report.Warnings) != 1 {
		t.Fatalf("warnings = %+v, want one code-first warning", report.Warnings)
	}
	if report.Warnings[0].Code != RecallBenchmarkWarningNoRelevantIDs || report.Warnings[0].Stage != RecallStageScore {
		t.Fatalf("warning = %+v, want no relevant IDs scoring warning", report.Warnings[0])
	}
	if report.Cases[0].RecallAt5 != 0 || report.Cases[0].ContextSatisfied {
		t.Fatalf("broken case = %+v, want zero recall and unsatisfied context", report.Cases[0])
	}
}

func loadRecallBenchmarkTrace(t *testing.T, path string) RecallTrace {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trace fixture: %v", err)
	}
	var trace RecallTrace
	if err := json.Unmarshal(raw, &trace); err != nil {
		t.Fatalf("decode trace fixture: %v", err)
	}
	return trace
}

func recallBenchmarkGoldenTurnTrace() RecallTrace {
	return RecallTrace{
		TraceID:         "trace-golden-turn-pref",
		PipelineVersion: "goncho-golden-e2e-v1",
		CreatedAt:       time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
		Query: RecallQuery{
			WorkspaceID: "default",
			Peer:        "telegram:6586915095",
			Query:       "How should you answer me about Goncho RecallTrace evidence?",
			SessionKey:  "sess-goncho-golden",
			ScopeID:     "default",
			Limit:       5,
			MaxTokens:   400,
		},
		ScoringConfig: RecallScoringConfig{
			Version:       "goncho-golden-scoring-v1",
			Weights:       map[string]float64{"keyword": 0.6, "semantic": 0.2, "graph": 0.1, "recency": 0.05, "importance": 0.03, "scope": 0.02},
			RRFK:          60,
			MMRLambda:     1,
			DiversityKeys: []string{"session_id", "source_type"},
			TokenBudget:   400,
		},
		Candidates: []ScoredRecallCandidate{{
			Candidate: RecallCandidate{
				MemoryID:   "golden-pref",
				SourceType: "conclusion",
				Content:    "When answering about Goncho recall, prefer evidence-first reports and mention RecallTrace when retrieval is involved.",
				SessionID:  "sess-goncho-golden",
				ScopeID:    "default",
				Importance: 0.9,
			},
			Score: RecallScore{KeywordScore: 1, FinalScore: 1.016393, WhySelected: []string{"final_score=1.016393", "scoring_config=goncho-golden-scoring-v1"}},
		}},
		Selected: []ScoredRecallCandidate{{
			Candidate: RecallCandidate{
				MemoryID:   "golden-pref",
				SourceType: "conclusion",
				Content:    "When answering about Goncho recall, prefer evidence-first reports and mention RecallTrace when retrieval is involved.",
				SessionID:  "sess-goncho-golden",
				ScopeID:    "default",
				Importance: 0.9,
			},
			Score: RecallScore{KeywordScore: 1, FinalScore: 1.016393, WhySelected: []string{"final_score=1.016393", "scoring_config=goncho-golden-scoring-v1", "projection=trace_only"}},
		}},
		Rejected: []RejectedRecallCandidate{{
			Candidate: RecallCandidate{
				MemoryID:   "negative-control",
				SourceType: "conclusion",
				Content:    "When answering about Goncho recall, prefer vague summaries and hide trace details.",
				SessionID:  "sess-goncho-golden",
				ScopeID:    "other",
			},
			Reason:      RecallRejectScopeMismatch,
			WhyRejected: []string{"candidate_scope=other", "query_scope=default"},
		}},
		Warnings: []RecallWarning{},
	}
}

func assertRecallBenchmarkReportFixture(t *testing.T, report RecallBenchmarkReport) {
	t.Helper()
	path := filepath.Join("testdata", "recall_benchmark", "report.golden.json")
	gotRaw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal benchmark report: %v", err)
	}
	if os.Getenv("GORMES_UPDATE_RECALL_BENCHMARK") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create benchmark fixture dir: %v", err)
		}
		if err := os.WriteFile(path, gotRaw, 0o644); err != nil {
			t.Fatalf("write benchmark fixture: %v", err)
		}
		return
	}
	wantRaw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fixture_missing at $: %s", path)
	}
	if err != nil {
		t.Fatalf("read benchmark fixture: %v", err)
	}
	if !bytes.Equal(wantRaw, gotRaw) {
		t.Fatalf("recall_benchmark_report_mismatch at %s", path)
	}
}
