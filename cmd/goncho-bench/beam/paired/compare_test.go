package paired

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBeamPairedResultsImportPairsMnemosyneQIDsByQuestion(t *testing.T) {
	dir := t.TempDir()
	resultsPath := filepath.Join(dir, "mnemosyne-beam_e2e_results.json")
	nestedResults := `{
  "metadata": {
    "config_id": "mnemosyne-v3",
    "run_started_at": "2026-05-24T00:00:00Z"
  },
  "results": [
    {
      "conversation_id": "conv-beam-real",
      "scale": "100K",
      "results": [
        {
          "qid": "conv-beam-real:q0",
          "ability": "IE",
          "question": "Who owns LedgerDB?",
          "score": 0.25
        }
      ]
    }
  ]
}
`
	if err := os.WriteFile(resultsPath, []byte(nestedResults), 0o644); err != nil {
		t.Fatalf("write nested Mnemosyne BEAM results: %v", err)
	}
	pairedPath := filepath.Join(dir, "paired_outcomes.jsonl")
	if err := AppendPairedOutcomesFromResults(Config{
		ResultsIn:       resultsPath,
		ResultsOut:      pairedPath,
		ResultsConfigID: "mnemosyne-v3",
	}); err != nil {
		t.Fatalf("import nested Mnemosyne BEAM results as paired outcomes: %v", err)
	}
	pairedRaw, err := os.ReadFile(pairedPath)
	if err != nil {
		t.Fatalf("read imported paired outcomes: %v", err)
	}
	var importedRow struct {
		SourcePath   string `json:"source_path"`
		SourceSHA256 string `json:"source_sha256"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(pairedRaw), &importedRow); err != nil {
		t.Fatalf("decode imported paired outcome row: %v", err)
	}
	if importedRow.SourcePath != resultsPath || importedRow.SourceSHA256 != checksumBytesSHA256([]byte(nestedResults)) {
		t.Fatalf("imported paired outcome source = %+v, want nested result path and checksum", importedRow)
	}
	candidateRow := `{"config_id":"goncho-current","run_started_at":"2026-05-24T00:01:00Z","scale":"100K","conversation_id":"conv-beam-real","qid":"q-source-beam-id","ability":"IE","question":"Who owns LedgerDB?","score":1,"correct":true}` + "\n"
	file, err := os.OpenFile(pairedPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open paired outcomes for candidate append: %v", err)
	}
	if _, err := file.WriteString(candidateRow); err != nil {
		_ = file.Close()
		t.Fatalf("append candidate paired outcome: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close paired outcomes: %v", err)
	}
	jsonOut := filepath.Join(dir, "beam-paired-real-qids.json")
	if err := RunPairedComparison(Config{
		ComparePath:             pairedPath,
		BaselineConfigID:        "mnemosyne-v3",
		CandidateConfigID:       "goncho-current",
		CompareJSONOut:          jsonOut,
		CompareBootstrapSamples: 200,
	}); err != nil {
		t.Fatalf("compare question-key paired BEAM outcomes: %v", err)
	}
	var report struct {
		PairedCount          int     `json:"paired_count"`
		DroppedUnpairedCount int     `json:"dropped_unpaired_count"`
		ScoreDelta           float64 `json:"score_delta"`
		Rows                 []struct {
			MatchKey             string `json:"match_key"`
			Question             string `json:"question"`
			BaselineQID          string `json:"baseline_qid"`
			CandidateQID         string `json:"candidate_qid"`
			BaselineSourcePath   string `json:"baseline_source_path"`
			BaselineSourceSHA256 string `json:"baseline_source_sha256"`
			QID                  string `json:"qid"`
		} `json:"rows"`
	}
	decodeTestJSONFile(t, jsonOut, &report)
	if report.PairedCount != 1 || report.DroppedUnpairedCount != 0 || report.ScoreDelta != 0.75 || len(report.Rows) != 1 {
		t.Fatalf("question-key paired report = %+v, want one paired +0.75 comparison", report)
	}
	row := report.Rows[0]
	if row.MatchKey != "question" || row.Question != "Who owns LedgerDB?" || row.BaselineQID != "conv-beam-real:q0" || row.CandidateQID != "q-source-beam-id" || row.QID != "q-source-beam-id" {
		t.Fatalf("question-key paired row = %+v, want visible qid mismatch matched by question", row)
	}
	if row.BaselineSourcePath != resultsPath || row.BaselineSourceSHA256 != checksumBytesSHA256([]byte(nestedResults)) {
		t.Fatalf("question-key paired source = %+v, want baseline nested result provenance", row)
	}
}

func TestRunBeamPairedComparisonRejectsAmbiguousQuestionFallback(t *testing.T) {
	dir := t.TempDir()
	pairedPath := filepath.Join(dir, "paired_outcomes.jsonl")
	pairedRows := strings.Join([]string{
		`{"config_id":"mnemosyne-v3","run_started_at":"2026-05-24T00:00:00Z","scale":"100K","conversation_id":"conv-ambiguous","qid":"conv-ambiguous:q0","ability":"IE","question":"Who owns LedgerDB?","score":0.25,"correct":false}`,
		`{"config_id":"goncho-current","run_started_at":"2026-05-24T00:01:00Z","scale":"100K","conversation_id":"conv-ambiguous","qid":"q-source-a","ability":"IE","question":"Who owns LedgerDB?","score":1,"correct":true}`,
		`{"config_id":"goncho-current","run_started_at":"2026-05-24T00:01:00Z","scale":"100K","conversation_id":"conv-ambiguous","qid":"q-source-b","ability":"IE","question":"Who owns LedgerDB?","score":0,"correct":false}`,
	}, "\n") + "\n"
	if err := os.WriteFile(pairedPath, []byte(pairedRows), 0o644); err != nil {
		t.Fatalf("write ambiguous paired outcomes: %v", err)
	}
	err := RunPairedComparison(Config{
		ComparePath:             pairedPath,
		BaselineConfigID:        "mnemosyne-v3",
		CandidateConfigID:       "goncho-current",
		CompareJSONOut:          filepath.Join(dir, "beam-paired-ambiguous.json"),
		CompareBootstrapSamples: 200,
	})
	if err == nil || !strings.Contains(err.Error(), "ambiguous BEAM paired question-key fallback") {
		t.Fatalf("ambiguous question fallback error = %v, want fail-closed pairing diagnostic", err)
	}
}

func TestRunBeamPairedComparisonReportsSuperiorityVerdict(t *testing.T) {
	dir := t.TempDir()
	pairedPath := filepath.Join(dir, "paired_outcomes.jsonl")
	pairedRows := strings.Join([]string{
		`{"config_id":"mnemosyne-v3","run_started_at":"2026-05-24T00:00:00Z","scale":"100K","conversation_id":"conv-a","qid":"q-ie-1","ability":"IE","score":0.7,"correct":true}`,
		`{"config_id":"goncho-current","run_started_at":"2026-05-24T00:01:00Z","scale":"100K","conversation_id":"conv-a","qid":"q-ie-1","ability":"IE","score":0.8,"correct":true}`,
		`{"config_id":"mnemosyne-v3","run_started_at":"2026-05-24T00:00:00Z","scale":"100K","conversation_id":"conv-a","qid":"q-ie-2","ability":"IE","score":0.6,"correct":true}`,
		`{"config_id":"goncho-current","run_started_at":"2026-05-24T00:01:00Z","scale":"100K","conversation_id":"conv-a","qid":"q-ie-2","ability":"IE","score":0.7,"correct":true}`,
		`{"config_id":"mnemosyne-v3","run_started_at":"2026-05-24T00:00:00Z","scale":"100K","conversation_id":"conv-a","qid":"q-mr-1","ability":"MR","score":0.5,"correct":true}`,
		`{"config_id":"goncho-current","run_started_at":"2026-05-24T00:01:00Z","scale":"100K","conversation_id":"conv-a","qid":"q-mr-1","ability":"MR","score":0.6,"correct":true}`,
	}, "\n") + "\n"
	if err := os.WriteFile(pairedPath, []byte(pairedRows), 0o644); err != nil {
		t.Fatalf("write paired outcomes: %v", err)
	}
	jsonOut := filepath.Join(dir, "beam-paired-verdict.json")
	mdOut := filepath.Join(dir, "beam-paired-verdict.md")
	if err := RunPairedComparison(Config{
		ComparePath:             pairedPath,
		BaselineConfigID:        "mnemosyne-v3",
		CandidateConfigID:       "goncho-current",
		CompareJSONOut:          jsonOut,
		CompareMarkdownOut:      mdOut,
		CompareBootstrapSamples: 200,
	}); err != nil {
		t.Fatalf("run BEAM paired verdict comparison: %v", err)
	}

	raw, err := os.ReadFile(jsonOut)
	if err != nil {
		t.Fatalf("read paired verdict JSON: %v", err)
	}
	var report struct {
		EffectSizeFloor  float64 `json:"effect_size_floor"`
		Conclusion       string  `json:"conclusion"`
		ConclusionReason string  `json:"conclusion_reason"`
		ScoreDeltaCI95   struct {
			Lower float64 `json:"lower"`
			Upper float64 `json:"upper"`
		} `json:"score_delta_ci95"`
		ByAbility map[string]struct {
			Conclusion       string `json:"conclusion"`
			ConclusionReason string `json:"conclusion_reason"`
		} `json:"by_ability"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode paired verdict JSON: %v", err)
	}
	if report.EffectSizeFloor != 0.02 || report.Conclusion != "candidate_superior" || report.ConclusionReason != "candidate_ci_above_effect_floor" {
		t.Fatalf("paired verdict = floor %.4f conclusion %q reason %q, want candidate superiority above 2pp noise floor", report.EffectSizeFloor, report.Conclusion, report.ConclusionReason)
	}
	if report.ScoreDeltaCI95.Lower <= report.EffectSizeFloor {
		t.Fatalf("paired verdict CI = %+v, want lower bound above effect floor %.4f", report.ScoreDeltaCI95, report.EffectSizeFloor)
	}
	if got := report.ByAbility["IE"]; got.Conclusion != "candidate_superior" || got.ConclusionReason != "candidate_delta_above_effect_floor" {
		t.Fatalf("IE ability verdict = %+v, want candidate superiority verdict", got)
	}
	assertBenchFileContains(t, mdOut, "- Verdict: `candidate_superior` (`candidate_ci_above_effect_floor`)")
	assertBenchFileContains(t, mdOut, "- Effect-size floor: `0.0200`")
}

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

	if err := RunPairedComparison(Config{
		ComparePath:             pairedPath,
		BaselineConfigID:        "mnemosyne-v3",
		CandidateConfigID:       "goncho-current",
		CompareJSONOut:          jsonOut,
		CompareMarkdownOut:      mdOut,
		CompareBootstrapSamples: 200,
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
