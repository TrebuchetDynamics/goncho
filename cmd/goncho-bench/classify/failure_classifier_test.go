package classify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifyFailureCasesSelectsHardRanksAndCategories(t *testing.T) {
	report := Report{System: "goncho", Dataset: "fixture", Questions: []QuestionReport{
		{ID: "rank1", Query: "easy", RelevantIDs: []string{"answer_a"}, RetrievedIDs: []string{"answer_a"}, Rank: 1},
		{ID: "temporal", Query: "Which trip happened first?", RelevantIDs: []string{"answer_trip_abs_1"}, RetrievedIDs: []string{"other_trip", "answer_trip_abs_1"}, Rank: 2},
		{ID: "numeric", Query: "How many plants did I buy?", RelevantIDs: []string{"answer_plants_1"}, RetrievedIDs: []string{"wrong_plants", "other", "answer_plants_1"}, Rank: 3},
		{ID: "miss", Query: "What obscure synonym should match?", RelevantIDs: []string{"answer_synonym"}, RetrievedIDs: []string{"d1", "d2", "d3"}, Rank: 0},
	}}
	cases := classifyFailureCases(report)
	if len(cases) != 3 {
		t.Fatalf("cases len = %d, want 3 hard cases", len(cases))
	}
	want := map[string]string{
		"temporal": "temporal_ambiguity",
		"numeric":  "numeric_entity_exactness",
		"miss":     "lexical_miss",
	}
	for _, got := range cases {
		if got.Category != want[got.ID] {
			t.Fatalf("case %s category = %q, want %q; case=%+v", got.ID, got.Category, want[got.ID], got)
		}
		if got.Rank != 0 && got.Rank != 2 && got.Rank != 3 {
			t.Fatalf("case %s rank = %d, want hard rank only", got.ID, got.Rank)
		}
	}
}

func TestWriteFailureCategoryReportsEmitsJSONLAndMarkdown(t *testing.T) {
	dir := t.TempDir()
	jsonl := filepath.Join(dir, "categories.jsonl")
	md := filepath.Join(dir, "summary.md")
	report := Report{System: "goncho", Dataset: "fixture", MRR: 0.91, RecallAnyAt10: 0.98, Questions: []QuestionReport{
		{ID: "duplicate", Query: "What did dad gave me?", RelevantIDs: []string{"answer_gift"}, RetrievedIDs: []string{"gift", "answer_gift"}, Rank: 2},
		{ID: "miss", Query: "Can you recommend recent publications?", RelevantIDs: []string{"answer_pubs"}, RetrievedIDs: []string{"a", "b"}, Rank: 0},
	}}
	cases := classifyFailureCases(report)
	if err := writeFailureCategoryReports(jsonl, md, report, cases); err != nil {
		t.Fatalf("write reports: %v", err)
	}
	rawJSONL, err := os.ReadFile(jsonl)
	if err != nil {
		t.Fatalf("read jsonl: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(rawJSONL)), "\n")
	if len(lines) != 2 {
		t.Fatalf("jsonl lines = %d, want 2: %s", len(lines), rawJSONL)
	}
	var row failureCategoryRow
	if err := json.Unmarshal([]byte(lines[0]), &row); err != nil {
		t.Fatalf("decode first row: %v", err)
	}
	if row.ID != "duplicate" || row.Category != "duplicate_near_duplicate_content" {
		t.Fatalf("first row = %+v, want duplicate_near_duplicate_content", row)
	}
	rawMD, err := os.ReadFile(md)
	if err != nil {
		t.Fatalf("read md: %v", err)
	}
	text := string(rawMD)
	for _, needle := range []string{"# LongMemEval-S Failure Category Report", "duplicate_near_duplicate_content", "lexical_miss", "misses in top 10"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("markdown missing %q:\n%s", needle, text)
		}
	}
}
