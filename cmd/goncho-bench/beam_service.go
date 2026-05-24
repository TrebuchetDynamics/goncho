package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Score          float64 `json:"score"`
	Correct        bool    `json:"correct"`
}

func writeBeamServiceComparisonArtifacts(report goncho.RecallBenchmarkReport, cfg config, runStartedAt time.Time) error {
	configID := normalizeBeamServiceConfigID(cfg.BeamServiceConfigID)
	if path := strings.TrimSpace(cfg.BeamServiceSummaryOut); path != "" {
		if err := writeBeamServiceSummary(path, report, configID, runStartedAt); err != nil {
			return err
		}
	}
	if path := strings.TrimSpace(cfg.BeamServicePairedOut); path != "" {
		if err := appendBeamServicePairedOutcomes(path, report, configID, runStartedAt); err != nil {
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

func writeBeamServiceSummary(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) error {
	summary := buildBeamServiceSummary(report, configID, runStartedAt)
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

func buildBeamServiceSummary(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) beamServiceSummaryFile {
	abilityTotals := map[string]float64{}
	abilityCounts := map[string]int{}
	var overallTotal float64
	for _, c := range report.Cases {
		ability := strings.ToUpper(strings.TrimSpace(c.Ability))
		if ability == "" {
			continue
		}
		score := beamServiceCaseScore(c)
		abilityTotals[ability] += score
		abilityCounts[ability]++
		overallTotal += score
	}
	byAbility := map[string]beamAbilityStats{}
	for ability, count := range abilityCounts {
		byAbility[ability] = beamAbilityStats{AvgScore: roundMetric(abilityTotals[ability] / float64(count)), Count: count}
	}
	if len(report.Cases) > 0 {
		byAbility["OVERALL"] = beamAbilityStats{AvgScore: roundMetric(overallTotal / float64(len(report.Cases))), Count: len(report.Cases)}
	}
	return beamServiceSummaryFile{
		Date: runStartedAt.UTC().Format(beamServiceSummaryDateFormat),
		Metadata: beamServiceSummaryMetadata{
			Model:       beamServiceModelName,
			SampleSize:  report.CaseCount,
			JudgeModel:  beamServiceJudgeModelName,
			ConfigID:    configID,
			PureRecall:  true,
			Service:     report.Service,
			Corpus:      report.CorpusVersion,
			CaseCount:   report.CaseCount,
			Description: "deterministic service-backed BEAM-style MEMORIA recall oracle; no LLM answerer or judge",
		},
		AbilitySummary: map[string]map[string]beamAbilityStats{beamServiceScale: byAbility},
	}
}

func appendBeamServicePairedOutcomes(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create BEAM service paired-outcomes dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("goncho-bench: open BEAM service paired outcomes: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, outcome := range buildBeamServicePairedOutcomes(report, configID, runStartedAt) {
		if err := encoder.Encode(outcome); err != nil {
			return fmt.Errorf("goncho-bench: write BEAM service paired outcome: %w", err)
		}
	}
	return nil
}

func buildBeamServicePairedOutcomes(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) []beamServicePairedOutcome {
	out := make([]beamServicePairedOutcome, 0, len(report.Cases))
	started := runStartedAt.UTC().Format(beamServicePairedDateTimeFormat)
	for _, c := range report.Cases {
		score := beamServiceCaseScore(c)
		out = append(out, beamServicePairedOutcome{
			ConfigID:       configID,
			RunStartedAt:   started,
			Scale:          beamServiceScale,
			ConversationID: beamServiceConversationID,
			QID:            c.ID,
			Ability:        strings.ToUpper(strings.TrimSpace(c.Ability)),
			Score:          score,
			Correct:        score >= 0.5,
		})
	}
	return out
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
