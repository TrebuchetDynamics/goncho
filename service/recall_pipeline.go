package goncho

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/hashutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/maputil"
	"github.com/TrebuchetDynamics/goncho/service/internal/recallscore"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
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
			KeywordScore:    roundRecallFloat(maxEvidenceScore(candidate.Provenance, "keyword", recallscore.Keyword(candidate.Content, q.Query))),
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
	flow := newRecallSelectionFlow(q, scored, e.opts.scoringConfig)
	return flow.run()
}

type recallSelectionFlow struct {
	query      RecallQuery
	config     RecallScoringConfig
	policy     recallSelectionPolicy
	eligible   []ScoredRecallCandidate
	remaining  []ScoredRecallCandidate
	selected   []ScoredRecallCandidate
	rejected   []RejectedRecallCandidate
	warnings   []RecallWarning
	usedTokens int
}

func newRecallSelectionFlow(q RecallQuery, scored []ScoredRecallCandidate, config RecallScoringConfig) recallSelectionFlow {
	policy := recallSelectionPolicyFor(q, config)
	eligible, rejected, warnings := recallScopeSelectionInputs(q, scored)
	return recallSelectionFlow{
		query:     q,
		config:    config,
		policy:    policy,
		eligible:  eligible,
		remaining: sliceutil.Clone(eligible),
		selected:  make([]ScoredRecallCandidate, 0, min(policy.Limit, len(eligible))),
		rejected:  rejected,
		warnings:  warnings,
	}
}

func (f *recallSelectionFlow) run() ([]ScoredRecallCandidate, []RejectedRecallCandidate, []RecallWarning) {
	f.addTemporalWarnings()
	for len(f.selected) < f.policy.Limit && len(f.remaining) > 0 {
		f.selectNextCandidate()
	}
	f.rejectUnselectedRemainder()
	return f.selected, f.rejected, f.warnings
}

func (f *recallSelectionFlow) addTemporalWarnings() {
	if recallQueryAsksCurrentTruth(f.query.Query) && recallHasSupersededEvidence(f.eligible) {
		f.warnings = appendRecallWarnings(f.warnings, RecallWarning{
			Code:     RecallWarningSupersededEvidenceObserved,
			Stage:    RecallStageSelect,
			Severity: RecallWarningInfo,
			Message:  "recall candidates include superseded evidence; current-truth routing adjusted selection",
		})
	}
}

func (f *recallSelectionFlow) selectNextCandidate() {
	bestIdx := recallBestSelectionIndex(f.remaining, f.selected, f.query.Query, f.config)
	chosen := applyRecallSelectionAdjustment(f.remaining[bestIdx], recallSelectionAdjustmentFor(f.remaining[bestIdx], f.selected, f.query.Query, f.config), true)
	action := recallSelectionActionFor(chosen, f.usedTokens, f.policy)
	if action.RejectReason != "" {
		f.rejected = append(f.rejected, recallRejectedCandidate(action.Candidate, action.RejectReason, action.RejectWhy))
		if action.Warning.Code != "" {
			f.warnings = appendRecallWarnings(f.warnings, action.Warning)
		}
		f.dropRemaining(bestIdx)
		return
	}
	f.usedTokens += action.TokenCost
	f.selected = append(f.selected, action.Candidate)
	f.dropRemaining(bestIdx)
}

type recallSelectionAction struct {
	Candidate    ScoredRecallCandidate
	TokenCost    int
	RejectReason string
	RejectWhy    []string
	Warning      RecallWarning
}

func recallSelectionActionFor(chosen ScoredRecallCandidate, usedTokens int, policy recallSelectionPolicy) recallSelectionAction {
	tokenCost := estimateRecallTokens(chosen.Candidate.Content)
	action := recallSelectionAction{Candidate: chosen, TokenCost: tokenCost}
	if !policy.FitsTokenBudget(usedTokens, tokenCost) {
		action.RejectReason = RecallRejectTokenBudget
		action.RejectWhy = policy.TokenBudgetRejectionReasons(usedTokens, tokenCost)
		action.Warning = policy.TokenBudgetWarning()
	}
	return action
}

func (f *recallSelectionFlow) dropRemaining(idx int) {
	f.remaining = append(f.remaining[:idx], f.remaining[idx+1:]...)
}

func (f *recallSelectionFlow) rejectUnselectedRemainder() {
	for _, item := range f.remaining {
		item = applyRecallSelectionAdjustment(item, recallRejectedSelectionAdjustmentFor(item, f.selected, f.config), false)
		f.rejected = append(f.rejected, recallRejectedCandidate(item, RecallRejectNotSelected, []string{
			fmt.Sprintf("limit=%d", f.policy.Limit),
		}))
	}
}

func recallScopeSelectionInputs(q RecallQuery, scored []ScoredRecallCandidate) ([]ScoredRecallCandidate, []RejectedRecallCandidate, []RecallWarning) {
	eligible := make([]ScoredRecallCandidate, 0, len(scored))
	rejected := make([]RejectedRecallCandidate, 0)
	scopeRejected := 0
	for _, item := range scored {
		if recallScopeMismatch(q, item.Candidate) {
			scopeRejected++
			rejected = append(rejected, recallRejectedCandidate(item, RecallRejectScopeMismatch, []string{
				fmt.Sprintf("candidate_scope=%s", item.Candidate.ScopeID),
				fmt.Sprintf("query_scope=%s", q.ScopeID),
			}))
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
	return eligible, rejected, warnings
}

type recallSelectionPolicy struct {
	Limit       int
	TokenBudget int
}

func recallSelectionPolicyFor(q RecallQuery, config RecallScoringConfig) recallSelectionPolicy {
	policy := recallSelectionPolicy{
		Limit:       limitutil.Default(q.Limit, 5),
		TokenBudget: config.TokenBudget,
	}
	if q.MaxTokens > 0 {
		policy.TokenBudget = q.MaxTokens
	}
	return policy
}

func (p recallSelectionPolicy) FitsTokenBudget(usedTokens, candidateTokens int) bool {
	return p.TokenBudget <= 0 || usedTokens+candidateTokens <= p.TokenBudget
}

func (p recallSelectionPolicy) TokenBudgetRejectionReasons(usedTokens, candidateTokens int) []string {
	return []string{
		fmt.Sprintf("used_tokens=%d", usedTokens),
		fmt.Sprintf("candidate_tokens=%d", candidateTokens),
		fmt.Sprintf("token_budget=%d", p.TokenBudget),
	}
}

func (p recallSelectionPolicy) TokenBudgetWarning() RecallWarning {
	return RecallWarning{
		Code:     RecallWarningTokenBudgetTruncated,
		Stage:    RecallStageSelect,
		Severity: RecallWarningDegraded,
		Message:  "token budget truncated selected recall context",
		Evidence: map[string]string{"token_budget": fmt.Sprintf("%d", p.TokenBudget)},
	}
}

func recallRejectedCandidate(item ScoredRecallCandidate, reason string, why []string) RejectedRecallCandidate {
	return RejectedRecallCandidate{
		Candidate:   item.Candidate,
		Score:       item.Score,
		Reason:      reason,
		WhyRejected: sliceutil.Clone(why),
	}
}

type recallSelectionAdjustment struct {
	DiversityPenalty   float64
	CoverageBonus      float64
	TemporalAdjustment float64
	SpeakerAdjustment  float64
	EffectiveScore     float64
}

func recallBestSelectionIndex(remaining, selected []ScoredRecallCandidate, query string, config RecallScoringConfig) int {
	bestIdx := 0
	bestScore := math.Inf(-1)
	for i := range remaining {
		adjustment := recallSelectionAdjustmentFor(remaining[i], selected, query, config)
		if adjustment.EffectiveScore > bestScore || (adjustment.EffectiveScore == bestScore && compareScoredRecall(remaining[i], remaining[bestIdx]) < 0) {
			bestScore = adjustment.EffectiveScore
			bestIdx = i
		}
	}
	return bestIdx
}

func recallSelectionAdjustmentFor(item ScoredRecallCandidate, selected []ScoredRecallCandidate, query string, config RecallScoringConfig) recallSelectionAdjustment {
	adjustment := recallSelectionAdjustment{
		DiversityPenalty:   recallDiversityPenalty(item, selected, config),
		CoverageBonus:      recallCoverageBonus(item, selected),
		TemporalAdjustment: recallTemporalAdjustment(item, query),
		SpeakerAdjustment:  recallSpeakerAdjustment(item, query),
	}
	adjustment.EffectiveScore = item.Score.FinalScore - adjustment.DiversityPenalty + adjustment.CoverageBonus + adjustment.TemporalAdjustment + adjustment.SpeakerAdjustment
	return adjustment
}

func recallRejectedSelectionAdjustmentFor(item ScoredRecallCandidate, selected []ScoredRecallCandidate, config RecallScoringConfig) recallSelectionAdjustment {
	adjustment := recallSelectionAdjustment{DiversityPenalty: recallDiversityPenalty(item, selected, config)}
	adjustment.EffectiveScore = item.Score.FinalScore - adjustment.DiversityPenalty
	return adjustment
}

func applyRecallSelectionAdjustment(item ScoredRecallCandidate, adjustment recallSelectionAdjustment, includeAdjustmentReasons bool) ScoredRecallCandidate {
	item.Score.DiversityPenalty = roundRecallFloat(adjustment.DiversityPenalty)
	item.Score.FinalScore = roundRecallFloat(item.Score.FinalScore - item.Score.DiversityPenalty + adjustment.CoverageBonus + adjustment.TemporalAdjustment + adjustment.SpeakerAdjustment)
	item.Score.WhySelected = recallWhySelectedWithFinalScore(item.Score.WhySelected, item.Score.FinalScore)
	if !includeAdjustmentReasons {
		return item
	}
	item.Score.WhySelected = append(item.Score.WhySelected, fmt.Sprintf("diversity_penalty=%.6f", item.Score.DiversityPenalty))
	if adjustment.CoverageBonus > 0 {
		item.Score.WhySelected = append(item.Score.WhySelected, fmt.Sprintf("coverage_bonus=%.6f", adjustment.CoverageBonus))
	}
	if adjustment.TemporalAdjustment != 0 {
		item.Score.WhySelected = append(item.Score.WhySelected, fmt.Sprintf("temporal_adjustment=%.6f", adjustment.TemporalAdjustment))
	}
	if adjustment.SpeakerAdjustment > 0 {
		item.Score.WhySelected = append(item.Score.WhySelected, fmt.Sprintf("speaker_adjustment=%.6f", adjustment.SpeakerAdjustment))
	}
	return item
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
		config.DiversityKeys = sliceutil.Clone(config.DiversityKeys)
	}
	return config
}

func recallWarningsFromGenerator(generator recallCandidateGenerator) []RecallWarning {
	reporter, ok := generator.(recallWarningReporter)
	if !ok {
		return []RecallWarning{}
	}
	return cloneRecallWarnings(reporter.RecallWarnings())
}

func appendRecallWarnings(existing []RecallWarning, warnings ...RecallWarning) []RecallWarning {
	seen := make(map[string]struct{}, len(existing)+len(warnings))
	out := make([]RecallWarning, 0, len(existing)+len(warnings))
	for _, warning := range append(existing, warnings...) {
		if warning.Code == "" {
			continue
		}
		key := recallWarningDedupKey(warning)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, cloneRecallWarning(warning))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Stage != out[j].Stage {
			return out[i].Stage < out[j].Stage
		}
		return out[i].Code < out[j].Code
	})
	return out
}

func cloneRecallWarnings(warnings []RecallWarning) []RecallWarning {
	if warnings == nil {
		return []RecallWarning{}
	}
	out := make([]RecallWarning, len(warnings))
	for i, warning := range warnings {
		out[i] = cloneRecallWarning(warning)
	}
	return out
}

func cloneRecallWarning(warning RecallWarning) RecallWarning {
	if warning.Evidence != nil {
		warning.Evidence = maputil.CloneStringString(warning.Evidence)
	}
	return warning
}

func recallWarningDedupKey(warning RecallWarning) string {
	return warning.Stage + "\x00" + warning.Code + "\x00" + recallWarningEvidenceKey(warning.Evidence)
}

func recallWarningEvidenceKey(evidence map[string]string) string {
	if len(evidence) == 0 {
		return ""
	}
	keys := make([]string, 0, len(evidence))
	for key := range evidence {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(evidence[key])
		b.WriteByte('\x00')
	}
	return b.String()
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

func recallWhySelectedWithFinalScore(reasons []string, finalScore float64) []string {
	out := sliceutil.Clone(reasons)
	updated := fmt.Sprintf("final_score=%.6f", finalScore)
	for i, reason := range out {
		if strings.HasPrefix(reason, "final_score=") {
			out[i] = updated
			return out
		}
	}
	return append([]string{updated}, out...)
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
		indexes := rankedRecallSignalIndexes(items, signal.score)
		for rank, idx := range indexes {
			items[idx].Score.RRFScore += weight / float64(config.RRFK+rank+1)
		}
	}
	for i := range items {
		items[i].Score.RRFScore = roundRecallFloat(items[i].Score.RRFScore)
	}
}

func rankedRecallSignalIndexes(items []ScoredRecallCandidate, score func(RecallScore) float64) []int {
	indexes := make([]int, 0, len(items))
	for i := range items {
		if score(items[i].Score) <= 0 {
			continue
		}
		indexes = append(indexes, i)
	}
	sort.SliceStable(indexes, func(i, j int) bool {
		left := items[indexes[i]]
		right := items[indexes[j]]
		if score(left.Score) != score(right.Score) {
			return score(left.Score) > score(right.Score)
		}
		return left.Candidate.MemoryID < right.Candidate.MemoryID
	})
	return indexes
}

const recallTemporalCurrentBonus = 0.08
const recallTemporalSupersededPenalty = 0.20
const recallSpeakerMatchBonus = 0.12

type recallTemporalEvidenceState int

const (
	recallTemporalEvidenceUnknown recallTemporalEvidenceState = iota
	recallTemporalEvidenceCurrent
	recallTemporalEvidenceSuperseded
)

func recallTemporalAdjustment(candidate ScoredRecallCandidate, query string) float64 {
	if !recallQueryAsksCurrentTruth(query) {
		return 0
	}
	state := recallTemporalState(candidate.Candidate.Provenance)
	switch state {
	case recallTemporalEvidenceSuperseded:
		return -recallTemporalSupersededPenalty
	case recallTemporalEvidenceCurrent:
		return recallTemporalCurrentBonus
	default:
		return 0
	}
}

func recallTemporalState(items []EvidenceItem) recallTemporalEvidenceState {
	state := recallTemporalEvidenceUnknown
	for _, evidence := range items {
		if evidence.Kind != "temporal" {
			continue
		}
		switch recallTemporalEvidenceNoteState(evidence.Note) {
		case recallTemporalEvidenceSuperseded:
			return recallTemporalEvidenceSuperseded
		case recallTemporalEvidenceCurrent:
			state = recallTemporalEvidenceCurrent
		}
	}
	return state
}

func recallTemporalEvidenceNoteState(note string) recallTemporalEvidenceState {
	state := recallTemporalEvidenceUnknown
	for _, field := range recallTemporalEvidenceNoteFields(note) {
		field = strings.Trim(field, " .:()[]{}")
		switch {
		case strings.HasPrefix(field, "superseded_by=") && strings.TrimSpace(strings.TrimPrefix(field, "superseded_by=")) != "":
			return recallTemporalEvidenceSuperseded
		case field == "superseded":
			return recallTemporalEvidenceSuperseded
		case field == "current_fact" || field == "valid_now":
			state = recallTemporalEvidenceCurrent
		}
	}
	return state
}

func recallTemporalEvidenceNoteFields(note string) []string {
	note = textutil.LowerTrimmed(note)
	if note == "" {
		return nil
	}
	return strings.FieldsFunc(note, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', ',', ';':
			return true
		default:
			return false
		}
	})
}

var recallCurrentTruthIntentTokens = map[string]struct{}{
	"now":       {},
	"current":   {},
	"currently": {},
	"latest":    {},
	"today":     {},
}

func recallQueryAsksCurrentTruth(query string) bool {
	return recallQueryHasAnyToken(query, recallCurrentTruthIntentTokens)
}

func recallQueryHasAnyToken(query string, targets map[string]struct{}) bool {
	for _, token := range recallIntentTokens(query) {
		if _, ok := targets[token]; ok {
			return true
		}
	}
	return false
}

func recallIntentTokens(query string) []string {
	query = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return ' '
	}, query)
	return strings.Fields(query)
}

func recallHasSupersededEvidence(candidates []ScoredRecallCandidate) bool {
	for _, candidate := range candidates {
		if recallTemporalState(candidate.Candidate.Provenance) == recallTemporalEvidenceSuperseded {
			return true
		}
	}
	return false
}

func recallSpeakerAdjustment(candidate ScoredRecallCandidate, query string) float64 {
	query = textutil.LowerTrimmed(query)
	if query == "" {
		return 0
	}
	speaker := recallCandidateSpeaker(candidate.Candidate)
	if speaker == "" {
		return 0
	}
	if recallQueryMatchesSpeaker(query, speaker) {
		return recallSpeakerMatchBonus
	}
	return 0
}

func recallQueryMatchesSpeaker(query, speaker string) bool {
	targets := recallQuerySpeakerTargets(query)
	if len(targets) > 0 {
		return recallAnySpeakerTargetMatchesSpeaker(targets, speaker)
	}
	return recallQueryMentionsSpeaker(query, speaker)
}

func recallAnySpeakerTargetMatchesSpeaker(targets []string, speaker string) bool {
	for _, target := range targets {
		if recallSpeakerTargetMatchesSpeaker(target, speaker) {
			return true
		}
	}
	return false
}

func recallSpeakerTargetMatchesSpeaker(target, speaker string) bool {
	targetTokens := recallQueryTokens(target)
	speakerTokens := recallQueryTokens(speaker)
	if len(targetTokens) == 0 || len(speakerTokens) == 0 || len(targetTokens) > len(speakerTokens) {
		return false
	}
	for i := range targetTokens {
		if targetTokens[i] != speakerTokens[i] {
			return false
		}
	}
	return true
}

func recallQueryMentionsSpeaker(query, speaker string) bool {
	queryTokens := recallQueryTokens(query)
	speakerTokens := recallQueryTokens(speaker)
	if len(queryTokens) == 0 || len(speakerTokens) == 0 || len(speakerTokens) > len(queryTokens) {
		return false
	}
	for i := 0; i+len(speakerTokens) <= len(queryTokens); i++ {
		matched := true
		for j := range speakerTokens {
			if queryTokens[i+j] != speakerTokens[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func recallCandidateSpeaker(candidate RecallCandidate) string {
	for _, evidence := range candidate.Provenance {
		if evidence.Kind != "speaker" {
			continue
		}
		if speaker := recallSpeakerEvidenceIdentity(evidence); speaker != "" {
			return speaker
		}
	}
	return textutil.LowerTrimmed(candidate.AgentID)
}

func recallSpeakerEvidenceIdentity(evidence EvidenceItem) string {
	if speaker := recallSpeakerIdentityFromNote(evidence.Note); speaker != "" {
		return speaker
	}
	return textutil.LowerTrimmed(evidence.Source)
}

func recallSpeakerIdentityFromNote(note string) string {
	note = textutil.LowerTrimmed(note)
	if !strings.HasPrefix(note, "speaker=") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(note, "speaker="))
}

func recallQuerySpeakerTargets(query string) []string {
	tokens := recallQueryTokens(query)
	for i, token := range tokens {
		switch token {
		case "did":
			if target, ok := recallSpeakerTargetBetween(tokens, i+1, "say"); ok {
				return []string{target}
			}
		case "has":
			if target, ok := recallSpeakerTargetBetween(tokens, i+1, "said"); ok {
				return []string{target}
			}
		}
	}
	return nil
}

func recallSpeakerTargetBetween(tokens []string, start int, endToken string) (string, bool) {
	if start >= len(tokens) {
		return "", false
	}
	for end := start; end < len(tokens); end++ {
		if tokens[end] != endToken {
			continue
		}
		if end == start {
			return "", false
		}
		return strings.Join(tokens[start:end], " "), true
	}
	return "", false
}

func recallQueryTokens(query string) []string {
	query = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
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
			if recallGraphEvidenceLinksSelectedMemory(evidence, item.Candidate.MemoryID) {
				return recallGraphCoverageBonus
			}
		}
	}
	return 0
}

func recallGraphEvidenceLinksSelectedMemory(evidence EvidenceItem, selectedMemoryID string) bool {
	selectedMemoryID = strings.TrimSpace(selectedMemoryID)
	if selectedMemoryID == "" {
		return false
	}
	if strings.TrimSpace(evidence.Source) == selectedMemoryID {
		return true
	}
	return strings.HasPrefix(strings.TrimSpace(evidence.Note), selectedMemoryID+" -> ")
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
	switch recallDiversityKey(key) {
	case "memory_id":
		return candidate.MemoryID
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

func recallDiversityKey(key string) string {
	return textutil.LowerTrimmed(key)
}

func estimateRecallTokens(content string) int {
	n := textutil.WordCount(content)
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
	return hashutil.JSONSHA256Hex(view)
}

type recallTraceReplayCandidateView struct {
	Candidate RecallCandidate `json:"candidate"`
	Score     RecallScore     `json:"score"`
}

type recallTraceReplayRejectedView struct {
	Candidate   RecallCandidate `json:"candidate"`
	Score       RecallScore     `json:"score"`
	Reason      string          `json:"reason"`
	WhyRejected []string        `json:"why_rejected,omitempty"`
}

func recallTraceReplayFingerprint(trace RecallTrace) string {
	return hashutil.JSONSHA256Hex(recallTraceReplayFingerprintView(trace))
}

func recallTraceReplayFingerprintView(trace RecallTrace) any {
	view := struct {
		Query           RecallQuery                      `json:"query"`
		Candidates      []recallTraceReplayCandidateView `json:"candidates"`
		Selected        []recallTraceReplayCandidateView `json:"selected"`
		Rejected        []recallTraceReplayRejectedView  `json:"rejected"`
		Warnings        []RecallWarning                  `json:"warnings"`
		ScoringConfig   RecallScoringConfig              `json:"scoring_config"`
		PipelineVersion string                           `json:"pipeline_version"`
	}{
		Query:           trace.Query,
		Candidates:      []recallTraceReplayCandidateView{},
		Selected:        []recallTraceReplayCandidateView{},
		Rejected:        []recallTraceReplayRejectedView{},
		Warnings:        normalizeRecallTraceReplayWarnings(trace.Warnings),
		ScoringConfig:   trace.ScoringConfig,
		PipelineVersion: trace.PipelineVersion,
	}
	for _, item := range trace.Candidates {
		view.Candidates = append(view.Candidates, recallTraceReplayCandidateView{Candidate: item.Candidate, Score: item.Score})
	}
	for _, item := range trace.Selected {
		view.Selected = append(view.Selected, recallTraceReplayCandidateView{Candidate: item.Candidate, Score: item.Score})
	}
	for _, item := range trace.Rejected {
		view.Rejected = append(view.Rejected, recallTraceReplayRejectedView{Candidate: item.Candidate, Score: item.Score, Reason: item.Reason, WhyRejected: item.WhyRejected})
	}
	return view
}

func normalizeRecallTraceReplayWarnings(warnings []RecallWarning) []RecallWarning {
	if warnings == nil {
		return []RecallWarning{}
	}
	return warnings
}

func clampRecall(value float64) float64 {
	return recallscore.Clamp(value)
}

func roundRecallFloat(value float64) float64 {
	return recallscore.Round(value)
}
