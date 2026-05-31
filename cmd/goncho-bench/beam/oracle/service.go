package oracle

import (
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/casecontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	"github.com/TrebuchetDynamics/goncho/service"
)

const (
	beamServiceDefaultConfigID   = casecontract.DefaultConfigID
	beamServiceScale             = casecontract.DefaultScale
	beamServiceConversationID    = casecontract.DefaultConversationID
	beamServiceModelName         = casecontract.ModelName
	beamServiceJudgeModelName    = casecontract.JudgeModelName
	beamServiceSummaryDateFormat = time.RFC3339
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

type beamServicePairedOutcome = shared.PairedOutcome

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

func writeBeamServiceComparisonArtifacts(report goncho.RecallBenchmarkReport, cfg ServiceConfig, runStartedAt time.Time) error {
	configID := normalizeBeamServiceConfigID(cfg.ServiceConfigID)
	if path := strings.TrimSpace(cfg.ServiceResultsOut); path != "" {
		if err := writeBeamServiceResults(path, report, configID, runStartedAt, cfg.conversionDiagnostics, cfg.leakageChecks, cfg.judgments); err != nil {
			return err
		}
	}
	if path := strings.TrimSpace(cfg.ServiceSummaryOut); path != "" {
		if err := writeBeamServiceSummary(path, report, configID, runStartedAt, cfg.judgments); err != nil {
			return err
		}
	}
	if path := strings.TrimSpace(cfg.ServicePairedOut); path != "" {
		if err := appendBeamServicePairedOutcomes(path, report, configID, runStartedAt, cfg.judgments); err != nil {
			return err
		}
	}
	if path := strings.TrimSpace(cfg.ServiceFailuresOut); path != "" {
		if err := writeBeamServiceFailureAudit(path, report, configID, runStartedAt); err != nil {
			return err
		}
	}
	if path := strings.TrimSpace(cfg.ServiceJudgeRequestsOut); path != "" {
		if err := writeBeamServiceJudgeRequests(path, report, configID, runStartedAt); err != nil {
			return err
		}
	}
	return nil
}

func normalizeBeamServiceConfigID(configID string) string {
	return casecontract.NormalizeConfigID(configID)
}

func writeBeamServiceResults(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, conversionDiagnostics *beamConversionDiagnostics, leakageChecks *beamServiceLeakageChecks, judgments *beamServiceJudgmentSet) error {
	results := buildBeamServiceResults(report, configID, runStartedAt, conversionDiagnostics, leakageChecks, judgments)
	raw, err := shared.MarshalIndentedJSON(results)
	if err != nil {
		return fmt.Errorf("goncho-bench: encode BEAM service results: %w", err)
	}
	return shared.WriteBytesArtifact(path, raw, "goncho-bench: create BEAM service results dir", "goncho-bench: write BEAM service results")
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
			score = shared.RoundMetric(judgment.Score)
			aiAnswer = strings.TrimSpace(judgment.AIAnswer)
			nuggets = append([]string(nil), judgment.Nuggets...)
			assessment = strings.TrimSpace(judgment.Assessment)
			answerTimeMS = judgment.AnswerTimeMS
			judgeTimeMS = judgment.JudgeTimeMS
		}
		acc.results = append(acc.results, beamServiceQuestionResult{
			QID:                  c.ID,
			Ability:              shared.NormalizeAbility(c.Ability),
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
	scaleList := shared.SortedStringMapKeys(scales)
	started := shared.FormatArtifactTimestamp(runStartedAt)
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
	return casecontract.VoiceMap(kinds)
}

func beamServiceTopResultTier(kinds []string) string {
	return casecontract.TopResultTier(kinds)
}

func beamServiceCaseAssessment(c goncho.RecallBenchmarkCaseReport, score float64) string {
	return casecontract.Assessment(c, score)
}

func writeBeamServiceSummary(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, judgments *beamServiceJudgmentSet) error {
	summary := buildBeamServiceSummary(report, configID, runStartedAt, judgments)
	raw, err := shared.MarshalIndentedJSON(summary)
	if err != nil {
		return fmt.Errorf("goncho-bench: encode BEAM service summary: %w", err)
	}
	return shared.WriteFileWithParents(path, raw, "goncho-bench: create BEAM service summary dir", "goncho-bench: write BEAM service summary")
}

func buildBeamServiceSummary(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, judgments *beamServiceJudgmentSet) beamServiceSummaryFile {
	type scaleStats struct {
		abilityTallies map[string]*shared.ScoreTally
		overallTally   shared.ScoreTally
	}
	stats := map[string]*scaleStats{}
	for _, c := range report.Cases {
		ability := shared.NormalizeAbility(c.Ability)
		if ability == "" {
			continue
		}
		scale := beamServiceCaseScale(c)
		acc := stats[scale]
		if acc == nil {
			acc = &scaleStats{abilityTallies: map[string]*shared.ScoreTally{}}
			stats[scale] = acc
		}
		tally := acc.abilityTallies[ability]
		if tally == nil {
			tally = &shared.ScoreTally{}
			acc.abilityTallies[ability] = tally
		}
		score := beamServiceArtifactScore(c, judgments)
		tally.Add(score)
		acc.overallTally.Add(score)
	}
	abilitySummary := map[string]map[string]beamAbilityStats{}
	for scale, acc := range stats {
		byAbility := map[string]beamAbilityStats{}
		for ability, tally := range acc.abilityTallies {
			byAbility[ability] = beamAbilityStats{AvgScore: tally.Average(), Count: tally.Count()}
		}
		if acc.overallTally.Count() > 0 {
			byAbility["OVERALL"] = beamAbilityStats{AvgScore: acc.overallTally.Average(), Count: acc.overallTally.Count()}
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
	return shared.AppendJSONLFileWithParents(path, "goncho-bench: create BEAM service paired-outcomes dir", "goncho-bench: open BEAM service paired outcomes", "goncho-bench: write BEAM service paired outcome", buildBeamServicePairedOutcomes(report, configID, runStartedAt, judgments))
}

func buildBeamServicePairedOutcomes(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, judgments *beamServiceJudgmentSet) []beamServicePairedOutcome {
	out := make([]beamServicePairedOutcome, 0, len(report.Cases))
	started := shared.FormatArtifactTimestamp(runStartedAt)
	for _, c := range report.Cases {
		score := beamServiceArtifactScore(c, judgments)
		out = append(out, beamServicePairedOutcome{
			ConfigID:       configID,
			RunStartedAt:   started,
			Scale:          beamServiceCaseScale(c),
			ConversationID: beamServiceCaseConversationID(c),
			QID:            c.ID,
			Ability:        shared.NormalizeAbility(c.Ability),
			Question:       strings.TrimSpace(c.Question),
			Score:          score,
			Correct:        shared.PairedOutcomeCorrect(score),
		})
	}
	return out
}

func writeBeamServiceFailureAudit(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) error {
	return shared.WriteJSONLFileWithParents(path, "goncho-bench: create BEAM service failure audit dir", "goncho-bench: create BEAM service failure audit", "goncho-bench: write BEAM service failure audit row", buildBeamServiceFailureAuditRows(report, configID, runStartedAt))
}

func buildBeamServiceFailureAuditRows(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) []beamServiceFailureAuditRow {
	out := []beamServiceFailureAuditRow{}
	started := shared.FormatArtifactTimestamp(runStartedAt)
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
			Ability:               shared.NormalizeAbility(c.Ability),
			Question:              strings.TrimSpace(c.Question),
			Score:                 score,
			FailureMode:           beamServiceFailureMode(c, score),
			Rank:                  beamServiceFirstRelevantRank(c.CandidateMemoryIDs, c.RelevantIDs),
			RelevantIDs:           append([]string(nil), c.RelevantIDs...),
			RequiredEvidenceKinds: append([]string(nil), c.RequiredEvidenceKinds...),
			ExpectedNoAnswer:      c.ExpectedNoAnswer,
			CandidateMemoryIDs:    append([]string(nil), c.CandidateMemoryIDs...),
			SelectedMemoryIDs:     append([]string(nil), c.SelectedMemoryIDs...),
			RetrievedTop10:        shared.TopN(c.CandidateMemoryIDs, 10),
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
	return casecontract.FirstRelevantRank(candidateIDs, relevantIDs)
}

func beamServiceFailureMode(c goncho.RecallBenchmarkCaseReport, score float64) string {
	return casecontract.FailureMode(c, score)
}

func beamServiceCaseScale(c goncho.RecallBenchmarkCaseReport) string {
	return casecontract.Scale(c)
}

func beamServiceCaseConversationID(c goncho.RecallBenchmarkCaseReport) string {
	return casecontract.ConversationID(c)
}

func beamServiceArtifactScore(c goncho.RecallBenchmarkCaseReport, judgments *beamServiceJudgmentSet) float64 {
	if row, ok := judgments.find(c); ok {
		return shared.RoundMetric(row.Score)
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
	return casecontract.Score(c)
}
