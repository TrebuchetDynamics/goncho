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
