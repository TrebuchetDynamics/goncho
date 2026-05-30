package goncho

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

const (
	RecallBenchmarkCorpusVersion          = "goncho-recall-benchmark-v1"
	RecallBenchmarkServicePipelineVersion = "goncho-recall-benchmark-service-v1"

	RecallBenchmarkWarningMissingTrace  = "benchmark_missing_trace"
	RecallBenchmarkWarningNoRelevantIDs = "benchmark_no_relevant_ids"
)

// RecallBenchmarkCase is one hermetic retrieval-evaluation case. It consumes
// an already-produced RecallTrace; it never runs retrieval or opens storage.
type RecallBenchmarkCase struct {
	ID                    string
	Ability               string
	Scale                 string
	ConversationID        string
	IdealAnswer           string
	Rubric                []string
	Trace                 RecallTrace
	RelevantIDs           []string
	ContextContains       []string
	RequiredEvidenceKinds []string
	ExpectedNoAnswer      bool
	Latency               time.Duration
}

// RecallBenchmarkServiceMemory is one public Service.Conclude write used by a
// service-backed benchmark case. Ref is local to the case and maps benchmark
// relevant IDs to the concrete conclusion IDs generated during ingestion.
type RecallBenchmarkServiceMemory struct {
	Ref        string
	Conclusion string
	Peer       string
	SessionKey string
	Scope      string
}

// RecallBenchmarkServiceCase runs a tiny BEAM-style fixture through public
// conclusion writes and the recall pipeline before EvaluateRecallBenchmark
// aggregates the resulting RecallTrace.
type RecallBenchmarkServiceCase struct {
	ID                    string
	Ability               string
	Scale                 string
	ConversationID        string
	Peer                  string
	SessionKey            string
	Query                 string
	IdealAnswer           string
	Rubric                []string
	ScopeID               string
	Memories              []RecallBenchmarkServiceMemory
	RelevantRefs          []string
	ContextContains       []string
	RequiredEvidenceKinds []string
	ExpectedNoAnswer      bool
	Limit                 int
	MaxTokens             int
	ScoringConfig         RecallScoringConfig
	PipelineVersion       string
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
	ID                     string   `json:"id"`
	Ability                string   `json:"ability,omitempty"`
	Scale                  string   `json:"scale,omitempty"`
	ConversationID         string   `json:"conversation_id,omitempty"`
	Question               string   `json:"question,omitempty"`
	IdealAnswer            string   `json:"ideal_answer,omitempty"`
	Rubric                 []string `json:"rubric,omitempty"`
	TraceID                string   `json:"trace_id"`
	PipelineVersion        string   `json:"pipeline_version"`
	ScoringConfigVersion   string   `json:"scoring_config_version"`
	RelevantIDs            []string `json:"relevant_ids"`
	RequiredEvidenceKinds  []string `json:"required_evidence_kinds,omitempty"`
	ExpectedNoAnswer       bool     `json:"expected_no_answer,omitempty"`
	RubricContextScore     float64  `json:"rubric_context_score,omitempty"`
	RubricContextMatches   []string `json:"rubric_context_matches,omitempty"`
	CandidateMemoryIDs     []string `json:"candidate_memory_ids"`
	SelectedMemoryIDs      []string `json:"selected_memory_ids"`
	SelectedContext        string   `json:"selected_context,omitempty"`
	CandidateEvidenceKinds []string `json:"candidate_evidence_kinds,omitempty"`
	SelectedEvidenceKinds  []string `json:"selected_evidence_kinds,omitempty"`
	TopEvidenceKinds       []string `json:"top_evidence_kinds,omitempty"`
	RecallAt5              float64  `json:"recall_at_5"`
	RecallAt10             float64  `json:"recall_at_10"`
	ContextSatisfied       bool     `json:"context_satisfied"`
	ProvenanceSatisfied    bool     `json:"provenance_satisfied,omitempty"`
	TokenBudget            int      `json:"token_budget"`
	SelectedTokens         int      `json:"selected_tokens"`
	TokenBudgetWithin      bool     `json:"token_budget_within"`
	LatencyMS              int      `json:"latency_ms"`
	WarningCodes           []string `json:"warning_codes"`
}

func EvaluateServiceRecallBenchmark(ctx context.Context, svc *Service, cases []RecallBenchmarkServiceCase) (RecallBenchmarkReport, error) {
	if svc == nil {
		return RecallBenchmarkReport{}, fmt.Errorf("goncho recall benchmark: service is required")
	}
	benchmarkCases := make([]RecallBenchmarkCase, 0, len(cases))
	for i, c := range cases {
		benchmarkCase, err := runServiceRecallBenchmarkCase(ctx, svc, i, c)
		if err != nil {
			return RecallBenchmarkReport{}, err
		}
		benchmarkCases = append(benchmarkCases, benchmarkCase)
	}
	return EvaluateRecallBenchmark(benchmarkCases), nil
}

func runServiceRecallBenchmarkCase(ctx context.Context, svc *Service, index int, c RecallBenchmarkServiceCase) (RecallBenchmarkCase, error) {
	if err := ctx.Err(); err != nil {
		return RecallBenchmarkCase{}, err
	}
	id := strings.TrimSpace(c.ID)
	if id == "" {
		id = fmt.Sprintf("service-case-%03d", index+1)
	}
	peer := strings.TrimSpace(c.Peer)
	if peer == "" {
		peer = "benchmark"
	}
	sessionKey := strings.TrimSpace(c.SessionKey)
	if sessionKey == "" {
		sessionKey = "sess-" + id
	}
	query := strings.TrimSpace(c.Query)
	if query == "" {
		return RecallBenchmarkCase{}, fmt.Errorf("goncho recall benchmark: case %q query is required", id)
	}
	started := time.Now()
	refToID := map[string]string{}
	for i, memory := range c.Memories {
		ref := strings.TrimSpace(memory.Ref)
		if ref == "" {
			ref = fmt.Sprintf("memory-%03d", i+1)
		}
		if _, exists := refToID[ref]; exists {
			return RecallBenchmarkCase{}, fmt.Errorf("goncho recall benchmark: case %q duplicate memory ref %q", id, ref)
		}
		conclusion := strings.TrimSpace(memory.Conclusion)
		if conclusion == "" {
			return RecallBenchmarkCase{}, fmt.Errorf("goncho recall benchmark: case %q memory %q conclusion is required", id, ref)
		}
		memoryPeer := strings.TrimSpace(memory.Peer)
		if memoryPeer == "" {
			memoryPeer = peer
		}
		memorySessionKey := strings.TrimSpace(memory.SessionKey)
		if memorySessionKey == "" {
			memorySessionKey = sessionKey
		}
		written, err := svc.Conclude(ctx, ConcludeParams{
			Peer:       memoryPeer,
			Conclusion: conclusion,
			SessionKey: memorySessionKey,
			Scope:      memory.Scope,
		})
		if err != nil {
			return RecallBenchmarkCase{}, fmt.Errorf("goncho recall benchmark: case %q write memory %q: %w", id, ref, err)
		}
		refToID[ref] = strconv.FormatInt(written.ID, 10)
	}
	relevantIDs := make([]string, 0, len(c.RelevantRefs))
	for _, ref := range c.RelevantRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		memoryID, ok := refToID[ref]
		if !ok {
			return RecallBenchmarkCase{}, fmt.Errorf("goncho recall benchmark: case %q relevant ref %q was not written", id, ref)
		}
		relevantIDs = append(relevantIDs, memoryID)
	}
	limit := c.Limit
	if limit <= 0 {
		limit = 5
	}
	pipelineVersion := strings.TrimSpace(c.PipelineVersion)
	if pipelineVersion == "" {
		pipelineVersion = RecallBenchmarkServicePipelineVersion
	}
	engine := newRecallPipelineEngine(svc.retrieval(), recallPipelineOptions{
		pipelineVersion: pipelineVersion,
		scoringConfig:   c.ScoringConfig,
	})
	trace, err := engine.Run(ctx, RecallQuery{
		WorkspaceID: svc.workspaceID,
		Peer:        peer,
		Query:       query,
		SessionKey:  sessionKey,
		ScopeID:     normalizeMemoryScope(c.ScopeID, ""),
		Limit:       limit,
		MaxTokens:   c.MaxTokens,
	})
	if err != nil {
		return RecallBenchmarkCase{}, fmt.Errorf("goncho recall benchmark: case %q run recall: %w", id, err)
	}
	return RecallBenchmarkCase{
		ID:                    id,
		Ability:               c.Ability,
		Scale:                 c.Scale,
		ConversationID:        c.ConversationID,
		IdealAnswer:           c.IdealAnswer,
		Rubric:                append([]string(nil), c.Rubric...),
		Trace:                 trace,
		RelevantIDs:           relevantIDs,
		ContextContains:       append([]string(nil), c.ContextContains...),
		RequiredEvidenceKinds: append([]string(nil), c.RequiredEvidenceKinds...),
		ExpectedNoAnswer:      c.ExpectedNoAnswer,
		Latency:               time.Since(started),
	}, nil
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
	rubricContextScore, rubricContextMatches := recallBenchmarkRubricContextCoverage(c.Trace, c.Rubric)
	caseReport := RecallBenchmarkCaseReport{
		ID:                     id,
		Ability:                normalizeRecallBenchmarkAbility(c.Ability),
		Scale:                  strings.TrimSpace(c.Scale),
		ConversationID:         strings.TrimSpace(c.ConversationID),
		Question:               strings.TrimSpace(c.Trace.Query.Query),
		IdealAnswer:            strings.TrimSpace(c.IdealAnswer),
		Rubric:                 append([]string(nil), c.Rubric...),
		TraceID:                c.Trace.TraceID,
		PipelineVersion:        c.Trace.PipelineVersion,
		ScoringConfigVersion:   c.Trace.ScoringConfig.Version,
		RelevantIDs:            append([]string(nil), c.RelevantIDs...),
		RequiredEvidenceKinds:  requiredEvidenceKinds,
		ExpectedNoAnswer:       c.ExpectedNoAnswer,
		RubricContextScore:     rubricContextScore,
		RubricContextMatches:   rubricContextMatches,
		CandidateMemoryIDs:     candidateIDs,
		SelectedMemoryIDs:      selectedIDs,
		SelectedContext:        recallBenchmarkSelectedContext(c.Trace),
		CandidateEvidenceKinds: recallBenchmarkEvidenceKinds(c.Trace.Candidates),
		SelectedEvidenceKinds:  recallBenchmarkEvidenceKinds(c.Trace.Selected),
		TopEvidenceKinds:       recallBenchmarkTopEvidenceKinds(c.Trace.Selected),
		RecallAt5:              recallBenchmarkCaseRecallAtK(candidateIDs, selectedIDs, c.RelevantIDs, c.ExpectedNoAnswer, 5),
		RecallAt10:             recallBenchmarkCaseRecallAtK(candidateIDs, selectedIDs, c.RelevantIDs, c.ExpectedNoAnswer, 10),
		ContextSatisfied:       recallBenchmarkContextSatisfied(c.Trace, c.RelevantIDs, c.ContextContains, c.ExpectedNoAnswer),
		ProvenanceSatisfied:    len(requiredEvidenceKinds) > 0 && recallBenchmarkProvenanceSatisfied(c.Trace, c.RelevantIDs, requiredEvidenceKinds),
		TokenBudget:            budget,
		SelectedTokens:         selectedTokens,
		TokenBudgetWithin:      budget <= 0 || selectedTokens <= budget,
		LatencyMS:              int(c.Latency / time.Millisecond),
		WarningCodes:           recallBenchmarkWarningCodes(c.Trace.Warnings),
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
	if len(c.RelevantIDs) == 0 && !c.ExpectedNoAnswer {
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
	return textutil.UniqueLowerTrimmed(kinds, true)
}

func recallBenchmarkEvidenceKinds(items []ScoredRecallCandidate) []string {
	kinds := []string{}
	for _, item := range items {
		for _, evidence := range item.Candidate.Provenance {
			kinds = append(kinds, evidence.Kind)
		}
	}
	return normalizeRecallBenchmarkEvidenceKinds(kinds)
}

func recallBenchmarkTopEvidenceKinds(items []ScoredRecallCandidate) []string {
	if len(items) == 0 {
		return nil
	}
	kinds := make([]string, 0, len(items[0].Candidate.Provenance))
	for _, evidence := range items[0].Candidate.Provenance {
		kinds = append(kinds, evidence.Kind)
	}
	return normalizeRecallBenchmarkEvidenceKinds(kinds)
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

func recallBenchmarkCaseRecallAtK(candidateIDs, selectedIDs, relevantIDs []string, expectedNoAnswer bool, k int) float64 {
	if expectedNoAnswer {
		if len(selectedIDs) == 0 {
			return 1
		}
		return 0
	}
	return recallAtK(candidateIDs, relevantIDs, k)
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

func recallBenchmarkRubricContextCoverage(trace RecallTrace, rubric []string) (float64, []string) {
	if len(rubric) == 0 {
		return 0, nil
	}
	contextTokens := map[string]struct{}{}
	for _, item := range trace.Selected {
		for _, token := range recallBenchmarkRubricTokens(item.Candidate.Content) {
			contextTokens[token] = struct{}{}
		}
	}
	matched := []string{}
	denom := 0
	for _, item := range rubric {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		tokens := recallBenchmarkRubricTokens(item)
		if len(tokens) == 0 {
			continue
		}
		denom++
		allPresent := true
		for _, token := range tokens {
			if _, ok := contextTokens[token]; !ok {
				allPresent = false
				break
			}
		}
		if allPresent {
			matched = append(matched, item)
		}
	}
	if denom == 0 {
		return 0, nil
	}
	return roundRecallFloat(float64(len(matched)) / float64(denom)), matched
}

func recallBenchmarkRubricTokens(text string) []string {
	stop := map[string]struct{}{
		"about": {}, "answer": {}, "contain": {}, "contains": {}, "correct": {}, "correctly": {}, "from": {}, "identify": {}, "identifies": {}, "include": {}, "includes": {}, "mention": {}, "mentions": {}, "name": {}, "names": {}, "note": {}, "project": {}, "say": {}, "says": {}, "state": {}, "states": {}, "that": {}, "the": {}, "with": {},
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, token := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		token = strings.TrimSpace(token)
		if len(token) < 2 {
			continue
		}
		if _, skip := stop[token]; skip {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out
}

func recallBenchmarkSelectedContext(trace RecallTrace) string {
	return strings.TrimSpace((&RecallProjector{}).ProjectContext(trace).Representation)
}

func recallBenchmarkContextSatisfied(trace RecallTrace, relevantIDs []string, contains []string, expectedNoAnswer bool) bool {
	if expectedNoAnswer {
		return len(trace.Selected) == 0 && len(contains) == 0
	}
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
