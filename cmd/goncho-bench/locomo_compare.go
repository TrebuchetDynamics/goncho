package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const locomoNotFoundRank = 999999

type locomoComparisonRow struct {
	QuestionID         string   `json:"question_id"`
	Category           string   `json:"category"`
	Question           string   `json:"question"`
	GoldMemoryIDs      []string `json:"gold_memory_ids"`
	BM25GoldBestRank   int      `json:"bm25_gold_best_rank"`
	GonchoGoldBestRank int      `json:"goncho_gold_best_rank"`
	BM25Top10          []string `json:"bm25_top_10"`
	GonchoTop10        []string `json:"goncho_top_10"`
	Winner             string   `json:"winner"`
	RankDelta          int      `json:"rank_delta"`
	LikelyFailureMode  string   `json:"likely_failure_mode"`
}

type locomoComparisonSummary struct {
	BM25Wins      int
	GonchoWins    int
	Ties          int
	BothMiss      int
	ByCategory    map[string]map[string]int
	ByFailureMode map[string]int
	Rows          []locomoComparisonRow
}

func generateLocomoComparison(reportPath, jsonlOut, mdOut string) error {
	report, err := loadLocomoReport(reportPath)
	if err != nil {
		return err
	}
	rows, err := compareLocomoSystems(report, "bm25", "goncho")
	if err != nil {
		return err
	}
	if err := writeLocomoComparisonJSONL(jsonlOut, rows); err != nil {
		return err
	}
	return writeLocomoComparisonMarkdown(mdOut, report, summarizeLocomoComparison(rows), reportPath, jsonlOut)
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
			GoldMemoryIDs:    append([]string(nil), aq.GoldMemoryIDs...),
			BM25GoldBestRank: ar, GonchoGoldBestRank: br,
			BM25Top10: topN(aq.RetrievedIDs, 10), GonchoTop10: topN(bq.RetrievedIDs, 10),
			Winner: compareWinner(ar, br), RankDelta: br - ar,
		}
		row.LikelyFailureMode = classifyLocomoComparison(row)
		rows = append(rows, row)
	}
	return rows, nil
}

func normalizedRank(rank int) int {
	if rank <= 0 {
		return locomoNotFoundRank
	}
	return rank
}

func compareWinner(bm25Rank, gonchoRank int) string {
	if bm25Rank == locomoNotFoundRank && gonchoRank == locomoNotFoundRank {
		return "both_miss"
	}
	if bm25Rank < gonchoRank {
		return "bm25"
	}
	if gonchoRank < bm25Rank {
		return "goncho"
	}
	return "tie"
}

func topN(ids []string, n int) []string {
	if len(ids) < n {
		n = len(ids)
	}
	return append([]string(nil), ids[:n]...)
}

func classifyLocomoComparison(row locomoComparisonRow) string {
	q := strings.ToLower(row.Question)
	if len(row.GoldMemoryIDs) > 1 {
		return "gold_ambiguity"
	}
	if row.Winner == "bm25" {
		if row.BM25GoldBestRank != locomoNotFoundRank && row.GonchoGoldBestRank == locomoNotFoundRank {
			return "missing_candidate"
		}
		if row.BM25GoldBestRank != locomoNotFoundRank && row.GonchoGoldBestRank != locomoNotFoundRank {
			return "rerank_regression"
		}
	}
	if containsAny(q, []string{"who ", "said", "told", "mentioned", "according to"}) {
		return "speaker_attribution"
	}
	if containsAny(q, []string{"now", "current", "currently", "latest", "recent", "before", "after", "when", "how long"}) {
		return "temporal_evolution"
	}
	if containsAny(q, []string{"replace", "changed", "migrated", "used to", "formerly", "instead"}) {
		return "contradiction_handling"
	}
	if containsAny(q, []string{"which", "what", "where", "when", "how many", "how much"}) {
		return "entity_exactness"
	}
	if row.Winner == "both_miss" {
		return "unknown"
	}
	return "lexical_grounding"
}

func summarizeLocomoComparison(rows []locomoComparisonRow) locomoComparisonSummary {
	s := locomoComparisonSummary{ByCategory: map[string]map[string]int{}, ByFailureMode: map[string]int{}, Rows: rows}
	for _, row := range rows {
		switch row.Winner {
		case "bm25":
			s.BM25Wins++
		case "goncho":
			s.GonchoWins++
		case "tie":
			s.Ties++
		case "both_miss":
			s.BothMiss++
		}
		if _, ok := s.ByCategory[row.Category]; !ok {
			s.ByCategory[row.Category] = map[string]int{}
		}
		s.ByCategory[row.Category][row.Winner]++
		if row.Winner == "bm25" {
			s.ByFailureMode[row.LikelyFailureMode]++
		}
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
	bm25, goncho := findLocomoSystem(report, "bm25"), findLocomoSystem(report, "goncho")
	var b strings.Builder
	b.WriteString("# LOCOMO BM25 vs Goncho Failure Analysis — 2026-05-20\n\n")
	b.WriteString("Diagnosis only: no ranking changes, no LLM judge, no answer-generation scoring.\n\n")
	fmt.Fprintf(&b, "- Source report: `%s`\n- JSONL comparison: `%s`\n- Questions: `%d`\n\n", reportPath, jsonlPath, report.QuestionCount)
	b.WriteString("## Summary metrics\n\n| System | recall_any@5 | recall_any@10 | strict@5 | strict@10 | MRR |\n| --- | ---: | ---: | ---: | ---: | ---: |\n")
	for _, sys := range []locomoSystemReport{bm25, goncho} {
		fmt.Fprintf(&b, "| %s | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% |\n", sys.System, sys.RecallAnyAt5*100, sys.RecallAnyAt10*100, sys.StrictRecallAt5*100, sys.StrictRecallAt10*100, sys.MRR*100)
	}
	b.WriteString("\n## Winner counts\n\n| Winner | Count |\n| --- | ---: |\n")
	fmt.Fprintf(&b, "| BM25 wins | %d |\n| Goncho wins | %d |\n| Ties | %d |\n| Both miss | %d |\n", s.BM25Wins, s.GonchoWins, s.Ties, s.BothMiss)
	b.WriteString("\n## Category breakdown\n\n| Category | BM25 wins | Goncho wins | Ties | Both miss |\n| --- | ---: | ---: | ---: | ---: |\n")
	for _, cat := range sortedNestedKeys(s.ByCategory) {
		m := s.ByCategory[cat]
		fmt.Fprintf(&b, "| `%s` | %d | %d | %d | %d |\n", cat, m["bm25"], m["goncho"], m["tie"], m["both_miss"])
	}
	b.WriteString("\n## BM25-win failure modes\n\n| Failure mode | Count |\n| --- | ---: |\n")
	for _, mode := range sortedIntKeys(s.ByFailureMode) {
		fmt.Fprintf(&b, "| `%s` | %d |\n", mode, s.ByFailureMode[mode])
	}
	b.WriteString("\n## Worst BM25-over-Goncho cases\n\n")
	for _, row := range topComparisonRows(s.Rows, "bm25", 10) {
		fmt.Fprintf(&b, "- `%s` delta `%d` mode `%s`: %s\n", row.QuestionID, row.RankDelta, row.LikelyFailureMode, row.Question)
	}
	b.WriteString("\n## Top Goncho-over-BM25 cases\n\n")
	for _, row := range topComparisonRows(s.Rows, "goncho", 10) {
		fmt.Fprintf(&b, "- `%s` delta `%d` mode `%s`: %s\n", row.QuestionID, row.RankDelta, row.LikelyFailureMode, row.Question)
	}
	b.WriteString("\n## Interpretation\n\n")
	b.WriteString("BM25's lead is primarily a lexical/candidate-ranking signal. The comparison separates missing candidates from reranking regressions: `missing_candidate` means BM25 retrieved a gold memory in top 10 while Goncho did not; `rerank_regression` means both found gold but Goncho ranked it lower. Optimize only after reviewing these buckets.\n\n")
	b.WriteString("Recommended next slice: inspect BM25-win `missing_candidate` and `rerank_regression` rows side by side with memory content, then decide whether candidate generation, metadata/noise, or conservative reranking needs work.\n")
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
