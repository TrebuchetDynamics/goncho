package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocomoCompareWinnerAndRankDelta(t *testing.T) {
	if normalizedRank(0) != locomoNotFoundRank {
		t.Fatalf("not found rank convention changed")
	}
	if compareWinner(1, 2) != "bm25" || compareWinner(3, 1) != "goncho" || compareWinner(2, 2) != "tie" || compareWinner(locomoNotFoundRank, locomoNotFoundRank) != "both_miss" {
		t.Fatalf("winner classification failed")
	}
	row := locomoComparisonRow{BM25GoldBestRank: 2, GonchoGoldBestRank: 5}
	row.RankDelta = row.GonchoGoldBestRank - row.BM25GoldBestRank
	if row.RankDelta != 3 {
		t.Fatalf("rank delta = %d, want 3", row.RankDelta)
	}
}

func TestLocomoDiagnosticHeuristics(t *testing.T) {
	row := locomoComparisonRow{Winner: "bm25", BM25GoldBestRank: 2, GonchoGoldBestRank: locomoNotFoundRank, Question: "What is current?", GoldMemoryIDs: []string{"m1"}}
	if got := classifyLocomoComparison(row); got != "missing_candidate" {
		t.Fatalf("mode = %s", got)
	}
	row = locomoComparisonRow{Winner: "bm25", BM25GoldBestRank: 2, GonchoGoldBestRank: 5, Question: "What is current?", GoldMemoryIDs: []string{"m1"}}
	if got := classifyLocomoComparison(row); got != "rerank_regression" {
		t.Fatalf("mode = %s", got)
	}
	row = locomoComparisonRow{Winner: "tie", BM25GoldBestRank: 1, GonchoGoldBestRank: 1, Question: "Who said hello?", GoldMemoryIDs: []string{"m1"}}
	if got := classifyLocomoComparison(row); got != "speaker_attribution" {
		t.Fatalf("mode = %s", got)
	}
	row = locomoComparisonRow{Winner: "tie", BM25GoldBestRank: 1, GonchoGoldBestRank: 1, Question: "What happened?", GoldMemoryIDs: []string{"m1", "m2"}}
	if got := classifyLocomoComparison(row); got != "gold_ambiguity" {
		t.Fatalf("mode = %s", got)
	}
}

func TestLocomoComparisonMarkdownJSONLConsistency(t *testing.T) {
	report := locomoReport{BenchmarkName: "LOCOMO", Mode: "retrieval", NoLLMJudge: true, QuestionCount: 2, Systems: []locomoSystemReport{
		{System: "bm25", Questions: 2, RecallAnyAt5: 1, RecallAnyAt10: 1, MRR: 0.75, QuestionsDetail: []locomoQuestionResult{
			{QuestionID: "q1", Category: "temporal_retrieval", Question: "When now?", GoldMemoryIDs: []string{"m1"}, RetrievedIDs: []string{"m1"}, Rank: 1},
			{QuestionID: "q2", Category: "single_hop_retrieval", Question: "What?", GoldMemoryIDs: []string{"m2"}, RetrievedIDs: []string{"x", "m2"}, Rank: 2},
		}},
		{System: "goncho", Questions: 2, RecallAnyAt5: 0.5, RecallAnyAt10: 0.5, MRR: 0.5, QuestionsDetail: []locomoQuestionResult{
			{QuestionID: "q1", Category: "temporal_retrieval", Question: "When now?", GoldMemoryIDs: []string{"m1"}, RetrievedIDs: []string{"x"}, Rank: 0},
			{QuestionID: "q2", Category: "single_hop_retrieval", Question: "What?", GoldMemoryIDs: []string{"m2"}, RetrievedIDs: []string{"m2"}, Rank: 1},
		}},
	}}
	rows, err := compareLocomoSystems(report, "bm25", "goncho")
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if len(rows) != 2 || rows[0].Winner != "bm25" || rows[1].Winner != "goncho" {
		t.Fatalf("rows = %+v", rows)
	}
	dir := t.TempDir()
	jsonl := filepath.Join(dir, "cmp.jsonl")
	md := filepath.Join(dir, "cmp.md")
	if err := writeLocomoComparisonJSONL(jsonl, rows); err != nil {
		t.Fatalf("jsonl: %v", err)
	}
	if err := writeLocomoComparisonMarkdown(md, report, summarizeLocomoComparison(rows), "report.json", jsonl); err != nil {
		t.Fatalf("md: %v", err)
	}
	raw, _ := os.ReadFile(jsonl)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("jsonl line count = %d", len(lines))
	}
	var first locomoComparisonRow
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if first.RankDelta != locomoNotFoundRank-1 {
		t.Fatalf("delta = %d", first.RankDelta)
	}
	mdRaw, _ := os.ReadFile(md)
	text := string(mdRaw)
	for _, needle := range []string{"Winner counts", "BM25 wins", "Goncho wins", "Failure mode"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("markdown missing %s", needle)
		}
	}
}
