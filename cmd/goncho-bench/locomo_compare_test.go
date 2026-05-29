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
	if compareWinner(1, 2) != "a" || compareWinner(3, 1) != "b" || compareWinner(2, 2) != "tie" || compareWinner(locomoNotFoundRank, locomoNotFoundRank) != "both_miss" {
		t.Fatalf("winner classification failed")
	}
	row := locomoComparisonRow{AGoldBestRank: 2, BGoldBestRank: 5}
	row.RankDelta = row.BGoldBestRank - row.AGoldBestRank
	if row.RankDelta != 3 {
		t.Fatalf("rank delta = %d, want 3", row.RankDelta)
	}
}

func TestLocomoDeltaBuckets(t *testing.T) {
	cases := []struct {
		name string
		row  locomoComparisonRow
		want string
	}{
		{name: "a only hit", row: locomoComparisonRow{Winner: "a", AGoldBestRank: 2, BGoldBestRank: locomoNotFoundRank}, want: "a_only_hit"},
		{name: "b only hit", row: locomoComparisonRow{Winner: "b", AGoldBestRank: locomoNotFoundRank, BGoldBestRank: 2}, want: "b_only_hit"},
		{name: "a rank better", row: locomoComparisonRow{Winner: "a", AGoldBestRank: 2, BGoldBestRank: 5}, want: "a_rank_better"},
		{name: "b rank better", row: locomoComparisonRow{Winner: "b", AGoldBestRank: 5, BGoldBestRank: 2}, want: "b_rank_better"},
		{name: "same rank", row: locomoComparisonRow{Winner: "tie", AGoldBestRank: 1, BGoldBestRank: 1}, want: "same_rank"},
		{name: "both miss", row: locomoComparisonRow{Winner: "both_miss", AGoldBestRank: locomoNotFoundRank, BGoldBestRank: locomoNotFoundRank}, want: "both_miss"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyLocomoDeltaBucket(tc.row); got != tc.want {
				t.Fatalf("bucket = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestLocomoComparisonMarkdownJSONLConsistency(t *testing.T) {
	report := locomoReport{BenchmarkName: "LOCOMO", Mode: "retrieval", NoLLMJudge: true, QuestionCount: 2, Systems: []locomoSystemReport{
		{System: "goncho", Questions: 2, RecallAnyAt5: 1, RecallAnyAt10: 1, MRR: 0.75, QuestionsDetail: []locomoQuestionResult{
			{QuestionID: "q1", Category: "temporal_retrieval", Question: "When now?", GoldMemoryIDs: []string{"m1"}, RetrievedIDs: []string{"m1"}, Rank: 1},
			{QuestionID: "q2", Category: "single_hop_retrieval", Question: "What?", GoldMemoryIDs: []string{"m2"}, RetrievedIDs: []string{"x", "m2"}, Rank: 2},
		}},
		{System: "goncho-recall", Questions: 2, RecallAnyAt5: 0.5, RecallAnyAt10: 0.5, MRR: 0.5, QuestionsDetail: []locomoQuestionResult{
			{QuestionID: "q1", Category: "temporal_retrieval", Question: "When now?", GoldMemoryIDs: []string{"m1"}, RetrievedIDs: []string{"x"}, Rank: 0},
			{QuestionID: "q2", Category: "single_hop_retrieval", Question: "What?", GoldMemoryIDs: []string{"m2"}, RetrievedIDs: []string{"m2"}, Rank: 1},
		}},
	}}
	rows, err := compareLocomoSystems(report, "goncho", "goncho-recall")
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if len(rows) != 2 || rows[0].ASystem != "goncho" || rows[0].BSystem != "goncho-recall" || rows[0].Winner != "a" || rows[0].DeltaBucket != "a_only_hit" || rows[1].Winner != "b" || rows[1].DeltaBucket != "b_rank_better" {
		t.Fatalf("rows = %+v", rows)
	}
	dir := t.TempDir()
	jsonl := filepath.Join(dir, "cmp.jsonl")
	md := filepath.Join(dir, "cmp.md")
	if err := writeLocomoComparisonJSONL(jsonl, rows); err != nil {
		t.Fatalf("jsonl: %v", err)
	}
	summary := summarizeLocomoComparison("goncho", "goncho-recall", rows)
	if err := writeLocomoComparisonMarkdown(md, report, summary, "report.json", jsonl); err != nil {
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
	if first.RankDelta != locomoNotFoundRank-1 || first.AGoldBestRank != 1 || first.BGoldBestRank != locomoNotFoundRank {
		t.Fatalf("first row = %+v", first)
	}
	mdRaw, _ := os.ReadFile(md)
	text := string(mdRaw)
	for _, needle := range []string{"LOCOMO Paired Delta Audit", "`goncho`", "`goncho-recall`", "Delta bucket counts", "Category × delta bucket", "a_only_hit", "b_rank_better"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("markdown missing %s", needle)
		}
	}
}
