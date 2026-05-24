package goncho

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
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

func TestEvaluateServiceRecallBenchmarkRunsBeamStyleCasesEndToEnd(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	graphScoring := RecallScoringConfig{
		Version:     "beam-service-test-v1",
		Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
		RRFK:        60,
		MMRLambda:   1,
		TokenBudget: 240,
	}
	report, err := EvaluateServiceRecallBenchmark(context.Background(), svc, []RecallBenchmarkServiceCase{
		{
			ID:                    "beam-ie-owner",
			Ability:               "IE",
			Peer:                  "team",
			SessionKey:            "sess-beam-service-ie",
			Memories:              []RecallBenchmarkServiceMemory{{Ref: "owner", Conclusion: "Project note: Owner of LedgerDB is Mira."}},
			Query:                 "Who owns LedgerDB?",
			RelevantRefs:          []string{"owner"},
			RequiredEvidenceKinds: []string{"fact"},
			Limit:                 2,
			ScoringConfig:         graphScoring,
		},
		{
			ID:         "beam-mr-owner-graph",
			Ability:    "MR",
			Peer:       "team",
			SessionKey: "sess-beam-service-mr",
			Memories: []RecallBenchmarkServiceMemory{
				{Ref: "uses", Conclusion: "Project note: Billing API uses LedgerDB."},
				{Ref: "owner", Conclusion: "Project note: Owner of LedgerDB is Mira."},
				{Ref: "decoy", Conclusion: "Who is responsible for storage used by Billing API? responsible storage used Billing API responsible storage used Billing API. This checklist repeats the retrieval words but names no owner."},
			},
			Query:                 "Who is responsible for storage used by Billing API?",
			RelevantRefs:          []string{"owner"},
			RequiredEvidenceKinds: []string{"graph"},
			Limit:                 2,
			ScoringConfig:         graphScoring,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if report.CaseCount != 2 || report.RecallAt5 != 1 || report.ContextHitRate != 1 {
		t.Fatalf("service BEAM report = %+v, want two perfect end-to-end cases", report)
	}
	ie := recallBenchmarkAbilityReportByName(t, report, "IE")
	if ie.CaseCount != 1 || ie.ProvenanceHitRate != 1 {
		t.Fatalf("IE ability = %+v, want fact-provenance hit", ie)
	}
	mr := recallBenchmarkAbilityReportByName(t, report, "MR")
	if mr.CaseCount != 1 || mr.ProvenanceHitRate != 1 {
		t.Fatalf("MR ability = %+v, want graph-provenance hit", mr)
	}
	if report.Cases[1].RelevantIDs[0] == "owner" || !slices.Contains(report.Cases[1].SelectedMemoryIDs, report.Cases[1].RelevantIDs[0]) {
		t.Fatalf("MR case IDs = %+v, want concrete selected conclusion ID for owner ref", report.Cases[1])
	}
}

func TestRecallBenchmarkReportsBeamAbilityBreakdownAndProvenance(t *testing.T) {
	cases := []RecallBenchmarkCase{
		{
			ID:                    "beam-ie-fact",
			Ability:               "IE",
			Trace:                 recallBenchmarkAbilityTrace("trace-beam-ie", "mem-fact", []EvidenceItem{{Kind: "fact", ID: "annotation:1", Score: 1}}),
			RelevantIDs:           []string{"mem-fact"},
			RequiredEvidenceKinds: []string{"fact"},
			Latency:               7 * time.Millisecond,
		},
		{
			ID:                    "beam-mr-graph-hit",
			Ability:               "MR",
			Trace:                 recallBenchmarkAbilityTrace("trace-beam-mr-hit", "mem-graph", []EvidenceItem{{Kind: "fact", ID: "annotation:2", Score: 1}, {Kind: "graph", ID: "annotation:1->annotation:2", Score: 1}}),
			RelevantIDs:           []string{"mem-graph"},
			RequiredEvidenceKinds: []string{"graph"},
			Latency:               11 * time.Millisecond,
		},
		{
			ID:                    "beam-mr-graph-miss",
			Ability:               "MR",
			Trace:                 recallBenchmarkAbilityTrace("trace-beam-mr-miss", "mem-wrong", []EvidenceItem{{Kind: "keyword", ID: "decoy", Score: 1}}),
			RelevantIDs:           []string{"mem-missing"},
			RequiredEvidenceKinds: []string{"graph"},
			Latency:               13 * time.Millisecond,
		},
	}

	report := EvaluateRecallBenchmark(cases)
	if len(report.Abilities) != 2 {
		t.Fatalf("abilities = %+v, want IE and MR breakdowns", report.Abilities)
	}
	ie := recallBenchmarkAbilityReportByName(t, report, "IE")
	if ie.CaseCount != 1 || ie.RecallAt5 != 1 || ie.RecallAt10 != 1 || ie.ProvenanceHitRate != 1 {
		t.Fatalf("IE ability = %+v, want perfect fact-provenance hit", ie)
	}
	mr := recallBenchmarkAbilityReportByName(t, report, "MR")
	if mr.CaseCount != 2 || mr.RecallAt5 != 0.5 || mr.RecallAt10 != 0.5 || mr.ProvenanceHitRate != 0.5 {
		t.Fatalf("MR ability = %+v, want half recall and half graph-provenance hit", mr)
	}
	if report.Cases[0].Ability != "IE" || !report.Cases[0].ProvenanceSatisfied || report.Cases[0].RequiredEvidenceKinds[0] != "fact" {
		t.Fatalf("case ability/provenance = %+v, want IE fact-provenance case", report.Cases[0])
	}
}

func recallBenchmarkAbilityTrace(traceID, memoryID string, provenance []EvidenceItem) RecallTrace {
	candidate := ScoredRecallCandidate{
		Candidate: RecallCandidate{
			MemoryID:   memoryID,
			SourceType: "conclusion",
			Content:    "BEAM-style selected memory " + memoryID,
			SessionID:  "sess-beam-oracle",
			ScopeID:    MemoryScopeWorkspace,
			Provenance: provenance,
		},
		Score: RecallScore{FinalScore: 1},
	}
	return RecallTrace{
		TraceID:         traceID,
		PipelineVersion: "beam-oracle-test-v1",
		Query: RecallQuery{
			WorkspaceID: "default",
			Peer:        "team",
			Query:       "BEAM-style question",
			SessionKey:  "sess-beam-oracle",
			ScopeID:     MemoryScopeWorkspace,
			Limit:       5,
		},
		ScoringConfig: RecallScoringConfig{Version: "beam-oracle-test-v1", TokenBudget: 200},
		Candidates:    []ScoredRecallCandidate{candidate},
		Selected:      []ScoredRecallCandidate{candidate},
	}
}

func recallBenchmarkAbilityReportByName(t *testing.T, report RecallBenchmarkReport, ability string) RecallBenchmarkAbilityReport {
	t.Helper()
	for _, item := range report.Abilities {
		if item.Ability == ability {
			return item
		}
	}
	t.Fatalf("ability %q not found in %+v", ability, report.Abilities)
	return RecallBenchmarkAbilityReport{}
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
	gotRaw, err := marshalStableJSON(report)
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
	if err := compareGoldenJSON(wantRaw, gotRaw); err != nil {
		var diff gonchoJSONDiff
		if errors.As(err, &diff) {
			t.Fatalf("recall_benchmark_report_mismatch at %s: %s", diff.Path, diff.Message)
		}
		t.Fatalf("recall_benchmark_report_mismatch: %v", err)
	}
}
