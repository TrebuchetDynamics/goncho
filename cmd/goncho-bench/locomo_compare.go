package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	benchlocomo "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/locomo"
)

const locomoNotFoundRank = benchlocomo.NotFoundRank

type locomoComparisonRow struct {
	QuestionID     string   `json:"question_id"`
	Category       string   `json:"category"`
	Question       string   `json:"question"`
	GoldMemoryIDs  []string `json:"gold_memory_ids"`
	ASystem        string   `json:"a_system"`
	BSystem        string   `json:"b_system"`
	AGoldBestRank  int      `json:"a_gold_best_rank"`
	BGoldBestRank  int      `json:"b_gold_best_rank"`
	ATop10         []string `json:"a_top_10"`
	BTop10         []string `json:"b_top_10"`
	Winner         string   `json:"winner"`
	RankDelta      int      `json:"rank_delta"`
	DeltaBucket    string   `json:"delta_bucket"`
	DiagnosticHint string   `json:"diagnostic_hint,omitempty"`
}

type locomoComparisonSummary struct {
	ASystem       string
	BSystem       string
	AWins         int
	BWins         int
	Ties          int
	BothMiss      int
	ByCategory    map[string]map[string]int
	ByDeltaBucket map[string]int
	Rows          []locomoComparisonRow
}

func generateLocomoComparison(reportPath, jsonlOut, mdOut, aName, bName string) error {
	report, err := loadLocomoReport(reportPath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(aName) == "" {
		aName = "bm25"
	}
	if strings.TrimSpace(bName) == "" {
		bName = "goncho"
	}
	rows, err := compareLocomoSystems(report, aName, bName)
	if err != nil {
		return err
	}
	if err := writeLocomoComparisonJSONL(jsonlOut, rows); err != nil {
		return err
	}
	return writeLocomoComparisonMarkdown(mdOut, report, summarizeLocomoComparison(aName, bName, rows), reportPath, jsonlOut)
}

func loadLocomoReport(path string) (locomoReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return locomoReport{}, fmt.Errorf("goncho-bench: read LOCOMO report: %w", err)
	}
	var report locomoReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return locomoReport{}, fmt.Errorf("goncho-bench: decode LOCOMO report: %w", err)
	}
	return report, nil
}

func compareLocomoSystems(report locomoReport, aName, bName string) ([]locomoComparisonRow, error) {
	a, b := (*locomoSystemReport)(nil), (*locomoSystemReport)(nil)
	for i := range report.Systems {
		if report.Systems[i].System == aName {
			a = &report.Systems[i]
		}
		if report.Systems[i].System == bName {
			b = &report.Systems[i]
		}
	}
	if a == nil || b == nil {
		return nil, fmt.Errorf("goncho-bench: LOCOMO report missing %s or %s system", aName, bName)
	}
	bByID := map[string]locomoQuestionResult{}
	for _, q := range b.QuestionsDetail {
		bByID[q.QuestionID] = q
	}
	rows := make([]locomoComparisonRow, 0, len(a.QuestionsDetail))
	for _, aq := range a.QuestionsDetail {
		bq, ok := bByID[aq.QuestionID]
		if !ok {
			return nil, fmt.Errorf("goncho-bench: question %s missing from %s", aq.QuestionID, bName)
		}
		ar, br := normalizedRank(aq.Rank), normalizedRank(bq.Rank)
		row := locomoComparisonRow{
			QuestionID: aq.QuestionID, Category: aq.Category, Question: aq.Question,
			GoldMemoryIDs: append([]string(nil), aq.GoldMemoryIDs...),
			ASystem:       aName, BSystem: bName,
			AGoldBestRank: ar, BGoldBestRank: br,
			ATop10: topN(aq.RetrievedIDs, 10), BTop10: topN(bq.RetrievedIDs, 10),
			Winner: compareWinner(ar, br), RankDelta: br - ar,
		}
		row.DeltaBucket = classifyLocomoDeltaBucket(row)
		row.DiagnosticHint = classifyLocomoComparison(row)
		rows = append(rows, row)
	}
	return rows, nil
}

func normalizedRank(rank int) int {
	return benchlocomo.NormalizedRank(rank)
}

func compareWinner(aRank, bRank int) string {
	return benchlocomo.CompareWinner(aRank, bRank)
}

func topN(ids []string, n int) []string {
	if len(ids) < n {
		n = len(ids)
	}
	return append([]string(nil), ids[:n]...)
}

func classifyLocomoDeltaBucket(row locomoComparisonRow) string {
	return benchlocomo.ClassifyDeltaBucket(toBenchLocomoComparisonRow(row))
}

func classifyLocomoComparison(row locomoComparisonRow) string {
	return benchlocomo.ClassifyComparison(toBenchLocomoComparisonRow(row))
}

func toBenchLocomoComparisonRow(row locomoComparisonRow) benchlocomo.ComparisonRow {
	return benchlocomo.ComparisonRow{
		Question:      row.Question,
		GoldMemoryIDs: append([]string(nil), row.GoldMemoryIDs...),
		AGoldBestRank: row.AGoldBestRank,
		BGoldBestRank: row.BGoldBestRank,
		Winner:        row.Winner,
		DeltaBucket:   row.DeltaBucket,
	}
}

func summarizeLocomoComparison(aName, bName string, rows []locomoComparisonRow) locomoComparisonSummary {
	s := locomoComparisonSummary{ASystem: aName, BSystem: bName, ByCategory: map[string]map[string]int{}, ByDeltaBucket: map[string]int{}, Rows: rows}
	for _, row := range rows {
		switch row.Winner {
		case "a":
			s.AWins++
		case "b":
			s.BWins++
		case "tie":
			s.Ties++
		case "both_miss":
			s.BothMiss++
		}
		if _, ok := s.ByCategory[row.Category]; !ok {
			s.ByCategory[row.Category] = map[string]int{}
		}
		s.ByCategory[row.Category][row.DeltaBucket]++
		s.ByDeltaBucket[row.DeltaBucket]++
	}
	return s
}

func writeLocomoComparisonJSONL(path string, rows []locomoComparisonRow) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			_ = file.Close()
			return err
		}
	}
	return file.Close()
}

func writeLocomoComparisonMarkdown(path string, report locomoReport, s locomoComparisonSummary, reportPath, jsonlPath string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	a, bsys := findLocomoSystem(report, s.ASystem), findLocomoSystem(report, s.BSystem)
	var b strings.Builder
	b.WriteString("# LOCOMO Paired Delta Audit\n\n")
	b.WriteString("Diagnosis only: no ranking changes, no LLM judge, no answer-generation scoring. Winner labels `a` and `b` refer to the systems named below.\n\n")
	fmt.Fprintf(&b, "- Source report: `%s`\n- JSONL comparison: `%s`\n- A system: `%s`\n- B system: `%s`\n- Questions: `%d`\n\n", reportPath, jsonlPath, s.ASystem, s.BSystem, report.QuestionCount)
	b.WriteString("## Summary metrics\n\n| Side | System | recall_any@5 | recall_any@10 | strict@5 | strict@10 | MRR |\n| --- | --- | ---: | ---: | ---: | ---: | ---: |\n")
	for _, item := range []struct {
		side string
		sys  locomoSystemReport
	}{{"a", a}, {"b", bsys}} {
		fmt.Fprintf(&b, "| %s | `%s` | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% |\n", item.side, item.sys.System, item.sys.RecallAnyAt5*100, item.sys.RecallAnyAt10*100, item.sys.StrictRecallAt5*100, item.sys.StrictRecallAt10*100, item.sys.MRR*100)
	}
	b.WriteString("\n## Winner counts\n\n| Winner | Count |\n| --- | ---: |\n")
	fmt.Fprintf(&b, "| a (`%s`) wins | %d |\n| b (`%s`) wins | %d |\n| Ties | %d |\n| Both miss | %d |\n", s.ASystem, s.AWins, s.BSystem, s.BWins, s.Ties, s.BothMiss)
	b.WriteString("\n## Delta bucket counts\n\n| Delta bucket | Count |\n| --- | ---: |\n")
	for _, bucket := range sortedIntKeys(s.ByDeltaBucket) {
		fmt.Fprintf(&b, "| `%s` | %d |\n", bucket, s.ByDeltaBucket[bucket])
	}
	b.WriteString("\n## Category × delta bucket\n\n| Category | a_only_hit | b_only_hit | a_rank_better | b_rank_better | same_rank | both_miss |\n| --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, cat := range sortedNestedKeys(s.ByCategory) {
		m := s.ByCategory[cat]
		fmt.Fprintf(&b, "| `%s` | %d | %d | %d | %d | %d | %d |\n", cat, m["a_only_hit"], m["b_only_hit"], m["a_rank_better"], m["b_rank_better"], m["same_rank"], m["both_miss"])
	}
	b.WriteString("\n## Largest a-over-b cases\n\n")
	for _, row := range topComparisonRows(s.Rows, "a", 10) {
		fmt.Fprintf(&b, "- `%s` delta `%d` bucket `%s`: %s\n", row.QuestionID, row.RankDelta, row.DeltaBucket, row.Question)
	}
	b.WriteString("\n## Largest b-over-a cases\n\n")
	for _, row := range topComparisonRows(s.Rows, "b", 10) {
		fmt.Fprintf(&b, "- `%s` delta `%d` bucket `%s`: %s\n", row.QuestionID, row.RankDelta, row.DeltaBucket, row.Question)
	}
	b.WriteString("\n## Next action\n\nInspect the largest `a_only_hit` and `a_rank_better` buckets before tuning. `a_only_hit` means system B missed a gold memory that system A retrieved; `a_rank_better` means both retrieved gold but B ranked it lower.\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func findLocomoSystem(report locomoReport, name string) locomoSystemReport {
	for _, s := range report.Systems {
		if s.System == name {
			return s
		}
	}
	return locomoSystemReport{}
}
func sortedNestedKeys(m map[string]map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func sortedIntKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func topComparisonRows(rows []locomoComparisonRow, winner string, n int) []locomoComparisonRow {
	out := []locomoComparisonRow{}
	for _, r := range rows {
		if r.Winner == winner {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if abs(out[i].RankDelta) == abs(out[j].RankDelta) {
			return out[i].QuestionID < out[j].QuestionID
		}
		return abs(out[i].RankDelta) > abs(out[j].RankDelta)
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
