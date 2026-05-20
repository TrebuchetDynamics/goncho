package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
