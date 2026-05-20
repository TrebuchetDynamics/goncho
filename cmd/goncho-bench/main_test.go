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
	if report.RecallAt5 != 1 || report.RecallAt10 != 1 || report.MRR != 0.6111 {
		t.Fatalf("metrics = R@5 %.3f R@10 %.3f MRR %.3f, want 1/1/0.6111", report.RecallAt5, report.RecallAt10, report.MRR)
	}
	if len(report.Questions) != 3 || report.Questions[0].Rank != 1 || report.Questions[1].Rank != 2 || report.Questions[2].Rank != 3 {
		t.Fatalf("question reports = %+v, want deterministic real ranks 1/2/3", report.Questions)
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
