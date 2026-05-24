package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBeamPairedComparisonWritesBootstrapReport(t *testing.T) {
	dir := t.TempDir()
	pairedPath := filepath.Join(dir, "paired_outcomes.jsonl")
	pairedRows := strings.Join([]string{
		`{"config_id":"mnemosyne-v3","run_started_at":"2026-05-24T00:00:00Z","scale":"100K","conversation_id":"conv-a","qid":"q-ie","ability":"IE","score":0.5,"correct":true}`,
		`{"config_id":"goncho-current","run_started_at":"2026-05-24T00:01:00Z","scale":"100K","conversation_id":"conv-a","qid":"q-ie","ability":"IE","score":1,"correct":true}`,
		`{"config_id":"mnemosyne-v3","run_started_at":"2026-05-24T00:00:00Z","scale":"100K","conversation_id":"conv-a","qid":"q-mr","ability":"MR","score":1,"correct":true}`,
		`{"config_id":"goncho-current","run_started_at":"2026-05-24T00:01:00Z","scale":"100K","conversation_id":"conv-a","qid":"q-mr","ability":"MR","score":0,"correct":false}`,
		`{"config_id":"mnemosyne-v3","run_started_at":"2026-05-24T00:00:00Z","scale":"500K","conversation_id":"conv-b","qid":"q-tr","ability":"TR","score":0,"correct":false}`,
		`{"config_id":"goncho-current","run_started_at":"2026-05-24T00:01:00Z","scale":"500K","conversation_id":"conv-b","qid":"q-tr","ability":"TR","score":1,"correct":true}`,
		`{"config_id":"goncho-current","run_started_at":"2026-05-24T00:01:00Z","scale":"500K","conversation_id":"conv-b","qid":"q-unpaired","ability":"ABS","score":1,"correct":true}`,
	}, "\n") + "\n"
	if err := os.WriteFile(pairedPath, []byte(pairedRows), 0o644); err != nil {
		t.Fatalf("write paired outcomes: %v", err)
	}
	jsonOut := filepath.Join(dir, "beam-paired-comparison.json")
	mdOut := filepath.Join(dir, "beam-paired-comparison.md")

	if err := run(context.Background(), config{
		BeamPairedComparePath:             pairedPath,
		BeamPairedBaselineConfigID:        "mnemosyne-v3",
		BeamPairedCandidateConfigID:       "goncho-current",
		BeamPairedCompareJSONOut:          jsonOut,
		BeamPairedCompareMarkdownOut:      mdOut,
		BeamPairedCompareBootstrapSamples: 200,
	}); err != nil {
		t.Fatalf("run BEAM paired comparison: %v", err)
	}

	raw, err := os.ReadFile(jsonOut)
	if err != nil {
		t.Fatalf("read paired comparison JSON: %v", err)
	}
	var report struct {
		BaselineConfigID     string  `json:"baseline_config_id"`
		CandidateConfigID    string  `json:"candidate_config_id"`
		PairedCount          int     `json:"paired_count"`
		DroppedUnpairedCount int     `json:"dropped_unpaired_count"`
		BaselineWins         int     `json:"baseline_wins"`
		CandidateWins        int     `json:"candidate_wins"`
		Ties                 int     `json:"ties"`
		BaselineAvgScore     float64 `json:"baseline_avg_score"`
		CandidateAvgScore    float64 `json:"candidate_avg_score"`
		ScoreDelta           float64 `json:"score_delta"`
		BootstrapSamples     int     `json:"bootstrap_samples"`
		ScoreDeltaCI95       struct {
			Lower float64 `json:"lower"`
			Upper float64 `json:"upper"`
		} `json:"score_delta_ci95"`
		ByAbility map[string]struct {
			PairedCount   int     `json:"paired_count"`
			ScoreDelta    float64 `json:"score_delta"`
			BaselineWins  int     `json:"baseline_wins"`
			CandidateWins int     `json:"candidate_wins"`
		} `json:"by_ability"`
		Rows []struct {
			Scale          string  `json:"scale"`
			ConversationID string  `json:"conversation_id"`
			QID            string  `json:"qid"`
			Ability        string  `json:"ability"`
			BaselineScore  float64 `json:"baseline_score"`
			CandidateScore float64 `json:"candidate_score"`
			ScoreDelta     float64 `json:"score_delta"`
			Winner         string  `json:"winner"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode paired comparison JSON: %v", err)
	}
	if report.BaselineConfigID != "mnemosyne-v3" || report.CandidateConfigID != "goncho-current" {
		t.Fatalf("comparison config IDs = %q/%q, want mnemosyne-v3/goncho-current", report.BaselineConfigID, report.CandidateConfigID)
	}
	if report.PairedCount != 3 || report.DroppedUnpairedCount != 1 || report.CandidateWins != 2 || report.BaselineWins != 1 || report.Ties != 0 {
		t.Fatalf("comparison counts = %+v, want 3 paired, 1 unpaired, 2 candidate wins, 1 baseline win", report)
	}
	if report.BaselineAvgScore != 0.5 || report.CandidateAvgScore != 0.6667 || report.ScoreDelta != 0.1667 {
		t.Fatalf("comparison scores = baseline %.4f candidate %.4f delta %.4f, want 0.5000/0.6667/+0.1667", report.BaselineAvgScore, report.CandidateAvgScore, report.ScoreDelta)
	}
	if report.BootstrapSamples != 200 || report.ScoreDeltaCI95.Lower >= report.ScoreDeltaCI95.Upper {
		t.Fatalf("bootstrap CI = %+v samples=%d, want deterministic non-empty 95%% CI", report.ScoreDeltaCI95, report.BootstrapSamples)
	}
	if got := report.ByAbility["MR"]; got.PairedCount != 1 || got.ScoreDelta != -1 || got.BaselineWins != 1 || got.CandidateWins != 0 {
		t.Fatalf("MR ability comparison = %+v, want baseline-only win with -1 delta", got)
	}
	if len(report.Rows) != 3 || report.Rows[0].QID != "q-ie" || report.Rows[0].Winner != "candidate" {
		t.Fatalf("paired rows = %+v, want stable paired rows with candidate q-ie win first", report.Rows)
	}
	assertBenchFileContains(t, mdOut, "# BEAM Paired Outcome Comparison")
	assertBenchFileContains(t, mdOut, "- Baseline config: `mnemosyne-v3`")
	assertBenchFileContains(t, mdOut, "- Candidate config: `goncho-current`")
	assertBenchFileContains(t, mdOut, "| OVERALL | 3 | 0.5000 | 0.6667 | +0.1667 | 2 | 1 | 0 |")
}
