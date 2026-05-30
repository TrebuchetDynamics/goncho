package classify

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	categoryTemporalAmbiguity        = "temporal_ambiguity"
	categoryNumericEntityExactness   = "numeric_entity_exactness"
	categoryDuplicateNearDuplicate   = "duplicate_near_duplicate_content"
	categoryDirectAnswerMismatch     = "direct_answer_mismatch"
	categoryLexicalMiss              = "lexical_miss"
	categoryStaleContradictoryMemory = "stale_contradictory_memory"
	categoryBenchmarkGoldAmbiguity   = "benchmark_gold_ambiguity"
	categoryTrueRetrievalFailure     = "true_retrieval_failure"
)

var benchmarkIDNumericSuffixPattern = regexp.MustCompile(`_[0-9]+$`)

type Report struct {
	System        string           `json:"system"`
	Dataset       string           `json:"dataset"`
	RecallAnyAt10 float64          `json:"recall_any_at_10"`
	MRR           float64          `json:"mrr"`
	Questions     []QuestionReport `json:"questions"`
}

type QuestionReport struct {
	ID           string   `json:"id"`
	Query        string   `json:"query"`
	RelevantIDs  []string `json:"relevant_ids"`
	RetrievedIDs []string `json:"retrieved_ids"`
	Rank         int      `json:"rank"`
}

type failureCategoryRow struct {
	ID           string   `json:"id"`
	Query        string   `json:"query"`
	Rank         int      `json:"rank"`
	Bucket       string   `json:"bucket"`
	Category     string   `json:"category"`
	Reason       string   `json:"reason"`
	RelevantIDs  []string `json:"relevant_ids"`
	RetrievedIDs []string `json:"retrieved_ids"`
}

func GenerateFailureCategoryReports(reportPath, failurePath, jsonlOut, mdOut string) error {
	report, err := loadBenchmarkReport(reportPath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(failurePath) != "" {
		if _, err := loadFailureAuditIDs(failurePath); err != nil {
			return err
		}
	}
	cases := classifyFailureCases(report)
	return writeFailureCategoryReports(jsonlOut, mdOut, report, cases)
}

func loadBenchmarkReport(path string) (Report, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Report{}, fmt.Errorf("goncho-bench: read benchmark report: %w", err)
	}
	var report Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return Report{}, fmt.Errorf("goncho-bench: decode benchmark report: %w", err)
	}
	return report, nil
}

func loadFailureAuditIDs(path string) (map[string]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("goncho-bench: open failure audit: %w", err)
	}
	defer file.Close()
	ids := map[string]struct{}{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 32*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row QuestionReport
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("goncho-bench: decode failure audit row: %w", err)
		}
		ids[row.ID] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("goncho-bench: scan failure audit: %w", err)
	}
	return ids, nil
}

func classifyFailureCases(report Report) []failureCategoryRow {
	rows := []failureCategoryRow{}
	for _, q := range report.Questions {
		if !isHardFailureCase(q) {
			continue
		}
		category, reason := classifyFailureCase(q)
		rows = append(rows, failureCategoryRow{
			ID:           q.ID,
			Query:        q.Query,
			Rank:         q.Rank,
			Bucket:       rankBucket(q.Rank),
			Category:     category,
			Reason:       reason,
			RelevantIDs:  append([]string(nil), q.RelevantIDs...),
			RetrievedIDs: append([]string(nil), q.RetrievedIDs...),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Rank == 0 && rows[j].Rank != 0 {
			return false
		}
		if rows[i].Rank != 0 && rows[j].Rank == 0 {
			return true
		}
		if rows[i].Rank != rows[j].Rank {
			return rows[i].Rank < rows[j].Rank
		}
		return rows[i].ID < rows[j].ID
	})
	return rows
}

func isHardFailureCase(q QuestionReport) bool {
	return q.Rank == 0 || q.Rank == 2 || q.Rank == 3
}

func classifyFailureCase(q QuestionReport) (string, string) {
	query := strings.ToLower(q.Query)
	if hasSameBaseIDBeforeGold(q) {
		return categoryDuplicateNearDuplicate, "a retrieved ID before the first strict gold hit shares a normalized base ID with a relevant ID"
	}
	if hasAnswerVariantMismatchBeforeGold(q) {
		return categoryDirectAnswerMismatch, "a retrieved answer/non-answer or abstention variant appears before the strict gold ID"
	}
	if q.Rank == 0 && len(q.RetrievedIDs) > 0 {
		return categoryLexicalMiss, "no relevant ID appears in the retrieved top-k despite non-empty retrieval results"
	}
	if q.Rank == 0 {
		return categoryTrueRetrievalFailure, "no relevant ID was retrieved"
	}
	if looksTemporal(query) {
		return categoryTemporalAmbiguity, "query asks for time, order, recency, duration, or before/after comparison"
	}
	if looksBenchmarkAmbiguous(q) {
		return categoryBenchmarkGoldAmbiguity, "multiple gold IDs or abstention variants suggest ambiguous strict attribution"
	}
	if looksNumericEntity(query) {
		return categoryNumericEntityExactness, "query asks for exact count, amount, name, entity, or object identification"
	}
	if looksStaleContradictory(query) {
		return categoryStaleContradictoryMemory, "query asks for current, previous, changed, or replacement state"
	}
	return categoryDirectAnswerMismatch, "relevant evidence is retrieved but not at the top rank"
}

func hasSameBaseIDBeforeGold(q QuestionReport) bool {
	gold := map[string]struct{}{}
	for _, id := range q.RelevantIDs {
		gold[normalizeBenchmarkIDBase(id)] = struct{}{}
	}
	limit := len(q.RetrievedIDs)
	if q.Rank > 0 && q.Rank-1 < limit {
		limit = q.Rank - 1
	}
	for _, id := range q.RetrievedIDs[:limit] {
		if _, ok := gold[normalizeBenchmarkIDBase(id)]; ok {
			return true
		}
	}
	return false
}

func hasAnswerVariantMismatchBeforeGold(q QuestionReport) bool {
	limit := len(q.RetrievedIDs)
	if q.Rank > 0 && q.Rank-1 < limit {
		limit = q.Rank - 1
	}
	goldAnswerish := false
	goldAbs := false
	for _, id := range q.RelevantIDs {
		goldAnswerish = goldAnswerish || strings.HasPrefix(id, "answer_")
		goldAbs = goldAbs || strings.Contains(id, "_abs")
	}
	goldBases := map[string]struct{}{}
	for _, id := range q.RelevantIDs {
		goldBases[normalizeBenchmarkIDBase(id)] = struct{}{}
	}
	for _, id := range q.RetrievedIDs[:limit] {
		if _, sameBase := goldBases[normalizeBenchmarkIDBase(id)]; !sameBase {
			continue
		}
		if strings.HasPrefix(id, "answer_") != goldAnswerish || strings.Contains(id, "_abs") != goldAbs {
			return true
		}
	}
	return false
}

func looksBenchmarkAmbiguous(q QuestionReport) bool {
	if len(q.RelevantIDs) > 1 {
		return true
	}
	for _, id := range append(append([]string{}, q.RelevantIDs...), q.RetrievedIDs...) {
		if strings.Contains(id, "_abs_") {
			return true
		}
	}
	return false
}

func normalizeBenchmarkIDBase(id string) string {
	id = strings.TrimPrefix(id, "answer_")
	id = strings.ReplaceAll(id, "_abs", "")
	id = benchmarkIDNumericSuffixPattern.ReplaceAllString(id, "")
	return id
}

func looksTemporal(query string) bool {
	needles := []string{"when ", "before", "after", "first", "last", "recent", "currently", "current", "previous", "how long", "how many days", "how many years", "order", "date", "weekend", "today", "yesterday", "month", "year"}
	return containsAny(query, needles)
}

func looksNumericEntity(query string) bool {
	needles := []string{"how many", "how much", "what is the name", "what name", "who ", "which ", "what type", "what kind", "what did", "what was", "where ", "amount", "number"}
	return containsAny(query, needles)
}

func looksStaleContradictory(query string) bool {
	needles := []string{"current", "currently", "now", "previous", "used to", "changed", "replaced", "instead", "latest", "new"}
	return containsAny(query, needles)
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func rankBucket(rank int) string {
	switch rank {
	case 0:
		return "misses in top 10"
	case 2:
		return "rank-2 cases"
	case 3:
		return "rank-3 cases"
	default:
		return fmt.Sprintf("rank-%d cases", rank)
	}
}

func writeFailureCategoryReports(jsonlOut, mdOut string, report Report, cases []failureCategoryRow) error {
	if strings.TrimSpace(jsonlOut) != "" {
		if err := os.MkdirAll(filepath.Dir(jsonlOut), 0o755); err != nil {
			return fmt.Errorf("goncho-bench: create category jsonl dir: %w", err)
		}
		file, err := os.Create(jsonlOut)
		if err != nil {
			return fmt.Errorf("goncho-bench: create category jsonl: %w", err)
		}
		enc := json.NewEncoder(file)
		for _, row := range cases {
			if err := enc.Encode(row); err != nil {
				_ = file.Close()
				return fmt.Errorf("goncho-bench: encode category row: %w", err)
			}
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("goncho-bench: close category jsonl: %w", err)
		}
	}
	if strings.TrimSpace(mdOut) != "" {
		if err := os.MkdirAll(filepath.Dir(mdOut), 0o755); err != nil {
			return fmt.Errorf("goncho-bench: create category markdown dir: %w", err)
		}
		if err := os.WriteFile(mdOut, []byte(renderFailureCategoryMarkdown(report, cases)), 0o644); err != nil {
			return fmt.Errorf("goncho-bench: write category markdown: %w", err)
		}
	}
	return nil
}

func renderFailureCategoryMarkdown(report Report, cases []failureCategoryRow) string {
	byCategory := map[string]int{}
	byBucket := map[string]int{}
	for _, row := range cases {
		byCategory[row.Category]++
		byBucket[row.Bucket]++
	}
	var b strings.Builder
	b.WriteString("# LongMemEval-S Failure Category Report\n\n")
	b.WriteString("This report classifies remaining hard cases before any ranking optimization. It is diagnostic only; it does not change scoring or retrieval.\n\n")
	fmt.Fprintf(&b, "- system: `%s`\n", report.System)
	fmt.Fprintf(&b, "- dataset: `%s`\n", report.Dataset)
	fmt.Fprintf(&b, "- MRR: `%.4f`\n", report.MRR)
	fmt.Fprintf(&b, "- recall_any@10: `%.4f`\n", report.RecallAnyAt10)
	fmt.Fprintf(&b, "- hard cases classified: `%d`\n\n", len(cases))
	b.WriteString("## Counts by bucket\n\n| Bucket | Count |\n| --- | ---: |\n")
	for _, key := range sortedKeys(byBucket) {
		fmt.Fprintf(&b, "| %s | %d |\n", key, byBucket[key])
	}
	b.WriteString("\n## Counts by category\n\n| Category | Count |\n| --- | ---: |\n")
	for _, key := range sortedKeys(byCategory) {
		fmt.Fprintf(&b, "| `%s` | %d |\n", key, byCategory[key])
	}
	b.WriteString("\n## Examples\n\n")
	seen := map[string]int{}
	for _, row := range cases {
		if seen[row.Category] >= 3 {
			continue
		}
		seen[row.Category]++
		fmt.Fprintf(&b, "- `%s` rank `%d` category `%s`: %s\n", row.ID, row.Rank, row.Category, row.Query)
		fmt.Fprintf(&b, "  - reason: %s\n", row.Reason)
		fmt.Fprintf(&b, "  - relevant: `%s`\n", strings.Join(row.RelevantIDs, "`, `"))
		limit := len(row.RetrievedIDs)
		if limit > 5 {
			limit = 5
		}
		fmt.Fprintf(&b, "  - retrieved top %d: `%s`\n", limit, strings.Join(row.RetrievedIDs[:limit], "`, `"))
	}
	return b.String()
}

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
