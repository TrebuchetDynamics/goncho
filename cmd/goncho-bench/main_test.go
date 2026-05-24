package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho"
)

func TestRunBeamServiceRecallOracleWritesAbilityReport(t *testing.T) {
	out := filepath.Join(t.TempDir(), "beam-service-report.json")
	if err := run(context.Background(), config{
		BeamServiceOut: out,
		DatabasePath:   filepath.Join(t.TempDir(), "beam-service.db"),
	}); err != nil {
		t.Fatalf("run BEAM service oracle: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read BEAM service report: %v", err)
	}
	var report goncho.RecallBenchmarkReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode BEAM service report: %v", err)
	}
	wantAbilities := []string{"CR", "EO", "IE", "IF", "KU", "MR", "PF", "TR"}
	gotAbilities := make([]string, 0, len(report.Abilities))
	for _, ability := range report.Abilities {
		gotAbilities = append(gotAbilities, ability.Ability)
		if ability.CaseCount != 1 || ability.RecallAt5 != 1 || ability.ContextHitRate != 1 || ability.ProvenanceHitRate != 1 {
			t.Fatalf("ability %s = %+v, want one perfect service-backed fixture", ability.Ability, ability)
		}
	}
	if report.CaseCount != len(wantAbilities) || !slices.Equal(gotAbilities, wantAbilities) {
		t.Fatalf("abilities = %v case_count=%d, want %v", gotAbilities, report.CaseCount, wantAbilities)
	}
	if report.RecallAt5 != 1 || report.RecallAt10 != 1 || report.ContextHitRate != 1 || report.TokenBudgetPassRate != 1 || report.WarningCount != 0 {
		t.Fatalf("BEAM service report = %+v, want perfect deterministic local oracle", report)
	}
}

func TestRunBeamServiceRecallOracleWritesMnemosyneCompatibleArtifacts(t *testing.T) {
	dir := t.TempDir()
	summaryPath := filepath.Join(dir, "beam_e2e_summary.json")
	pairedPath := filepath.Join(dir, "paired_outcomes.jsonl")
	if err := run(context.Background(), config{
		BeamServiceOut:        filepath.Join(dir, "beam-service-report.json"),
		BeamServiceSummaryOut: summaryPath,
		BeamServicePairedOut:  pairedPath,
		BeamServiceConfigID:   "test-beam-service",
		DatabasePath:          filepath.Join(dir, "beam-service.db"),
	}); err != nil {
		t.Fatalf("run BEAM service oracle: %v", err)
	}

	summaryRaw, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	var summary struct {
		Date           string         `json:"date"`
		Metadata       map[string]any `json:"metadata"`
		AbilitySummary map[string]map[string]struct {
			AvgScore float64 `json:"avg_score"`
			Count    int     `json:"count"`
		} `json:"ability_summary"`
	}
	if err := json.Unmarshal(summaryRaw, &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.Date == "" || summary.Metadata["model"] != "goncho-service-recall" || summary.Metadata["judge_model"] != "none" {
		t.Fatalf("summary metadata = %+v date=%q, want Mnemosyne-compatible local recall metadata", summary.Metadata, summary.Date)
	}
	scale := summary.AbilitySummary["100K"]
	wantAbilities := []string{"CR", "EO", "IE", "IF", "KU", "MR", "PF", "TR"}
	for _, ability := range wantAbilities {
		stats, ok := scale[ability]
		if !ok || stats.Count != 1 || stats.AvgScore != 1 {
			t.Fatalf("summary ability %s = %+v ok=%v, want avg_score=1 count=1", ability, stats, ok)
		}
	}
	if overall := scale["OVERALL"]; overall.Count != len(wantAbilities) || overall.AvgScore != 1 {
		t.Fatalf("summary OVERALL = %+v, want avg_score=1 count=%d", overall, len(wantAbilities))
	}

	pairedRaw, err := os.ReadFile(pairedPath)
	if err != nil {
		t.Fatalf("read paired outcomes: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(pairedRaw)), "\n")
	if len(lines) != len(wantAbilities) {
		t.Fatalf("paired rows = %d, want %d: %s", len(lines), len(wantAbilities), pairedRaw)
	}
	seen := map[string]bool{}
	for _, line := range lines {
		var row struct {
			ConfigID       string  `json:"config_id"`
			RunStartedAt   string  `json:"run_started_at"`
			Scale          string  `json:"scale"`
			ConversationID string  `json:"conversation_id"`
			QID            string  `json:"qid"`
			Ability        string  `json:"ability"`
			Score          float64 `json:"score"`
			Correct        bool    `json:"correct"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("decode paired row %q: %v", line, err)
		}
		if row.ConfigID != "test-beam-service" || row.RunStartedAt == "" || row.Scale != "100K" || row.ConversationID != "goncho-service-memoria-fixtures" || !strings.HasPrefix(row.QID, "beam-") || row.Score != 1 || !row.Correct {
			t.Fatalf("paired row = %+v, want Mnemosyne-compatible correct service recall outcome", row)
		}
		seen[row.Ability] = true
	}
	for _, ability := range wantAbilities {
		if !seen[ability] {
			t.Fatalf("paired outcomes missing ability %s in rows %s", ability, pairedRaw)
		}
	}
}

func TestRunLongMemEvalStyleFixtureComputesRetrievalMetrics(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.json")
	if err := run(context.Background(), config{
		DatasetPath:  filepath.Join("testdata", "tiny-longmemeval.jsonl"),
		OutPath:      out,
		DatabasePath: filepath.Join(t.TempDir(), "bench.db"),
		Limit:        10,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report BenchmarkReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Dataset != "tiny-longmemeval" || report.QuestionCount != 3 {
		t.Fatalf("report metadata = %+v", report)
	}
	if report.RecallAt5 != 1 || report.RecallAt10 != 1 || report.RecallAnyAt5 != 1 || report.RecallAnyAt10 != 1 || report.MRR != 1 {
		t.Fatalf("metrics = R@5 %.3f R@10 %.3f any@5 %.3f any@10 %.3f MRR %.3f, want all 1 after lexical ranking", report.RecallAt5, report.RecallAt10, report.RecallAnyAt5, report.RecallAnyAt10, report.MRR)
	}
	if len(report.Questions) != 3 || report.Questions[0].Rank != 1 || report.Questions[1].Rank != 1 || report.Questions[2].Rank != 1 {
		t.Fatalf("question reports = %+v, want deterministic rank-1 hits", report.Questions)
	}
}

func TestGonchoBenchmarkMapsDuplicateContentWithinQuestionPeer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duplicate-content.jsonl")
	raw := "{\"type\":\"meta\",\"dataset\":\"duplicate-content\"}\n" +
		"{\"type\":\"memory\",\"id\":\"wrong-peer-id\",\"peer\":\"p1\",\"content\":\"user: I bought a blue Snaggletooth action figure at a thrift store.\\nassistant: Nice find.\"}\n" +
		"{\"type\":\"question\",\"id\":\"q1\",\"peer\":\"p1\",\"query\":\"What action figure did I buy?\",\"relevant_ids\":[\"wrong-peer-id\"]}\n" +
		"{\"type\":\"memory\",\"id\":\"right-peer-id\",\"peer\":\"p2\",\"content\":\"user: I bought a blue Snaggletooth action figure at a thrift store.\\nassistant: Nice find.\"}\n" +
		"{\"type\":\"question\",\"id\":\"q2\",\"peer\":\"p2\",\"query\":\"What action figure did I buy?\",\"relevant_ids\":[\"right-peer-id\"]}\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write duplicate-content fixture: %v", err)
	}
	out := filepath.Join(t.TempDir(), "report.json")
	if err := run(context.Background(), config{DatasetPath: path, OutPath: out, DatabasePath: filepath.Join(t.TempDir(), "bench.db"), System: "goncho", Limit: 10}); err != nil {
		t.Fatalf("run: %v", err)
	}
	rawReport, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report BenchmarkReport
	if err := json.Unmarshal(rawReport, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.MRR != 1 || len(report.Questions) != 2 || report.Questions[0].Rank != 1 || report.Questions[1].Rank != 1 {
		t.Fatalf("report = %+v, want duplicate content mapped inside each question peer", report)
	}
}

func TestRunLongMemEvalStyleFixtureSupportsTwentyRunLoop(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.json")
	if err := run(context.Background(), config{
		DatasetPath:  filepath.Join("testdata", "tiny-longmemeval.jsonl"),
		OutPath:      out,
		DatabasePath: filepath.Join(t.TempDir(), "bench.db"),
		Limit:        10,
		Runs:         20,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report BenchmarkReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Runs != 20 || report.RecallAt5 != 1 || report.RecallAt10 != 1 || report.RecallAnyAt5 != 1 || report.RecallAnyAt10 != 1 || report.MRR != 1 {
		t.Fatalf("report = %+v, want 20 deterministic rank-1 runs", report)
	}
}

func TestScientificBenchmarkSmokeIncludesBaselinesLeakageAndFailures(t *testing.T) {
	dir := t.TempDir()
	for _, system := range []string{"goncho", "goncho-no-rank", "random", "bm25", "sqlite-fts5"} {
		out := filepath.Join(dir, system+".json")
		failures := filepath.Join(dir, system+"-failures.jsonl")
		if err := run(context.Background(), config{
			DatasetPath:  filepath.Join("testdata", "tiny-longmemeval.jsonl"),
			OutPath:      out,
			FailurePath:  failures,
			DatabasePath: filepath.Join(dir, system+".db"),
			Limit:        10,
			Runs:         2,
			System:       system,
		}); err != nil {
			t.Fatalf("run %s: %v", system, err)
		}
		raw, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("read %s report: %v", system, err)
		}
		var report BenchmarkReport
		if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("decode %s report: %v", system, err)
		}
		if report.System != system || report.Runs != 2 || report.Leakage.QueryInMemory != 0 || report.Leakage.GoldIDInMemory != 0 {
			t.Fatalf("%s report = %+v, want system/runs and no leakage", system, report)
		}
		if _, err := os.Stat(failures); err != nil {
			t.Fatalf("%s failure audit missing: %v", system, err)
		}
	}
}

func TestRunFailsOnLeakageWhenRequested(t *testing.T) {
	path := filepath.Join(t.TempDir(), "leaky.jsonl")
	raw := "{\"type\":\"meta\",\"dataset\":\"leaky\"}\n" +
		"{\"type\":\"memory\",\"id\":\"m1\",\"peer\":\"p\",\"content\":\"The exact query is hidden here.\"}\n" +
		"{\"type\":\"question\",\"id\":\"q1\",\"peer\":\"p\",\"query\":\"exact query\",\"relevant_ids\":[\"m1\"]}\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write leaky fixture: %v", err)
	}
	err := run(context.Background(), config{DatasetPath: path, OutPath: filepath.Join(t.TempDir(), "out.json"), System: "bm25", FailOnLeakage: true})
	if err == nil {
		t.Fatalf("run succeeded, want leakage failure")
	}
}

func TestLoadDatasetRejectsQuestionsWithoutRelevantIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	if err := os.WriteFile(path, []byte(`{"type":"question","id":"q1","query":"missing gold"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write bad fixture: %v", err)
	}
	_, err := loadDataset(path)
	if err == nil {
		t.Fatalf("loadDataset succeeded, want error for question without relevant_ids")
	}
}
