package goncho

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/maputil"
	"github.com/TrebuchetDynamics/goncho/service/internal/recallscore"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

const defaultRecallPipelineVersion = "goncho-recall-v1"

var defaultRecallWeights = map[string]float64{
	"keyword":    0.25,
	"semantic":   0.25,
	"graph":      0.20,
	"fact":       0.15,
	"recency":    0.07,
	"importance": 0.05,
	"scope":      0.03,
}

const recallGraphCoverageBonus = 0.05

type RecallEngine interface {
	Run(ctx context.Context, q RecallQuery) (RecallTrace, error)
}

type recallCandidateGenerator interface {
	Generate(ctx context.Context, q RecallQuery) ([]RecallCandidate, error)
}

type recallWarningReporter interface {
	RecallWarnings() []RecallWarning
}

type recallPipelineOptions struct {
	pipelineVersion string
	scoringConfig   RecallScoringConfig
	now             func() time.Time
}

type recallPipelineEngine struct {
	generator recallCandidateGenerator
	opts      recallPipelineOptions
}

func newRecallPipelineEngine(generator recallCandidateGenerator, opts recallPipelineOptions) *recallPipelineEngine {
	if opts.pipelineVersion == "" {
		opts.pipelineVersion = defaultRecallPipelineVersion
	}
	if opts.now == nil {
		opts.now = time.Now
	}
	opts.scoringConfig = normalizeRecallScoringConfig(opts.scoringConfig)
	return &recallPipelineEngine{generator: generator, opts: opts}
}

func (e *recallPipelineEngine) Run(ctx context.Context, q RecallQuery) (RecallTrace, error) {
	if e == nil || e.generator == nil {
		return RecallTrace{}, errors.New("goncho recall: nil candidate generator")
	}
	if err := ctx.Err(); err != nil {
		return RecallTrace{}, err
	}
	candidates, err := e.generator.Generate(ctx, q)
	if err != nil {
		return RecallTrace{}, err
	}
	warnings := recallWarningsFromGenerator(e.generator)
	scored := e.score(q, candidates)
	selected, rejected, selectWarnings := e.selectCandidates(q, scored)
	warnings = appendRecallWarnings(warnings, selectWarnings...)
	trace := RecallTrace{
		PipelineVersion:  e.opts.pipelineVersion,
		CreatedAt:        e.opts.now().UTC(),
		Query:            q,
		ScoringConfig:    cloneRecallScoringConfig(e.opts.scoringConfig),
		VoiceDiagnostics: buildRecallVoiceDiagnostics(scored, selected, e.opts.scoringConfig),
		Candidates:       scored,
		Selected:         selected,
		Rejected:         rejected,
		Warnings:         warnings,
	}
	trace.TraceID = recallTraceID(trace)
	return trace, nil
}

func (e *recallPipelineEngine) score(q RecallQuery, candidates []RecallCandidate) []ScoredRecallCandidate {
	now := e.opts.now().UTC()
	out := make([]ScoredRecallCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		score := RecallScore{
			KeywordScore:    roundRecallFloat(maxEvidenceScore(candidate.Provenance, "keyword", keywordRecallScore(candidate.Content, q.Query))),
			SemanticScore:   roundRecallFloat(maxEvidenceScore(candidate.Provenance, "semantic", 0)),
			GraphScore:      roundRecallFloat(maxEvidenceScore(candidate.Provenance, "graph", 0)),
			FactScore:       roundRecallFloat(maxEvidenceScore(candidate.Provenance, "fact", 0)),
			RecencyScore:    roundRecallFloat(recallRecencyScore(candidate.CreatedAt, now)),
			ImportanceScore: roundRecallFloat(clampRecall(candidate.Importance)),
			ScopeScore:      roundRecallFloat(scopeRecallScore(q, candidate)),
		}
		score.RRFScore = roundRecallFloat(0)
		score.FinalScore = roundRecallFloat(weightedRecallScore(score, e.opts.scoringConfig.Weights))
		out = append(out, ScoredRecallCandidate{Candidate: candidate, Score: score})
	}
	addRecallRRF(out, e.opts.scoringConfig)
	for i := range out {
		out[i].Score.FinalScore = roundRecallFloat(out[i].Score.FinalScore + out[i].Score.RRFScore)
		out[i].Score.WhySelected = []string{
			fmt.Sprintf("final_score=%.6f", out[i].Score.FinalScore),
			fmt.Sprintf("scoring_config=%s", e.opts.scoringConfig.Version),
		}
		if out[i].Score.FactScore > 0 {
			out[i].Score.WhySelected = append(out[i].Score.WhySelected, fmt.Sprintf("fact_score=%.6f", out[i].Score.FactScore))
		}
	}
	sortScoredRecall(out)
	return out
}

func (e *recallPipelineEngine) selectCandidates(q RecallQuery, scored []ScoredRecallCandidate) ([]ScoredRecallCandidate, []RejectedRecallCandidate, []RecallWarning) {
	limit := q.Limit
	if limit <= 0 {
		limit = 5
	}
	budget := e.opts.scoringConfig.TokenBudget
	if q.MaxTokens > 0 {
		budget = q.MaxTokens
	}
	eligible := make([]ScoredRecallCandidate, 0, len(scored))
	rejected := make([]RejectedRecallCandidate, 0)
	scopeRejected := 0
	for _, item := range scored {
		if recallScopeMismatch(q, item.Candidate) {
			scopeRejected++
			rejected = append(rejected, RejectedRecallCandidate{
				Candidate: item.Candidate,
				Score:     item.Score,
				Reason:    RecallRejectScopeMismatch,
				WhyRejected: []string{
					fmt.Sprintf("candidate_scope=%s", item.Candidate.ScopeID),
					fmt.Sprintf("query_scope=%s", q.ScopeID),
				},
			})
			continue
		}
		eligible = append(eligible, item)
	}

	var warnings []RecallWarning
	if len(scored) > 0 && len(eligible) == 0 && scopeRejected == len(scored) {
		warnings = append(warnings, RecallWarning{
			Code:     RecallWarningScopeExcludedAllCandidates,
			Stage:    RecallStageSelect,
			Severity: RecallWarningDegraded,
			Message:  "scope filter excluded every recall candidate",
			Evidence: map[string]string{"scope_id": q.ScopeID},
		})
	}
	if recallQueryAsksCurrentTruth(q.Query) && recallHasSupersededEvidence(eligible) {
		warnings = appendRecallWarnings(warnings, RecallWarning{
			Code:     RecallWarningSupersededEvidenceObserved,
			Stage:    RecallStageSelect,
			Severity: RecallWarningInfo,
			Message:  "recall candidates include superseded evidence; current-truth routing adjusted selection",
		})
	}

	selected := make([]ScoredRecallCandidate, 0, min(limit, len(eligible)))
	remaining := sliceutil.Clone(eligible)
	usedTokens := 0
	for len(selected) < limit && len(remaining) > 0 {
		bestIdx := 0
		bestScore := math.Inf(-1)
		for i := range remaining {
			penalty := recallDiversityPenalty(remaining[i], selected, e.opts.scoringConfig)
			coverageBonus := recallCoverageBonus(remaining[i], selected)
			temporalAdjustment := recallTemporalAdjustment(remaining[i], q.Query)
			speakerAdjustment := recallSpeakerAdjustment(remaining[i], q.Query)
			effectiveScore := remaining[i].Score.FinalScore - penalty + coverageBonus + temporalAdjustment + speakerAdjustment
			if effectiveScore > bestScore || (effectiveScore == bestScore && compareScoredRecall(remaining[i], remaining[bestIdx]) < 0) {
				bestScore = effectiveScore
				bestIdx = i
			}
		}
		chosen := remaining[bestIdx]
		coverageBonus := recallCoverageBonus(chosen, selected)
		temporalAdjustment := recallTemporalAdjustment(chosen, q.Query)
		speakerAdjustment := recallSpeakerAdjustment(chosen, q.Query)
		chosen.Score.DiversityPenalty = roundRecallFloat(recallDiversityPenalty(chosen, selected, e.opts.scoringConfig))
		chosen.Score.FinalScore = roundRecallFloat(chosen.Score.FinalScore - chosen.Score.DiversityPenalty + coverageBonus + temporalAdjustment + speakerAdjustment)
		chosen.Score.WhySelected = append(chosen.Score.WhySelected, fmt.Sprintf("diversity_penalty=%.6f", chosen.Score.DiversityPenalty))
		if coverageBonus > 0 {
			chosen.Score.WhySelected = append(chosen.Score.WhySelected, fmt.Sprintf("coverage_bonus=%.6f", coverageBonus))
		}
		if temporalAdjustment != 0 {
			chosen.Score.WhySelected = append(chosen.Score.WhySelected, fmt.Sprintf("temporal_adjustment=%.6f", temporalAdjustment))
		}
		if speakerAdjustment > 0 {
			chosen.Score.WhySelected = append(chosen.Score.WhySelected, fmt.Sprintf("speaker_adjustment=%.6f", speakerAdjustment))
		}
		tokenCost := estimateRecallTokens(chosen.Candidate.Content)
		if budget > 0 && usedTokens+tokenCost > budget {
			rejected = append(rejected, RejectedRecallCandidate{
				Candidate: chosen.Candidate,
				Score:     chosen.Score,
				Reason:    RecallRejectTokenBudget,
				WhyRejected: []string{
					fmt.Sprintf("used_tokens=%d", usedTokens),
					fmt.Sprintf("candidate_tokens=%d", tokenCost),
					fmt.Sprintf("token_budget=%d", budget),
				},
			})
			warnings = appendRecallWarnings(warnings, RecallWarning{
				Code:     RecallWarningTokenBudgetTruncated,
				Stage:    RecallStageSelect,
				Severity: RecallWarningDegraded,
				Message:  "token budget truncated selected recall context",
				Evidence: map[string]string{"token_budget": fmt.Sprintf("%d", budget)},
			})
			remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
			continue
		}
		usedTokens += tokenCost
		selected = append(selected, chosen)
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}
	for _, item := range remaining {
		item.Score.DiversityPenalty = roundRecallFloat(recallDiversityPenalty(item, selected, e.opts.scoringConfig))
		item.Score.FinalScore = roundRecallFloat(item.Score.FinalScore - item.Score.DiversityPenalty)
		rejected = append(rejected, RejectedRecallCandidate{
			Candidate: item.Candidate,
			Score:     item.Score,
			Reason:    RecallRejectNotSelected,
			WhyRejected: []string{
				fmt.Sprintf("limit=%d", limit),
			},
		})
	}
	sortRejectedRecall(rejected)
	return selected, rejected, warnings
}

func normalizeRecallScoringConfig(config RecallScoringConfig) RecallScoringConfig {
	if config.Version == "" {
		config.Version = "default-v1"
	}
	if len(config.Weights) == 0 {
		config.Weights = maputil.CloneStringFloat64(defaultRecallWeights)
	}
	config = cloneRecallScoringConfig(config)
	if config.RRFK <= 0 {
		config.RRFK = 60
	}
	if config.MMRLambda <= 0 || config.MMRLambda > 1 {
		config.MMRLambda = 0.7
	}
	return config
}

func cloneRecallScoringConfig(config RecallScoringConfig) RecallScoringConfig {
	if config.Weights != nil {
		config.Weights = maputil.CloneStringFloat64(config.Weights)
	}
	if config.DiversityKeys != nil {
		config.DiversityKeys = cloneStrings(config.DiversityKeys)
	}
	return config
}

func recallWarningsFromGenerator(generator recallCandidateGenerator) []RecallWarning {
	reporter, ok := generator.(recallWarningReporter)
	if !ok {
		return []RecallWarning{}
	}
	warnings := reporter.RecallWarnings()
	if warnings == nil {
		return []RecallWarning{}
	}
	return warnings
}

func appendRecallWarnings(existing []RecallWarning, warnings ...RecallWarning) []RecallWarning {
	seen := make(map[string]struct{}, len(existing)+len(warnings))
	out := make([]RecallWarning, 0, len(existing)+len(warnings))
	for _, warning := range append(existing, warnings...) {
		if warning.Code == "" {
			continue
		}
		key := warning.Stage + "\x00" + warning.Code
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, warning)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Stage != out[j].Stage {
			return out[i].Stage < out[j].Stage
		}
		return out[i].Code < out[j].Code
	})
	return out
}

func maxEvidenceScore(items []EvidenceItem, kind string, fallback float64) float64 {
	score := fallback
	for _, item := range items {
		if item.Kind != kind {
			continue
		}
		if item.Score > score {
			score = item.Score
		}
	}
	return clampRecall(score)
}

func keywordRecallScore(content, query string) float64 { return recallscore.Keyword(content, query) }

func recallRecencyScore(createdAt, now time.Time) float64 {
	return recallscore.Recency(createdAt, now, defaultDecayHalfLife)
}

func scopeRecallScore(q RecallQuery, candidate RecallCandidate) float64 {
	if q.ScopeID == "" {
		if candidate.ScopeID == "" {
			return 0.5
		}
		return 1
	}
	switch {
	case candidate.ScopeID == q.ScopeID:
		return 1
	case candidate.ScopeID == "":
		return 0.5
	default:
		return 0
	}
}

func recallScopeMismatch(q RecallQuery, candidate RecallCandidate) bool {
	return q.ScopeID != "" && candidate.ScopeID != "" && candidate.ScopeID != q.ScopeID
}

func buildRecallVoiceDiagnostics(scored, selected []ScoredRecallCandidate, config RecallScoringConfig) []RecallVoiceDiagnostic {
	type voiceAccessor struct {
		name  string
		score func(RecallScore) float64
	}
	voices := []voiceAccessor{
		{"keyword", func(s RecallScore) float64 { return s.KeywordScore }},
		{"semantic", func(s RecallScore) float64 { return s.SemanticScore }},
		{"graph", func(s RecallScore) float64 { return s.GraphScore }},
		{"fact", func(s RecallScore) float64 { return s.FactScore }},
		{"recency", func(s RecallScore) float64 { return s.RecencyScore }},
		{"importance", func(s RecallScore) float64 { return s.ImportanceScore }},
		{"scope", func(s RecallScore) float64 { return s.ScopeScore }},
	}
	diags := make([]RecallVoiceDiagnostic, 0, len(voices))
	for _, v := range voices {
		weight := config.Weights[v.name]
		enabled := weight > 0
		var candWith int
		var maxScore, minScore, sumScore float64
		minScore = -1 // sentinel: first non-zero will set it
		for _, c := range scored {
			s := v.score(c.Score)
			if s > 0 {
				candWith++
			}
			if minScore < 0 || s < minScore {
				minScore = s
			}
			if s > maxScore {
				maxScore = s
			}
			sumScore += s
		}
		var avgScore float64
		if len(scored) > 0 {
			avgScore = sumScore / float64(len(scored))
		}
		if minScore < 0 {
			minScore = 0
		}
		selectedCount := 0
		for _, s := range selected {
			if v.score(s.Score) > 0 {
				selectedCount++
			}
		}
		diags = append(diags, RecallVoiceDiagnostic{
			Name:           v.name,
			Enabled:        enabled,
			Weight:         weight,
			CandidatesWith: candWith,
			MaxScore:       roundRecallFloat(maxScore),
			MinScore:       roundRecallFloat(minScore),
			AvgScore:       roundRecallFloat(avgScore),
			SelectedCount:  selectedCount,
		})
	}
	return diags
}

func weightedRecallScore(score RecallScore, weights map[string]float64) float64 {
	return clampRecall(
		weights["keyword"]*score.KeywordScore +
			weights["semantic"]*score.SemanticScore +
			weights["graph"]*score.GraphScore +
			weights["fact"]*score.FactScore +
			weights["recency"]*score.RecencyScore +
			weights["importance"]*score.ImportanceScore +
			weights["scope"]*score.ScopeScore,
	)
}

func addRecallRRF(items []ScoredRecallCandidate, config RecallScoringConfig) {
	if len(items) == 0 {
		return
	}
	signals := []struct {
		name  string
		score func(RecallScore) float64
	}{
		{"keyword", func(s RecallScore) float64 { return s.KeywordScore }},
		{"semantic", func(s RecallScore) float64 { return s.SemanticScore }},
		{"graph", func(s RecallScore) float64 { return s.GraphScore }},
		{"fact", func(s RecallScore) float64 { return s.FactScore }},
		{"recency", func(s RecallScore) float64 { return s.RecencyScore }},
		{"importance", func(s RecallScore) float64 { return s.ImportanceScore }},
		{"scope", func(s RecallScore) float64 { return s.ScopeScore }},
	}
	for _, signal := range signals {
		weight := config.Weights[signal.name]
		if weight == 0 {
			continue
		}
		indexes := make([]int, len(items))
		for i := range items {
			indexes[i] = i
		}
		sort.SliceStable(indexes, func(i, j int) bool {
			left := items[indexes[i]]
			right := items[indexes[j]]
			if signal.score(left.Score) != signal.score(right.Score) {
				return signal.score(left.Score) > signal.score(right.Score)
			}
			return left.Candidate.MemoryID < right.Candidate.MemoryID
		})
		for rank, idx := range indexes {
			items[idx].Score.RRFScore += weight / float64(config.RRFK+rank+1)
		}
	}
	for i := range items {
		items[i].Score.RRFScore = roundRecallFloat(items[i].Score.RRFScore)
	}
}

const recallTemporalCurrentBonus = 0.08
const recallTemporalSupersededPenalty = 0.20
const recallSpeakerMatchBonus = 0.12

func recallTemporalAdjustment(candidate ScoredRecallCandidate, query string) float64 {
	if !recallQueryAsksCurrentTruth(query) {
		return 0
	}
	for _, evidence := range candidate.Candidate.Provenance {
		if evidence.Kind != "temporal" {
			continue
		}
		note := strings.ToLower(strings.TrimSpace(evidence.Note))
		if strings.Contains(note, "superseded_by=") || strings.Contains(note, "superseded") {
			return -recallTemporalSupersededPenalty
		}
		if strings.Contains(note, "current_fact") || strings.Contains(note, "valid_now") {
			return recallTemporalCurrentBonus
		}
	}
	return 0
}

func recallQueryAsksCurrentTruth(query string) bool {
	query = strings.ToLower(query)
	for _, marker := range []string{" now", "current", "currently", "latest", "today"} {
		if strings.Contains(query, marker) {
			return true
		}
	}
	return false
}

func recallHasSupersededEvidence(candidates []ScoredRecallCandidate) bool {
	for _, candidate := range candidates {
		for _, evidence := range candidate.Candidate.Provenance {
			if evidence.Kind != "temporal" {
				continue
			}
			note := strings.ToLower(strings.TrimSpace(evidence.Note))
			if strings.Contains(note, "superseded_by=") || strings.Contains(note, "superseded") {
				return true
			}
		}
	}
	return false
}

func recallSpeakerAdjustment(candidate ScoredRecallCandidate, query string) float64 {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return 0
	}
	speaker := recallCandidateSpeaker(candidate.Candidate)
	if speaker == "" {
		return 0
	}
	targets := recallQuerySpeakerTargets(query)
	if len(targets) > 0 {
		for _, target := range targets {
			if target == speaker {
				return recallSpeakerMatchBonus
			}
		}
		return 0
	}
	if strings.Contains(query, speaker) {
		return recallSpeakerMatchBonus
	}
	return 0
}

func recallCandidateSpeaker(candidate RecallCandidate) string {
	for _, evidence := range candidate.Provenance {
		if evidence.Kind != "speaker" {
			continue
		}
		speaker := strings.ToLower(strings.TrimSpace(evidence.Source))
		if speaker == "" {
			note := strings.ToLower(strings.TrimSpace(evidence.Note))
			if strings.HasPrefix(note, "speaker=") {
				speaker = strings.TrimSpace(strings.TrimPrefix(note, "speaker="))
			}
		}
		if speaker != "" {
			return speaker
		}
	}
	return strings.ToLower(strings.TrimSpace(candidate.AgentID))
}

func recallQuerySpeakerTargets(query string) []string {
	tokens := recallQueryTokens(query)
	for i := 0; i+2 < len(tokens); i++ {
		if tokens[i] == "did" && tokens[i+2] == "say" {
			return []string{tokens[i+1]}
		}
		if tokens[i] == "has" && tokens[i+2] == "said" {
			return []string{tokens[i+1]}
		}
	}
	return nil
}

func recallQueryTokens(query string) []string {
	query = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return ' '
	}, strings.ToLower(query))
	return strings.Fields(query)
}

func recallCoverageBonus(candidate ScoredRecallCandidate, selected []ScoredRecallCandidate) float64 {
	if len(selected) == 0 {
		return 0
	}
	for _, evidence := range candidate.Candidate.Provenance {
		if evidence.Kind != "graph" {
			continue
		}
		for _, item := range selected {
			if item.Candidate.MemoryID == "" || item.Candidate.MemoryID == candidate.Candidate.MemoryID {
				continue
			}
			if evidence.Source == item.Candidate.MemoryID || strings.HasPrefix(evidence.Note, item.Candidate.MemoryID+" -> ") {
				return recallGraphCoverageBonus
			}
		}
	}
	return 0
}

func recallDiversityPenalty(candidate ScoredRecallCandidate, selected []ScoredRecallCandidate, config RecallScoringConfig) float64 {
	if len(selected) == 0 || len(config.DiversityKeys) == 0 {
		return 0
	}
	collisions := 0
	for _, key := range config.DiversityKeys {
		value := recallDiversityValue(candidate.Candidate, key)
		if value == "" {
			continue
		}
		for _, item := range selected {
			if value == recallDiversityValue(item.Candidate, key) {
				collisions++
				break
			}
		}
	}
	if collisions == 0 {
		return 0
	}
	return clampRecall(float64(collisions) * (1 - config.MMRLambda))
}

func recallDiversityValue(candidate RecallCandidate, key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "session_id":
		return candidate.SessionID
	case "source_type":
		return candidate.SourceType
	case "agent_id":
		return candidate.AgentID
	case "scope_id":
		return candidate.ScopeID
	default:
		return ""
	}
}

func estimateRecallTokens(content string) int {
	n := len(strings.Fields(content))
	if n == 0 && strings.TrimSpace(content) != "" {
		return 1
	}
	return n
}

func sortScoredRecall(items []ScoredRecallCandidate) {
	sort.SliceStable(items, func(i, j int) bool {
		return compareScoredRecall(items[i], items[j]) < 0
	})
}

func compareScoredRecall(left, right ScoredRecallCandidate) int {
	if left.Score.FinalScore != right.Score.FinalScore {
		if left.Score.FinalScore > right.Score.FinalScore {
			return -1
		}
		return 1
	}
	if left.Score.RRFScore != right.Score.RRFScore {
		if left.Score.RRFScore > right.Score.RRFScore {
			return -1
		}
		return 1
	}
	if left.Candidate.MemoryID < right.Candidate.MemoryID {
		return -1
	}
	if left.Candidate.MemoryID > right.Candidate.MemoryID {
		return 1
	}
	return 0
}

func sortRejectedRecall(items []RejectedRecallCandidate) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Reason != items[j].Reason {
			return items[i].Reason < items[j].Reason
		}
		return items[i].Candidate.MemoryID < items[j].Candidate.MemoryID
	})
}

func recallTraceID(trace RecallTrace) string {
	view := struct {
		Query           RecallQuery        `json:"query"`
		CandidateIDs    []string           `json:"candidate_ids"`
		ScoringVersion  string             `json:"scoring_version"`
		PipelineVersion string             `json:"pipeline_version"`
		Weights         map[string]float64 `json:"weights"`
		DiversityKeys   []string           `json:"diversity_keys,omitempty"`
		RRFK            int                `json:"rrf_k"`
		MMRLambda       float64            `json:"mmr_lambda"`
		TokenBudget     int                `json:"token_budget,omitempty"`
	}{Query: trace.Query, ScoringVersion: trace.ScoringConfig.Version, PipelineVersion: trace.PipelineVersion, Weights: trace.ScoringConfig.Weights, DiversityKeys: trace.ScoringConfig.DiversityKeys, RRFK: trace.ScoringConfig.RRFK, MMRLambda: trace.ScoringConfig.MMRLambda, TokenBudget: trace.ScoringConfig.TokenBudget}
	for _, item := range trace.Candidates {
		view.CandidateIDs = append(view.CandidateIDs, item.Candidate.MemoryID)
	}
	raw, _ := json.Marshal(view)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func clampRecall(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func roundRecallFloat(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}
