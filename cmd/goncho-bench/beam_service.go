package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho"
)

const (
	beamServiceDefaultConfigID      = "goncho-service-beam-v1"
	beamServiceScale                = "100K"
	beamServiceConversationID       = "goncho-service-memoria-fixtures"
	beamServiceModelName            = "goncho-service-recall"
	beamServiceJudgeModelName       = "none"
	beamServiceSummaryDateFormat    = time.RFC3339
	beamServicePairedDateTimeFormat = time.RFC3339
)

type beamServiceSummaryFile struct {
	Date           string                                 `json:"date"`
	Metadata       beamServiceSummaryMetadata             `json:"metadata"`
	AbilitySummary map[string]map[string]beamAbilityStats `json:"ability_summary"`
}

type beamServiceSummaryMetadata struct {
	Model       string `json:"model"`
	SampleSize  int    `json:"sample_size"`
	JudgeModel  string `json:"judge_model"`
	ConfigID    string `json:"config_id"`
	PureRecall  bool   `json:"pure_recall"`
	Service     string `json:"service"`
	Corpus      string `json:"corpus_version"`
	CaseCount   int    `json:"case_count"`
	Description string `json:"description"`
}

type beamAbilityStats struct {
	AvgScore float64 `json:"avg_score"`
	Count    int     `json:"count"`
}

type beamServicePairedOutcome struct {
	ConfigID       string  `json:"config_id"`
	RunStartedAt   string  `json:"run_started_at"`
	Scale          string  `json:"scale"`
	ConversationID string  `json:"conversation_id"`
	QID            string  `json:"qid"`
	Ability        string  `json:"ability"`
	Question       string  `json:"question,omitempty"`
	Score          float64 `json:"score"`
	Correct        bool    `json:"correct"`
}

type beamServiceFailureAuditRow struct {
	ConfigID              string   `json:"config_id"`
	RunStartedAt          string   `json:"run_started_at"`
	Scale                 string   `json:"scale"`
	ConversationID        string   `json:"conversation_id"`
	QID                   string   `json:"qid"`
	Ability               string   `json:"ability"`
	Question              string   `json:"question"`
	Score                 float64  `json:"score"`
	FailureMode           string   `json:"failure_mode"`
	Rank                  int      `json:"rank"`
	RelevantIDs           []string `json:"relevant_ids"`
	RequiredEvidenceKinds []string `json:"required_evidence_kinds,omitempty"`
	ExpectedNoAnswer      bool     `json:"expected_no_answer,omitempty"`
	CandidateMemoryIDs    []string `json:"candidate_memory_ids"`
	SelectedMemoryIDs     []string `json:"selected_memory_ids"`
	RetrievedTop10        []string `json:"retrieved_top_10"`
	SelectedEvidenceKinds []string `json:"selected_evidence_kinds,omitempty"`
	TopEvidenceKinds      []string `json:"top_evidence_kinds,omitempty"`
	RecallAt5             float64  `json:"recall_at_5"`
	RecallAt10            float64  `json:"recall_at_10"`
	ContextSatisfied      bool     `json:"context_satisfied"`
	ProvenanceSatisfied   bool     `json:"provenance_satisfied"`
	TokenBudgetWithin     bool     `json:"token_budget_within"`
	WarningCodes          []string `json:"warning_codes,omitempty"`
}

type beamServiceResultsFile struct {
	Metadata beamServiceResultsMetadata       `json:"metadata"`
	Results  []beamServiceConversationResults `json:"results"`
}

type beamServiceResultsMetadata struct {
	Date               string                 `json:"date"`
	RunStartedAt       string                 `json:"run_started_at"`
	ConfigID           string                 `json:"config_id"`
	Model              string                 `json:"model"`
	JudgeModel         string                 `json:"judge_model"`
	TopK               int                    `json:"top_k"`
	SampleSize         int                    `json:"sample_size"`
	Scales             []string               `json:"scales"`
	TotalConversations int                    `json:"total_conversations"`
	PureRecall         bool                   `json:"pure_recall"`
	Config             map[string]any         `json:"config"`
	Diagnostics        map[string]interface{} `json:"diagnostics"`
}

type beamServiceConversationResults struct {
	ConversationID string                      `json:"conversation_id"`
	Scale          string                      `json:"scale"`
	NumQuestions   int                         `json:"num_questions"`
	NumEvaluated   int                         `json:"num_evaluated"`
	Results        []beamServiceQuestionResult `json:"results"`
}

type beamServiceQuestionResult struct {
	QID                  string                      `json:"qid"`
	Ability              string                      `json:"ability"`
	Question             string                      `json:"question,omitempty"`
	IdealAnswer          string                      `json:"ideal_answer,omitempty"`
	Rubric               []string                    `json:"rubric,omitempty"`
	RubricContextScore   float64                     `json:"rubric_context_score,omitempty"`
	RubricContextMatches []string                    `json:"rubric_context_matches,omitempty"`
	AIAnswer             string                      `json:"ai_answer"`
	RecallProvenance     beamServiceRecallProvenance `json:"recall_provenance"`
	Score                float64                     `json:"score"`
	Nuggets              []string                    `json:"nuggets"`
	Assessment           string                      `json:"assessment"`
	AnswerTimeMS         float64                     `json:"answer_time_ms"`
	JudgeTimeMS          float64                     `json:"judge_time_ms"`
}

type beamServiceRecallProvenance struct {
	Engine             string             `json:"engine"`
	KeptCount          int                `json:"kept_count"`
	VoiceSums          map[string]float64 `json:"voice_sums"`
	TopResultVoices    map[string]float64 `json:"top_result_voices"`
	TopResultTier      string             `json:"top_result_tier"`
	CandidateMemoryIDs []string           `json:"candidate_memory_ids,omitempty"`
	SelectedMemoryIDs  []string           `json:"selected_memory_ids,omitempty"`
}

func writeBeamServiceComparisonArtifacts(report goncho.RecallBenchmarkReport, cfg config, runStartedAt time.Time) error {
	configID := normalizeBeamServiceConfigID(cfg.BeamServiceConfigID)
	if path := strings.TrimSpace(cfg.BeamServiceResultsOut); path != "" {
		if err := writeBeamServiceResults(path, report, configID, runStartedAt, cfg.BeamConversionDiagnostics, cfg.BeamServiceLeakageChecks, cfg.BeamServiceJudgments); err != nil {
			return err
		}
	}
	if path := strings.TrimSpace(cfg.BeamServiceSummaryOut); path != "" {
		if err := writeBeamServiceSummary(path, report, configID, runStartedAt, cfg.BeamServiceJudgments); err != nil {
			return err
		}
	}
	if path := strings.TrimSpace(cfg.BeamServicePairedOut); path != "" {
		if err := appendBeamServicePairedOutcomes(path, report, configID, runStartedAt, cfg.BeamServiceJudgments); err != nil {
			return err
		}
	}
	if path := strings.TrimSpace(cfg.BeamServiceFailuresOut); path != "" {
		if err := writeBeamServiceFailureAudit(path, report, configID, runStartedAt); err != nil {
			return err
		}
	}
	if path := strings.TrimSpace(cfg.BeamServiceJudgeRequestsOut); path != "" {
		if err := writeBeamServiceJudgeRequests(path, report, configID, runStartedAt); err != nil {
			return err
		}
	}
	return nil
}

func normalizeBeamServiceConfigID(configID string) string {
	configID = strings.TrimSpace(configID)
	if configID == "" {
		return beamServiceDefaultConfigID
	}
	return configID
}

func writeBeamServiceResults(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, conversionDiagnostics *beamConversionDiagnostics, leakageChecks *beamServiceLeakageChecks, judgments *beamServiceJudgmentSet) error {
	results := buildBeamServiceResults(report, configID, runStartedAt, conversionDiagnostics, leakageChecks, judgments)
	raw, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("goncho-bench: encode BEAM service results: %w", err)
	}
	raw = append(raw, '\n')
	if path == "-" {
		if _, err := os.Stdout.Write(raw); err != nil {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create BEAM service results dir: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("goncho-bench: write BEAM service results: %w", err)
	}
	return nil
}

func buildBeamServiceResults(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, conversionDiagnostics *beamConversionDiagnostics, leakageChecks *beamServiceLeakageChecks, judgments *beamServiceJudgmentSet) beamServiceResultsFile {
	type conversationAccumulator struct {
		conversationID string
		scale          string
		results        []beamServiceQuestionResult
	}
	byConversation := map[string]*conversationAccumulator{}
	conversationOrder := []string{}
	scales := map[string]struct{}{}
	for _, c := range report.Cases {
		conversationID := beamServiceCaseConversationID(c)
		scale := beamServiceCaseScale(c)
		key := scale + "\x00" + conversationID
		acc := byConversation[key]
		if acc == nil {
			acc = &conversationAccumulator{conversationID: conversationID, scale: scale}
			byConversation[key] = acc
			conversationOrder = append(conversationOrder, key)
		}
		scales[scale] = struct{}{}
		judgment, hasJudgment := judgments.find(c)
		score := beamServiceCaseScore(c)
		aiAnswer := ""
		nuggets := []string{}
		assessment := beamServiceCaseAssessment(c, score)
		answerTimeMS := 0.0
		judgeTimeMS := 0.0
		if hasJudgment {
			score = roundMetric(judgment.Score)
			aiAnswer = strings.TrimSpace(judgment.AIAnswer)
			nuggets = append([]string(nil), judgment.Nuggets...)
			assessment = strings.TrimSpace(judgment.Assessment)
			answerTimeMS = judgment.AnswerTimeMS
			judgeTimeMS = judgment.JudgeTimeMS
		}
		acc.results = append(acc.results, beamServiceQuestionResult{
			QID:                  c.ID,
			Ability:              strings.ToUpper(strings.TrimSpace(c.Ability)),
			Question:             strings.TrimSpace(c.Question),
			IdealAnswer:          strings.TrimSpace(c.IdealAnswer),
			Rubric:               append([]string(nil), c.Rubric...),
			RubricContextScore:   c.RubricContextScore,
			RubricContextMatches: append([]string(nil), c.RubricContextMatches...),
			AIAnswer:             aiAnswer,
			RecallProvenance:     beamServiceCaseRecallProvenance(c),
			Score:                score,
			Nuggets:              nuggets,
			Assessment:           assessment,
			AnswerTimeMS:         answerTimeMS,
			JudgeTimeMS:          judgeTimeMS,
		})
	}
	conversationResults := make([]beamServiceConversationResults, 0, len(conversationOrder))
	for _, key := range conversationOrder {
		acc := byConversation[key]
		conversationResults = append(conversationResults, beamServiceConversationResults{
			ConversationID: acc.conversationID,
			Scale:          acc.scale,
			NumQuestions:   len(acc.results),
			NumEvaluated:   len(acc.results),
			Results:        acc.results,
		})
	}
	scaleList := make([]string, 0, len(scales))
	for scale := range scales {
		scaleList = append(scaleList, scale)
	}
	sort.Strings(scaleList)
	started := runStartedAt.UTC().Format(beamServicePairedDateTimeFormat)
	return beamServiceResultsFile{
		Metadata: beamServiceResultsMetadata{
			Date:               time.Now().UTC().Format(beamServiceSummaryDateFormat),
			RunStartedAt:       started,
			ConfigID:           configID,
			Model:              beamServiceModelName,
			JudgeModel:         beamServiceJudgeModel(judgments),
			TopK:               5,
			SampleSize:         len(conversationResults),
			Scales:             scaleList,
			TotalConversations: len(conversationResults),
			PureRecall:         judgments == nil,
			Config: map[string]any{
				"pure_recall":           judgments == nil,
				"external_judgments":    judgments != nil,
				"allow_harness_oracles": false,
				"full_context":          false,
				"use_cloud":             false,
			},
			Diagnostics: beamServiceResultsDiagnostics(report, conversionDiagnostics, leakageChecks, judgments),
		},
		Results: conversationResults,
	}
}

func beamServiceResultsDiagnostics(report goncho.RecallBenchmarkReport, conversionDiagnostics *beamConversionDiagnostics, leakageChecks *beamServiceLeakageChecks, judgments *beamServiceJudgmentSet) map[string]interface{} {
	diagnostics := map[string]interface{}{
		"recall": map[string]interface{}{
			"case_count":       report.CaseCount,
			"warning_count":    report.WarningCount,
			"recall_at_5":      report.RecallAt5,
			"recall_at_10":     report.RecallAt10,
			"context_hit_rate": report.ContextHitRate,
		},
	}
	if conversionDiagnostics != nil {
		diagnostics["conversion"] = *conversionDiagnostics
	}
	if leakageChecks != nil {
		diagnostics["leakage"] = *leakageChecks
	}
	if judgments != nil {
		diagnostics["judgments"] = judgments.diagnostics(report)
	}
	return diagnostics
}

func beamServiceCaseRecallProvenance(c goncho.RecallBenchmarkCaseReport) beamServiceRecallProvenance {
	return beamServiceRecallProvenance{
		Engine:             beamServiceModelName,
		KeptCount:          len(c.CandidateMemoryIDs),
		VoiceSums:          beamServiceVoiceMap(c.SelectedEvidenceKinds),
		TopResultVoices:    beamServiceVoiceMap(c.TopEvidenceKinds),
		TopResultTier:      beamServiceTopResultTier(c.TopEvidenceKinds),
		CandidateMemoryIDs: append([]string(nil), c.CandidateMemoryIDs...),
		SelectedMemoryIDs:  append([]string(nil), c.SelectedMemoryIDs...),
	}
}

func beamServiceVoiceMap(kinds []string) map[string]float64 {
	out := map[string]float64{}
	for _, kind := range kinds {
		kind = strings.ToLower(strings.TrimSpace(kind))
		if kind != "" {
			out[kind]++
		}
	}
	return out
}

func beamServiceTopResultTier(kinds []string) string {
	for _, kind := range kinds {
		switch strings.ToLower(strings.TrimSpace(kind)) {
		case "graph", "fact":
			return "structured"
		}
	}
	if len(kinds) > 0 {
		return "episodic"
	}
	return "unknown"
}

func beamServiceCaseAssessment(c goncho.RecallBenchmarkCaseReport, score float64) string {
	if score >= 1 {
		return "pure-recall context selected the required memory and provenance gates passed"
	}
	if len(c.WarningCodes) > 0 {
		return "pure-recall context did not satisfy benchmark gates; see warning_codes in the service report"
	}
	return "pure-recall context did not satisfy benchmark gates"
}

func writeBeamServiceSummary(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, judgments *beamServiceJudgmentSet) error {
	summary := buildBeamServiceSummary(report, configID, runStartedAt, judgments)
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("goncho-bench: encode BEAM service summary: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create BEAM service summary dir: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("goncho-bench: write BEAM service summary: %w", err)
	}
	return nil
}

func buildBeamServiceSummary(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, judgments *beamServiceJudgmentSet) beamServiceSummaryFile {
	type scaleStats struct {
		abilityTotals map[string]float64
		abilityCounts map[string]int
		overallTotal  float64
		overallCount  int
	}
	stats := map[string]*scaleStats{}
	for _, c := range report.Cases {
		ability := strings.ToUpper(strings.TrimSpace(c.Ability))
		if ability == "" {
			continue
		}
		scale := beamServiceCaseScale(c)
		acc := stats[scale]
		if acc == nil {
			acc = &scaleStats{abilityTotals: map[string]float64{}, abilityCounts: map[string]int{}}
			stats[scale] = acc
		}
		score := beamServiceArtifactScore(c, judgments)
		acc.abilityTotals[ability] += score
		acc.abilityCounts[ability]++
		acc.overallTotal += score
		acc.overallCount++
	}
	abilitySummary := map[string]map[string]beamAbilityStats{}
	for scale, acc := range stats {
		byAbility := map[string]beamAbilityStats{}
		for ability, count := range acc.abilityCounts {
			byAbility[ability] = beamAbilityStats{AvgScore: roundMetric(acc.abilityTotals[ability] / float64(count)), Count: count}
		}
		if acc.overallCount > 0 {
			byAbility["OVERALL"] = beamAbilityStats{AvgScore: roundMetric(acc.overallTotal / float64(acc.overallCount)), Count: acc.overallCount}
		}
		abilitySummary[scale] = byAbility
	}
	return beamServiceSummaryFile{
		Date: runStartedAt.UTC().Format(beamServiceSummaryDateFormat),
		Metadata: beamServiceSummaryMetadata{
			Model:       beamServiceModelName,
			SampleSize:  report.CaseCount,
			JudgeModel:  beamServiceJudgeModel(judgments),
			ConfigID:    configID,
			PureRecall:  judgments == nil,
			Service:     report.Service,
			Corpus:      report.CorpusVersion,
			CaseCount:   report.CaseCount,
			Description: beamServiceSummaryDescription(judgments),
		},
		AbilitySummary: abilitySummary,
	}
}

func appendBeamServicePairedOutcomes(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, judgments *beamServiceJudgmentSet) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create BEAM service paired-outcomes dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("goncho-bench: open BEAM service paired outcomes: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, outcome := range buildBeamServicePairedOutcomes(report, configID, runStartedAt, judgments) {
		if err := encoder.Encode(outcome); err != nil {
			return fmt.Errorf("goncho-bench: write BEAM service paired outcome: %w", err)
		}
	}
	return nil
}

func buildBeamServicePairedOutcomes(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, judgments *beamServiceJudgmentSet) []beamServicePairedOutcome {
	out := make([]beamServicePairedOutcome, 0, len(report.Cases))
	started := runStartedAt.UTC().Format(beamServicePairedDateTimeFormat)
	for _, c := range report.Cases {
		score := beamServiceArtifactScore(c, judgments)
		out = append(out, beamServicePairedOutcome{
			ConfigID:       configID,
			RunStartedAt:   started,
			Scale:          beamServiceCaseScale(c),
			ConversationID: beamServiceCaseConversationID(c),
			QID:            c.ID,
			Ability:        strings.ToUpper(strings.TrimSpace(c.Ability)),
			Question:       strings.TrimSpace(c.Question),
			Score:          score,
			Correct:        score >= 0.5,
		})
	}
	return out
}

func writeBeamServiceFailureAudit(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create BEAM service failure audit dir: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("goncho-bench: create BEAM service failure audit: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, row := range buildBeamServiceFailureAuditRows(report, configID, runStartedAt) {
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("goncho-bench: write BEAM service failure audit row: %w", err)
		}
	}
	return nil
}

func buildBeamServiceFailureAuditRows(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) []beamServiceFailureAuditRow {
	out := []beamServiceFailureAuditRow{}
	started := runStartedAt.UTC().Format(beamServicePairedDateTimeFormat)
	for _, c := range report.Cases {
		score := beamServiceCaseScore(c)
		if score >= 1 {
			continue
		}
		out = append(out, beamServiceFailureAuditRow{
			ConfigID:              configID,
			RunStartedAt:          started,
			Scale:                 beamServiceCaseScale(c),
			ConversationID:        beamServiceCaseConversationID(c),
			QID:                   c.ID,
			Ability:               strings.ToUpper(strings.TrimSpace(c.Ability)),
			Question:              strings.TrimSpace(c.Question),
			Score:                 score,
			FailureMode:           beamServiceFailureMode(c, score),
			Rank:                  beamServiceFirstRelevantRank(c.CandidateMemoryIDs, c.RelevantIDs),
			RelevantIDs:           append([]string(nil), c.RelevantIDs...),
			RequiredEvidenceKinds: append([]string(nil), c.RequiredEvidenceKinds...),
			ExpectedNoAnswer:      c.ExpectedNoAnswer,
			CandidateMemoryIDs:    append([]string(nil), c.CandidateMemoryIDs...),
			SelectedMemoryIDs:     append([]string(nil), c.SelectedMemoryIDs...),
			RetrievedTop10:        topN(c.CandidateMemoryIDs, 10),
			SelectedEvidenceKinds: append([]string(nil), c.SelectedEvidenceKinds...),
			TopEvidenceKinds:      append([]string(nil), c.TopEvidenceKinds...),
			RecallAt5:             c.RecallAt5,
			RecallAt10:            c.RecallAt10,
			ContextSatisfied:      c.ContextSatisfied,
			ProvenanceSatisfied:   c.ProvenanceSatisfied,
			TokenBudgetWithin:     c.TokenBudgetWithin,
			WarningCodes:          append([]string(nil), c.WarningCodes...),
		})
	}
	return out
}

func beamServiceFirstRelevantRank(candidateIDs, relevantIDs []string) int {
	relevant := map[string]struct{}{}
	for _, id := range relevantIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			relevant[id] = struct{}{}
		}
	}
	for i, id := range candidateIDs {
		if _, ok := relevant[strings.TrimSpace(id)]; ok {
			return i + 1
		}
	}
	return 0
}

func beamServiceFailureMode(c goncho.RecallBenchmarkCaseReport, score float64) string {
	if len(c.RelevantIDs) == 0 && !c.ExpectedNoAnswer {
		return "unscorable_missing_relevant_ids"
	}
	if c.ExpectedNoAnswer && len(c.SelectedMemoryIDs) > 0 {
		return "abstention_failed"
	}
	rank := beamServiceFirstRelevantRank(c.CandidateMemoryIDs, c.RelevantIDs)
	if c.RecallAt5 <= 0 {
		if rank == 0 {
			return "missing_candidate"
		}
		return "rank_too_low"
	}
	if !c.ContextSatisfied {
		return "context_unsatisfied"
	}
	if len(c.RequiredEvidenceKinds) > 0 && !c.ProvenanceSatisfied {
		return "provenance_unsatisfied"
	}
	if !c.TokenBudgetWithin {
		return "token_budget_exceeded"
	}
	if len(c.WarningCodes) > 0 {
		return "recall_warning"
	}
	if score < 1 {
		return "partial_recall"
	}
	return "unknown"
}

func beamServiceCaseScale(c goncho.RecallBenchmarkCaseReport) string {
	scale := strings.TrimSpace(c.Scale)
	if scale == "" {
		return beamServiceScale
	}
	return scale
}

func beamServiceCaseConversationID(c goncho.RecallBenchmarkCaseReport) string {
	conversationID := strings.TrimSpace(c.ConversationID)
	if conversationID == "" {
		return beamServiceConversationID
	}
	return conversationID
}

func beamServiceArtifactScore(c goncho.RecallBenchmarkCaseReport, judgments *beamServiceJudgmentSet) float64 {
	if row, ok := judgments.find(c); ok {
		return roundMetric(row.Score)
	}
	return beamServiceCaseScore(c)
}

func beamServiceJudgeModel(judgments *beamServiceJudgmentSet) string {
	if judgments != nil {
		return "external-beam-judge"
	}
	return beamServiceJudgeModelName
}

func beamServiceSummaryDescription(judgments *beamServiceJudgmentSet) string {
	if judgments != nil {
		return "service-backed BEAM recall context with imported official-style answer/judge scores"
	}
	return "deterministic service-backed BEAM-style MEMORIA recall oracle; no LLM answerer or judge"
}

func beamServiceCaseScore(c goncho.RecallBenchmarkCaseReport) float64 {
	if c.RecallAt5 <= 0 || !c.ContextSatisfied || !c.TokenBudgetWithin {
		return 0
	}
	if len(c.RequiredEvidenceKinds) > 0 && !c.ProvenanceSatisfied {
		return 0
	}
	return roundMetric(c.RecallAt5)
}
