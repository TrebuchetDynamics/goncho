package memorymirror

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBenchmarkTargetsMatchLongMemEvalAndTokenSavingsReferences(t *testing.T) {
	targets := BenchmarkTargets()
	if targets.LongMemEval.QuestionCount != 500 || targets.LongMemEval.EmbeddingModel != "all-MiniLM-L6-v2" {
		t.Fatalf("LongMemEval target = %+v", targets.LongMemEval)
	}
	if targets.LongMemEval.ReferenceRecallAnyAt5 != 0.952 || targets.LongMemEval.ReferenceRecallAnyAt10 != 0.986 || targets.LongMemEval.ReferenceMRR != 0.882 {
		t.Fatalf("reference metrics = %+v", targets.LongMemEval)
	}
	if targets.TokenSavings.PasteFullContextTokensPerYear != 19_500_000 || targets.TokenSavings.TargetTokensPerYear != 170_000 || targets.TokenSavings.LocalEmbeddingCostUSDPerYear != 0 {
		t.Fatalf("token-savings target = %+v", targets.TokenSavings)
	}
}

func TestFrozenLongMemEvalReportMeetsSimilarBenchmarkGate(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "benchmarks", "results", "longmemeval-s-2026-05-20-goncho.json"))
	if err != nil {
		t.Fatalf("read frozen LongMemEval report: %v", err)
	}
	var report LongMemEvalEvidence
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode frozen LongMemEval report: %v", err)
	}
	assessment := AssessLongMemEval(report, BenchmarkTargets().LongMemEval)
	if !assessment.MeetsSimilarGate {
		t.Fatalf("LongMemEval gate failed: %+v", assessment)
	}
	if assessment.RecallAnyAt5Delta <= 0 || assessment.MRRDelta <= 0 {
		t.Fatalf("Goncho should beat reference R@5/MRR, got %+v", assessment)
	}
	if assessment.RecallAnyAt10Delta < -0.01 {
		t.Fatalf("Goncho R@10 must stay within 1pp of reference, got %+v", assessment)
	}
}
