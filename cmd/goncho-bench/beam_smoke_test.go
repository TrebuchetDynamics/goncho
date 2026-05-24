package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPinnedBeamSmokeFixtureEmitsEndToEndArtifacts(t *testing.T) {
	dir := t.TempDir()
	rawFixture := filepath.Join("testdata", "beam-smoke", "hf-beam-smoke.jsonl")
	baselineResultsFixture := filepath.Join("testdata", "beam-smoke", "mnemosyne-smoke-beam_e2e_results.json")
	pairedPath := filepath.Join(dir, "paired_outcomes.jsonl")
	if err := run(context.Background(), config{
		BeamPairedResultsIn:       baselineResultsFixture,
		BeamPairedResultsOut:      pairedPath,
		BeamPairedResultsConfigID: "mnemosyne-smoke",
	}); err != nil {
		t.Fatalf("import pinned Mnemosyne BEAM smoke results: %v", err)
	}

	resultsPath := filepath.Join(dir, "beam_e2e_results.json")
	summaryPath := filepath.Join(dir, "beam_e2e_summary.json")
	failuresPath := filepath.Join(dir, "beam_failures.jsonl")
	judgeRequestsPath := filepath.Join(dir, "beam_judge_requests.jsonl")
	if err := run(context.Background(), config{
		BeamConvertIn:               rawFixture,
		BeamServiceResultsOut:       resultsPath,
		BeamServiceSummaryOut:       summaryPath,
		BeamServicePairedOut:        pairedPath,
		BeamServiceFailuresOut:      failuresPath,
		BeamServiceJudgeRequestsOut: judgeRequestsPath,
		BeamServiceConfigID:         "goncho-smoke",
		DatabasePath:                filepath.Join(dir, "beam-smoke.db"),
	}); err != nil {
		t.Fatalf("run pinned BEAM smoke fixture: %v", err)
	}

	var results struct {
		Metadata struct {
			Diagnostics struct {
				Conversion struct {
					SourceSHA256         string `json:"source_sha256"`
					ConvertedJSONLSHA256 string `json:"converted_jsonl_sha256"`
					QuestionCount        int    `json:"question_count"`
				} `json:"conversion"`
				Leakage struct {
					QuestionTextInMemory int `json:"question_text_in_memory"`
					RelevantIDInMemory   int `json:"relevant_id_in_memory"`
					RubricTextInMemory   int `json:"rubric_text_in_memory"`
				} `json:"leakage"`
			} `json:"diagnostics"`
		} `json:"metadata"`
		Results []struct {
			Results []struct {
				QID                string  `json:"qid"`
				Score              float64 `json:"score"`
				RubricContextScore float64 `json:"rubric_context_score"`
				RecallProvenance   struct {
					TopResultVoices map[string]float64 `json:"top_result_voices"`
				} `json:"recall_provenance"`
			} `json:"results"`
		} `json:"results"`
	}
	decodeTestJSONFile(t, resultsPath, &results)
	conversion := results.Metadata.Diagnostics.Conversion
	if conversion.SourceSHA256 == "" || conversion.ConvertedJSONLSHA256 == "" || conversion.QuestionCount != 1 {
		t.Fatalf("BEAM smoke conversion diagnostics = %+v, want source/converted checksums and one question", conversion)
	}
	if leakage := results.Metadata.Diagnostics.Leakage; leakage.QuestionTextInMemory != 0 || leakage.RelevantIDInMemory != 0 || leakage.RubricTextInMemory != 0 {
		t.Fatalf("BEAM smoke leakage diagnostics = %+v, want clean pinned fixture", leakage)
	}
	if len(results.Results) != 1 || len(results.Results[0].Results) != 1 {
		t.Fatalf("BEAM smoke results = %+v, want one conversation with one result", results.Results)
	}
	row := results.Results[0].Results[0]
	if row.QID != "q-mr-ledger-smoke" || row.Score != 1 || row.RubricContextScore != 1 || row.RecallProvenance.TopResultVoices["graph"] == 0 {
		t.Fatalf("BEAM smoke row = %+v, want perfect graph-backed MR recall with rubric context coverage", row)
	}

	var summary struct {
		AbilitySummary map[string]map[string]struct {
			AvgScore float64 `json:"avg_score"`
			Count    int     `json:"count"`
		} `json:"ability_summary"`
	}
	decodeTestJSONFile(t, summaryPath, &summary)
	if got := summary.AbilitySummary["100K"]["MR"]; got.Count != 1 || got.AvgScore != 1 {
		t.Fatalf("BEAM smoke summary MR = %+v, want one perfect MR case", got)
	}
	failureRaw, err := os.ReadFile(failuresPath)
	if err != nil {
		t.Fatalf("read BEAM smoke failure audit: %v", err)
	}
	if len(failureRaw) != 0 {
		t.Fatalf("BEAM smoke failure audit = %q, want empty passing smoke failure artifact", failureRaw)
	}
	judgeRaw, err := os.ReadFile(judgeRequestsPath)
	if err != nil {
		t.Fatalf("read BEAM smoke judge requests: %v", err)
	}
	if !strings.Contains(string(judgeRaw), "RETRIEVED MEMORIES") || !strings.Contains(string(judgeRaw), "QUESTION: Who is responsible for storage used by Billing API?") || strings.Contains(string(judgeRaw), "Mira is responsible for LedgerDB storage.\\n\\nQUESTION") {
		t.Fatalf("BEAM smoke judge request = %s, want answer prompt context/question without ideal answer as prompt prefix", judgeRaw)
	}

	comparisonPath := filepath.Join(dir, "beam-paired-comparison.json")
	comparisonMDPath := filepath.Join(dir, "beam-paired-comparison.md")
	if err := run(context.Background(), config{
		BeamPairedComparePath:             pairedPath,
		BeamPairedBaselineConfigID:        "mnemosyne-smoke",
		BeamPairedCandidateConfigID:       "goncho-smoke",
		BeamPairedCompareJSONOut:          comparisonPath,
		BeamPairedCompareMarkdownOut:      comparisonMDPath,
		BeamPairedCompareBootstrapSamples: 200,
	}); err != nil {
		t.Fatalf("compare pinned BEAM smoke outcomes: %v", err)
	}
	var comparison struct {
		PairedCount          int     `json:"paired_count"`
		ScoreDelta           float64 `json:"score_delta"`
		Conclusion           string  `json:"conclusion"`
		ConclusionReason     string  `json:"conclusion_reason"`
		CandidateWins        int     `json:"candidate_wins"`
		DroppedUnpairedCount int     `json:"dropped_unpaired_count"`
		Rows                 []struct {
			MatchKey     string `json:"match_key"`
			BaselineQID  string `json:"baseline_qid"`
			CandidateQID string `json:"candidate_qid"`
		} `json:"rows"`
	}
	decodeTestJSONFile(t, comparisonPath, &comparison)
	if comparison.PairedCount != 1 || comparison.DroppedUnpairedCount != 0 || comparison.ScoreDelta != 0.5 || comparison.CandidateWins != 1 || comparison.Conclusion != "candidate_superior" || comparison.ConclusionReason != "candidate_ci_above_effect_floor" {
		t.Fatalf("BEAM smoke comparison = %+v, want paired candidate-superior smoke verdict", comparison)
	}
	if len(comparison.Rows) != 1 || comparison.Rows[0].MatchKey != "question" || comparison.Rows[0].BaselineQID != "conv-beam-smoke:q0" || comparison.Rows[0].CandidateQID != "q-mr-ledger-smoke" {
		t.Fatalf("BEAM smoke comparison rows = %+v, want nested Mnemosyne qid paired to Goncho source qid by question", comparison.Rows)
	}
	assertBenchFileContains(t, comparisonMDPath, "# BEAM Paired Outcome Comparison")
}

func TestBenchBeamSmokeTargetImportsNestedMnemosyneResults(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "BEAM_SMOKE_BASELINE_RESULTS") || !strings.Contains(text, "--beam-paired-results-in $(BEAM_SMOKE_BASELINE_RESULTS)") || !strings.Contains(text, "--beam-paired-results-out ./artifacts/beam-smoke/paired_outcomes.jsonl") {
		t.Fatalf("bench-beam-smoke target must import nested Mnemosyne BEAM results before paired comparison:\n%s", text)
	}
}

func copyTestFile(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open %s: %v", src, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create %s: %v", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		t.Fatalf("copy %s to %s: %v", src, dst, err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close %s: %v", dst, err)
	}
}

func decodeTestJSONFile(t *testing.T, path string, out any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
