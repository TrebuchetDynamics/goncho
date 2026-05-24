package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunPinnedBeamSmokeFixtureEmitsEndToEndArtifacts(t *testing.T) {
	dir := t.TempDir()
	rawFixture := filepath.Join("testdata", "beam-smoke", "hf-beam-smoke.jsonl")
	baselineFixture := filepath.Join("testdata", "beam-smoke", "mnemosyne-smoke-paired_outcomes.jsonl")
	pairedPath := filepath.Join(dir, "paired_outcomes.jsonl")
	copyTestFile(t, baselineFixture, pairedPath)

	resultsPath := filepath.Join(dir, "beam_e2e_results.json")
	summaryPath := filepath.Join(dir, "beam_e2e_summary.json")
	if err := run(context.Background(), config{
		BeamConvertIn:         rawFixture,
		BeamServiceResultsOut: resultsPath,
		BeamServiceSummaryOut: summaryPath,
		BeamServicePairedOut:  pairedPath,
		BeamServiceConfigID:   "goncho-smoke",
		DatabasePath:          filepath.Join(dir, "beam-smoke.db"),
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
	}
	decodeTestJSONFile(t, comparisonPath, &comparison)
	if comparison.PairedCount != 1 || comparison.DroppedUnpairedCount != 0 || comparison.ScoreDelta != 0.5 || comparison.CandidateWins != 1 || comparison.Conclusion != "candidate_superior" || comparison.ConclusionReason != "candidate_ci_above_effect_floor" {
		t.Fatalf("BEAM smoke comparison = %+v, want paired candidate-superior smoke verdict", comparison)
	}
	assertBenchFileContains(t, comparisonMDPath, "# BEAM Paired Outcome Comparison")
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
