package goncho

import (
	"context"
	"strings"
	"testing"
)

func TestEvalRegistryConvertsKnownMissIntoStructuredCandidate(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	result, err := svc.RecordEvalFailures(ctx, EvalRegistryInput{
		BenchmarkName: "locomo-smoke",
		RunID:         "run-2026-05-25",
		Failures: []EvalFailure{
			{
				QuestionID:         "q-auth-owner",
				Category:           "single_hop_retrieval",
				Query:              "Who owns authentication?",
				ExpectedMemoryIDs:  []string{"mem-auth-owner"},
				RetrievedMemoryIDs: []string{"mem-auth-decoy"},
				TopHitPreview:      "authentication checklist repeats query terms but names no owner",
				FailureBucket:      "lexical_decoy",
			},
		},
	})
	if err != nil {
		t.Fatalf("RecordEvalFailures: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates = %+v, want one structured candidate", result.Candidates)
	}
	candidate := result.Candidates[0]
	if candidate.Kind != EvalCandidateQueryExpansionHint || candidate.Status != EvalCandidateOpen || candidate.QuestionID != "q-auth-owner" || candidate.BenchmarkName != "locomo-smoke" {
		t.Fatalf("candidate = %+v, want query-expansion open q-auth-owner", candidate)
	}
	if !strings.Contains(candidate.Rationale, "lexical_decoy") || len(candidate.EvidenceIDs) == 0 || candidate.EvidenceIDs[0] != "eval:locomo-smoke:run-2026-05-25:q-auth-owner" {
		t.Fatalf("candidate evidence/rationale = %+v", candidate)
	}
	listed, err := svc.ListEvalCandidates(ctx, EvalCandidateQuery{BenchmarkName: "locomo-smoke", Status: EvalCandidateOpen})
	if err != nil {
		t.Fatalf("ListEvalCandidates: %v", err)
	}
	if len(listed.Candidates) != 1 || listed.Candidates[0].ID != candidate.ID {
		t.Fatalf("listed = %+v, want persisted candidate %s", listed.Candidates, candidate.ID)
	}
}

func TestRecallFeedbackLabelsWriteReviewEvidenceWithoutPromotingClaims(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-feedback", SessionKey: "sess-feedback", Conclusion: "Active memory should remain unchanged."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	before := countExportRows(t, svc, `SELECT COUNT(*) FROM goncho_conclusions`)

	feedback, err := svc.RecordRecallFeedback(ctx, RecallFeedbackParams{
		WorkspaceID: "default",
		Peer:        "peer-feedback",
		SessionKey:  "sess-feedback",
		TraceID:     "trace-feedback-1",
		Query:       "who owns unsafe deployment?",
		Label:       RecallFeedbackUnsafe,
		MemoryID:    "conclusion:42",
		Reason:      "retrieved unsafe deployment advice",
		SubmittedBy: "human:ops",
	})
	if err != nil {
		t.Fatalf("RecordRecallFeedback: %v", err)
	}
	if feedback.Label != RecallFeedbackUnsafe || feedback.ReviewItemID == "" || feedback.Status != RecallFeedbackRecorded {
		t.Fatalf("feedback = %+v, want recorded unsafe review evidence", feedback)
	}
	if after := countExportRows(t, svc, `SELECT COUNT(*) FROM goncho_conclusions`); after != before {
		t.Fatalf("conclusion count changed from %d to %d; feedback must not promote active claims", before, after)
	}
	open, err := svc.ListReviewItems(ctx, ReviewQuery{PeerID: "peer-feedback", SessionKey: "sess-feedback", Status: ReviewStatusOpen})
	if err != nil {
		t.Fatalf("ListReviewItems: %v", err)
	}
	if len(open.Items) != 1 || !strings.Contains(open.Items[0].Reason, "unsafe") || !strings.Contains(strings.Join(open.Items[0].EvidenceIDs, " "), "trace-feedback-1") {
		t.Fatalf("review items = %+v, want unsafe feedback evidence", open.Items)
	}
	labels, err := svc.ListRecallFeedback(ctx, RecallFeedbackQuery{TraceID: "trace-feedback-1"})
	if err != nil {
		t.Fatalf("ListRecallFeedback: %v", err)
	}
	if len(labels.Items) != 1 || labels.Items[0].ReviewItemID != feedback.ReviewItemID {
		t.Fatalf("feedback labels = %+v, want persisted label", labels.Items)
	}
}

func TestBenchmarkTrendReportComparesBranchToFrozenBaseline(t *testing.T) {
	report := BuildBenchmarkTrendReport(BenchmarkTrendInput{
		BaselineID:  "frozen-smoke-v1",
		CandidateID: "current-branch",
		Metrics: []BenchmarkMetricComparison{
			{Metric: "recall_any_at_5", Baseline: 0.80, Current: 0.78, Tolerance: 0.03},
			{Metric: "mrr", Baseline: 0.70, Current: 0.61, Tolerance: 0.02},
		},
	})
	if report.BaselineID != "frozen-smoke-v1" || report.CandidateID != "current-branch" || report.Status != "regressed" {
		t.Fatalf("trend report = %+v, want regressed branch-vs-baseline report", report)
	}
	if len(report.Gates) != 2 || report.Gates[0].Metric != "recall_any_at_5" || !report.Gates[0].Pass || report.Gates[1].Pass {
		t.Fatalf("gates = %+v, want first pass and second fail", report.Gates)
	}
}

func TestRegressionGateRejectsMetricDropAndAcceptsNoiseWithinTolerance(t *testing.T) {
	rejected := EvaluateRegressionGate(RegressionGateInput{
		Metric:    "recall_any_at_5",
		Baseline:  0.80,
		Current:   0.72,
		Tolerance: 0.03,
	})
	if rejected.Pass || rejected.Drop != 0.08 || !strings.Contains(rejected.Reason, "exceeds tolerance") {
		t.Fatalf("rejected gate = %+v, want failure beyond tolerance", rejected)
	}
	accepted := EvaluateRegressionGate(RegressionGateInput{
		Metric:    "recall_any_at_5",
		Baseline:  0.80,
		Current:   0.785,
		Tolerance: 0.03,
	})
	if !accepted.Pass || accepted.Drop != 0.015 || !strings.Contains(accepted.Reason, "within tolerance") {
		t.Fatalf("accepted gate = %+v, want pass within tolerance", accepted)
	}
}
