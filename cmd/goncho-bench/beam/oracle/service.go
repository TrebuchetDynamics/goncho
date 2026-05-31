package oracle

import (
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/artifactcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/casecontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/serviceartifact"
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

type beamServiceSummaryFile = serviceartifact.SummaryFile

type beamServiceSummaryMetadata = serviceartifact.SummaryMetadata

type beamAbilityStats = serviceartifact.AbilityStats

type beamServicePairedOutcome = serviceartifact.PairedOutcome

type beamServiceFailureAuditRow = serviceartifact.FailureAuditRow

type beamServiceResultsFile = serviceartifact.ResultsFile

type beamServiceResultsMetadata = serviceartifact.ResultsMetadata

type beamServiceConversationResults = serviceartifact.ConversationResults

type beamServiceQuestionResult = serviceartifact.QuestionResult

type beamServiceRecallProvenance = artifactcontract.RecallProvenance

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
	return serviceartifact.BuildResults(report, serviceartifact.ResultsOptions{
		ConfigID:    configID,
		RunStarted:  runStartedAt,
		JudgeModel:  beamServiceJudgeModel(judgments),
		PureRecall:  judgments == nil,
		Diagnostics: beamServiceResultsDiagnostics(report, conversionDiagnostics, leakageChecks, judgments),
		Judgment: func(c goncho.RecallBenchmarkCaseReport) serviceartifact.CaseJudgment {
			judgment, ok := judgments.Find(c)
			return serviceartifact.CaseJudgment{
				Has:          ok,
				Score:        judgment.Score,
				AIAnswer:     judgment.AIAnswer,
				Nuggets:      judgment.Nuggets,
				Assessment:   judgment.Assessment,
				AnswerTimeMS: judgment.AnswerTimeMS,
				JudgeTimeMS:  judgment.JudgeTimeMS,
			}
		},
	})
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
		diagnostics["judgments"] = judgments.Diagnostics(report)
	}
	return diagnostics
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
	return serviceartifact.BuildSummary(report, serviceartifact.SummaryOptions{
		ConfigID:    configID,
		RunStarted:  runStartedAt,
		JudgeModel:  beamServiceJudgeModel(judgments),
		Description: beamServiceSummaryDescription(judgments),
		PureRecall:  judgments == nil,
		Score: func(c goncho.RecallBenchmarkCaseReport) float64 {
			return beamServiceArtifactScore(c, judgments)
		},
	})
}

func appendBeamServicePairedOutcomes(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, judgments *beamServiceJudgmentSet) error {
	return shared.AppendJSONLFileWithParents(path, "goncho-bench: create BEAM service paired-outcomes dir", "goncho-bench: open BEAM service paired outcomes", "goncho-bench: write BEAM service paired outcome", buildBeamServicePairedOutcomes(report, configID, runStartedAt, judgments))
}

func buildBeamServicePairedOutcomes(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, judgments *beamServiceJudgmentSet) []beamServicePairedOutcome {
	return serviceartifact.BuildPairedOutcomes(report, configID, runStartedAt, func(c goncho.RecallBenchmarkCaseReport) float64 {
		return beamServiceArtifactScore(c, judgments)
	})
}

func writeBeamServiceFailureAudit(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) error {
	return shared.WriteJSONLFileWithParents(path, "goncho-bench: create BEAM service failure audit dir", "goncho-bench: create BEAM service failure audit", "goncho-bench: write BEAM service failure audit row", buildBeamServiceFailureAuditRows(report, configID, runStartedAt))
}

func buildBeamServiceFailureAuditRows(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) []beamServiceFailureAuditRow {
	return serviceartifact.BuildFailureAuditRows(report, configID, runStartedAt)
}

func beamServiceArtifactScore(c goncho.RecallBenchmarkCaseReport, judgments *beamServiceJudgmentSet) float64 {
	if row, ok := judgments.Find(c); ok {
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
