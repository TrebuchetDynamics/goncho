package memorymirror

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	if targets.LongMemEval.ReproductionCommand != "make bench-longmemeval-s" || targets.LongMemEval.FrozenReportPath != "docs/benchmarks/results/longmemeval-s-2026-05-20-goncho.json" {
		t.Fatalf("reproducibility target = %+v", targets.LongMemEval)
	}
	if targets.LongMemEval.ClaimScope != "retrieval_only_no_llm_reader_or_judge" {
		t.Fatalf("claim scope = %q", targets.LongMemEval.ClaimScope)
	}
	if targets.TokenSavings.PasteFullContextTokensPerYear != 19_500_000 || targets.TokenSavings.TargetTokensPerYear != 170_000 || targets.TokenSavings.LocalEmbeddingCostUSDPerYear != 0 {
		t.Fatalf("token-savings target = %+v", targets.TokenSavings)
	}
}

func TestFrozenLongMemEvalReportMeetsSimilarBenchmarkGate(t *testing.T) {
	target := BenchmarkTargets().LongMemEval
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(target.FrozenReportPath)))
	if err != nil {
		t.Fatalf("read frozen LongMemEval report: %v", err)
	}
	var report LongMemEvalEvidence
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode frozen LongMemEval report: %v", err)
	}
	assessment := AssessLongMemEval(report, target)
	if !assessment.MeetsSimilarGate {
		t.Fatalf("LongMemEval gate failed: %+v", assessment)
	}
	if assessment.RecallAnyAt5Delta <= 0 || assessment.MRRDelta <= 0 {
		t.Fatalf("Goncho should beat reference R@5/MRR, got %+v", assessment)
	}
	if assessment.RecallAnyAt10Delta < -0.01 {
		t.Fatalf("Goncho R@10 must stay within 1pp of reference, got %+v", assessment)
	}
	if assessment.ReproductionCommand != "make bench-longmemeval-s" || assessment.ClaimScope != "retrieval_only_no_llm_reader_or_judge" {
		t.Fatalf("assessment reproducibility/caveat = %+v", assessment)
	}
}

func TestLongMemEvalReportDocumentsReproductionAndAvoidsFalseComparison(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "benchmarks", "longmemeval-s-2026-05-20.md"))
	if err != nil {
		t.Fatalf("read LongMemEval report doc: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"make bench-longmemeval-s",
		"retrieval-only, not end-to-end QA",
		"Goncho beats the cited BM25-only reference on recall_any@5, recall_any@10, and MRR.",
		"Goncho beats the cited BM25+Vector reference on recall_any@5 and MRR, but trails it on recall_any@10.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("LongMemEval report missing %q", want)
		}
	}
	if strings.Contains(text, "Goncho is slightly below the cited BM25-only reference on recall_any@10") {
		t.Fatalf("LongMemEval report still contains false BM25-only R@10 comparison")
	}
}
