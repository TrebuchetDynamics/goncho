package goncho

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	RecallBenchmarkCorpusVersion = "goncho-recall-benchmark-v1"

	RecallBenchmarkWarningMissingTrace  = "benchmark_missing_trace"
	RecallBenchmarkWarningNoRelevantIDs = "benchmark_no_relevant_ids"
)

// RecallBenchmarkCase is one hermetic retrieval-evaluation case. It consumes
// an already-produced RecallTrace; it never runs retrieval or opens storage.
type RecallBenchmarkCase struct {
	ID                    string
	Ability               string
	Trace                 RecallTrace
	RelevantIDs           []string
	ContextContains       []string
	RequiredEvidenceKinds []string
	Latency               time.Duration
}

type RecallBenchmarkReport struct {
	Service             string                         `json:"service"`
	CorpusVersion       string                         `json:"corpus_version"`
	CaseCount           int                            `json:"case_count"`
	RecallAt5           float64                        `json:"recall_at_5"`
	RecallAt10          float64                        `json:"recall_at_10"`
	ContextHitRate      float64                        `json:"context_hit_rate"`
	TokenBudgetPassRate float64                        `json:"token_budget_pass_rate"`
	Latency             RecallBenchmarkLatency         `json:"latency"`
	WarningCount        int                            `json:"warning_count"`
	Warnings            []RecallWarning                `json:"warnings"`
	Abilities           []RecallBenchmarkAbilityReport `json:"abilities,omitempty"`
	Cases               []RecallBenchmarkCaseReport    `json:"cases"`
}

type RecallBenchmarkLatency struct {
	MinMS int `json:"min_ms"`
	P50MS int `json:"p50_ms"`
	P95MS int `json:"p95_ms"`
	MaxMS int `json:"max_ms"`
}

type RecallBenchmarkAbilityReport struct {
	Ability             string  `json:"ability"`
	CaseCount           int     `json:"case_count"`
	RecallAt5           float64 `json:"recall_at_5"`
	RecallAt10          float64 `json:"recall_at_10"`
	ContextHitRate      float64 `json:"context_hit_rate"`
	TokenBudgetPassRate float64 `json:"token_budget_pass_rate"`
	ProvenanceHitRate   float64 `json:"provenance_hit_rate"`
}

type RecallBenchmarkCaseReport struct {
	ID                    string   `json:"id"`
	Ability               string   `json:"ability,omitempty"`
	TraceID               string   `json:"trace_id"`
	PipelineVersion       string   `json:"pipeline_version"`
	ScoringConfigVersion  string   `json:"scoring_config_version"`
	RelevantIDs           []string `json:"relevant_ids"`
	RequiredEvidenceKinds []string `json:"required_evidence_kinds,omitempty"`
	CandidateMemoryIDs    []string `json:"candidate_memory_ids"`
	SelectedMemoryIDs     []string `json:"selected_memory_ids"`
	RecallAt5             float64  `json:"recall_at_5"`
	RecallAt10            float64  `json:"recall_at_10"`
	ContextSatisfied      bool     `json:"context_satisfied"`
	ProvenanceSatisfied   bool     `json:"provenance_satisfied,omitempty"`
	TokenBudget           int      `json:"token_budget"`
	SelectedTokens        int      `json:"selected_tokens"`
	TokenBudgetWithin     bool     `json:"token_budget_within"`
	LatencyMS             int      `json:"latency_ms"`
	WarningCodes          []string `json:"warning_codes"`
}

func EvaluateRecallBenchmark(cases []RecallBenchmarkCase) RecallBenchmarkReport {
	report := RecallBenchmarkReport{
		Service:       "goncho",
		CorpusVersion: RecallBenchmarkCorpusVersion,
		CaseCount:     len(cases),
		Cases:         []RecallBenchmarkCaseReport{},
		Warnings:      []RecallWarning{},
	}
	if len(cases) == 0 {
		return report
	}

	latencies := make([]int, 0, len(cases))
	var recallAt5Sum, recallAt10Sum, contextHits, tokenBudgetPasses float64
	for i, c := range cases {
		caseReport, warnings := evaluateRecallBenchmarkCase(i, c)
		report.Cases = append(report.Cases, caseReport)
		report.Warnings = append(report.Warnings, warnings...)
		recallAt5Sum += caseReport.RecallAt5
		recallAt10Sum += caseReport.RecallAt10
		if caseReport.ContextSatisfied {
			contextHits++
		}
		if caseReport.TokenBudgetWithin {
			tokenBudgetPasses++
		}
		latencies = append(latencies, caseReport.LatencyMS)
	}
	report.RecallAt5 = roundRecallFloat(recallAt5Sum / float64(len(cases)))
	report.RecallAt10 = roundRecallFloat(recallAt10Sum / float64(len(cases)))
	report.ContextHitRate = roundRecallFloat(contextHits / float64(len(cases)))
	report.TokenBudgetPassRate = roundRecallFloat(tokenBudgetPasses / float64(len(cases)))
	report.Latency = summarizeRecallBenchmarkLatency(latencies)
	report.WarningCount = len(report.Warnings)
	report.Abilities = summarizeRecallBenchmarkAbilities(report.Cases)
	return report
}

func evaluateRecallBenchmarkCase(index int, c RecallBenchmarkCase) (RecallBenchmarkCaseReport, []RecallWarning) {
	id := strings.TrimSpace(c.ID)
	if id == "" {
		id = fmt.Sprintf("case-%03d", index+1)
	}
	candidateIDs := recallBenchmarkCandidateIDs(c.Trace.Candidates)
	selectedIDs := recallBenchmarkSelectedIDs(c.Trace.Selected)
	budget := recallBenchmarkTokenBudget(c.Trace)
	selectedTokens := recallBenchmarkSelectedTokens(c.Trace.Selected)
	requiredEvidenceKinds := normalizeRecallBenchmarkEvidenceKinds(c.RequiredEvidenceKinds)
	caseReport := RecallBenchmarkCaseReport{
		ID:                    id,
		Ability:               normalizeRecallBenchmarkAbility(c.Ability),
		TraceID:               c.Trace.TraceID,
		PipelineVersion:       c.Trace.PipelineVersion,
		ScoringConfigVersion:  c.Trace.ScoringConfig.Version,
		RelevantIDs:           append([]string(nil), c.RelevantIDs...),
		RequiredEvidenceKinds: requiredEvidenceKinds,
		CandidateMemoryIDs:    candidateIDs,
		SelectedMemoryIDs:     selectedIDs,
		RecallAt5:             recallAtK(candidateIDs, c.RelevantIDs, 5),
		RecallAt10:            recallAtK(candidateIDs, c.RelevantIDs, 10),
		ContextSatisfied:      recallBenchmarkContextSatisfied(c.Trace, c.RelevantIDs, c.ContextContains),
		ProvenanceSatisfied:   len(requiredEvidenceKinds) > 0 && recallBenchmarkProvenanceSatisfied(c.Trace, c.RelevantIDs, requiredEvidenceKinds),
		TokenBudget:           budget,
		SelectedTokens:        selectedTokens,
		TokenBudgetWithin:     budget <= 0 || selectedTokens <= budget,
		LatencyMS:             int(c.Latency / time.Millisecond),
		WarningCodes:          recallBenchmarkWarningCodes(c.Trace.Warnings),
	}
	if caseReport.RelevantIDs == nil {
		caseReport.RelevantIDs = []string{}
	}
	var warnings []RecallWarning
	if strings.TrimSpace(c.Trace.TraceID) == "" {
		warnings = append(warnings, RecallWarning{
			Code:     RecallBenchmarkWarningMissingTrace,
			Stage:    RecallStageGenerate,
			Severity: RecallWarningError,
			Message:  "benchmark case is missing trace_id",
			Evidence: map[string]string{"case_id": id},
		})
	}
	if len(c.RelevantIDs) == 0 {
		warnings = append(warnings, RecallWarning{
			Code:     RecallBenchmarkWarningNoRelevantIDs,
			Stage:    RecallStageScore,
			Severity: RecallWarningDegraded,
			Message:  "benchmark case has no relevant memory ids",
			Evidence: map[string]string{"case_id": id, "trace_id": c.Trace.TraceID},
		})
	}
	return caseReport, warnings
}

type recallBenchmarkAbilityAccumulator struct {
	caseCount           int
	recallAt5           float64
	recallAt10          float64
	contextHits         float64
	tokenBudgetPasses   float64
	provenanceSatisfied float64
}

func summarizeRecallBenchmarkAbilities(cases []RecallBenchmarkCaseReport) []RecallBenchmarkAbilityReport {
	stats := map[string]*recallBenchmarkAbilityAccumulator{}
	for _, c := range cases {
		ability := strings.TrimSpace(c.Ability)
		if ability == "" {
			continue
		}
		acc := stats[ability]
		if acc == nil {
			acc = &recallBenchmarkAbilityAccumulator{}
			stats[ability] = acc
		}
		acc.caseCount++
		acc.recallAt5 += c.RecallAt5
		acc.recallAt10 += c.RecallAt10
		if c.ContextSatisfied {
			acc.contextHits++
		}
		if c.TokenBudgetWithin {
			acc.tokenBudgetPasses++
		}
		if len(c.RequiredEvidenceKinds) == 0 || c.ProvenanceSatisfied {
			acc.provenanceSatisfied++
		}
	}
	if len(stats) == 0 {
		return nil
	}
	abilities := make([]string, 0, len(stats))
	for ability := range stats {
		abilities = append(abilities, ability)
	}
	sort.Strings(abilities)
	out := make([]RecallBenchmarkAbilityReport, 0, len(abilities))
	for _, ability := range abilities {
		acc := stats[ability]
		denom := float64(acc.caseCount)
		out = append(out, RecallBenchmarkAbilityReport{
			Ability:             ability,
			CaseCount:           acc.caseCount,
			RecallAt5:           roundRecallFloat(acc.recallAt5 / denom),
			RecallAt10:          roundRecallFloat(acc.recallAt10 / denom),
			ContextHitRate:      roundRecallFloat(acc.contextHits / denom),
			TokenBudgetPassRate: roundRecallFloat(acc.tokenBudgetPasses / denom),
			ProvenanceHitRate:   roundRecallFloat(acc.provenanceSatisfied / denom),
		})
	}
	return out
}

func normalizeRecallBenchmarkAbility(ability string) string {
	return strings.ToUpper(strings.TrimSpace(ability))
}

func normalizeRecallBenchmarkEvidenceKinds(kinds []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, kind := range kinds {
		kind = strings.ToLower(strings.TrimSpace(kind))
		if kind == "" {
			continue
		}
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		out = append(out, kind)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

func recallBenchmarkProvenanceSatisfied(trace RecallTrace, relevantIDs, requiredKinds []string) bool {
	required := map[string]struct{}{}
	for _, kind := range requiredKinds {
		kind = strings.ToLower(strings.TrimSpace(kind))
		if kind != "" {
			required[kind] = struct{}{}
		}
	}
	if len(required) == 0 {
		return false
	}
	relevant := map[string]struct{}{}
	for _, id := range relevantIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			relevant[id] = struct{}{}
		}
	}
	for _, item := range trace.Selected {
		if len(relevant) > 0 {
			if _, ok := relevant[item.Candidate.MemoryID]; !ok {
				continue
			}
		}
		seen := map[string]struct{}{}
		for _, evidence := range item.Candidate.Provenance {
			kind := strings.ToLower(strings.TrimSpace(evidence.Kind))
			if kind != "" {
				seen[kind] = struct{}{}
			}
		}
		matchedAll := true
		for kind := range required {
			if _, ok := seen[kind]; !ok {
				matchedAll = false
				break
			}
		}
		if matchedAll {
			return true
		}
	}
	return false
}

func recallBenchmarkCandidateIDs(items []ScoredRecallCandidate) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Candidate.MemoryID)
	}
	return out
}

func recallBenchmarkSelectedIDs(items []ScoredRecallCandidate) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Candidate.MemoryID)
	}
	return out
}

func recallAtK(candidateIDs, relevantIDs []string, k int) float64 {
	if len(relevantIDs) == 0 || k <= 0 {
		return 0
	}
	relevant := make(map[string]struct{}, len(relevantIDs))
	for _, id := range relevantIDs {
		if strings.TrimSpace(id) != "" {
			relevant[id] = struct{}{}
		}
	}
	if len(relevant) == 0 {
		return 0
	}
	limit := k
	if len(candidateIDs) < limit {
		limit = len(candidateIDs)
	}
	found := map[string]struct{}{}
	for _, id := range candidateIDs[:limit] {
		if _, ok := relevant[id]; ok {
			found[id] = struct{}{}
		}
	}
	return roundRecallFloat(float64(len(found)) / float64(len(relevant)))
}

func recallBenchmarkContextSatisfied(trace RecallTrace, relevantIDs []string, contains []string) bool {
	context := (&RecallProjector{}).ProjectContext(trace).Representation
	if len(contains) > 0 {
		for _, needle := range contains {
			if !strings.Contains(context, needle) {
				return false
			}
		}
		return true
	}
	selected := recallBenchmarkSelectedIDs(trace.Selected)
	for _, id := range relevantIDs {
		for _, selectedID := range selected {
			if id == selectedID {
				return true
			}
		}
	}
	return false
}

func recallBenchmarkTokenBudget(trace RecallTrace) int {
	if trace.Query.MaxTokens > 0 {
		return trace.Query.MaxTokens
	}
	return trace.ScoringConfig.TokenBudget
}

func recallBenchmarkSelectedTokens(items []ScoredRecallCandidate) int {
	total := 0
	for _, item := range items {
		total += estimateRecallTokens(item.Candidate.Content)
	}
	return total
}

func recallBenchmarkWarningCodes(warnings []RecallWarning) []string {
	out := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		if warning.Code != "" {
			out = append(out, warning.Code)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func summarizeRecallBenchmarkLatency(latencies []int) RecallBenchmarkLatency {
	if len(latencies) == 0 {
		return RecallBenchmarkLatency{}
	}
	sorted := append([]int(nil), latencies...)
	sort.Ints(sorted)
	return RecallBenchmarkLatency{
		MinMS: sorted[0],
		P50MS: nearestRank(sorted, 0.50),
		P95MS: nearestRank(sorted, 0.95),
		MaxMS: sorted[len(sorted)-1],
	}
}

func nearestRank(sorted []int, percentile float64) int {
	if len(sorted) == 0 {
		return 0
	}
	if percentile <= 0 {
		return sorted[0]
	}
	if percentile >= 1 {
		return sorted[len(sorted)-1]
	}
	rank := int(math.Ceil(percentile * float64(len(sorted))))
	if rank <= 0 {
		rank = 1
	}
	if rank > len(sorted) {
		rank = len(sorted)
	}
	return sorted[rank-1]
}
