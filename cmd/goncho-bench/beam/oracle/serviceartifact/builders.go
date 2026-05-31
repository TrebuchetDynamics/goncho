package serviceartifact

import (
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/artifactcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/casecontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

// CaseScoreFunc supplies the artifact score used by rows that can be pure-recall
// scored or external-judgment scored.
type CaseScoreFunc func(goncho.RecallBenchmarkCaseReport) float64

// BuildPairedOutcomes projects recall cases into the shared paired-outcome contract.
func BuildPairedOutcomes(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, score CaseScoreFunc) []PairedOutcome {
	out := make([]PairedOutcome, 0, len(report.Cases))
	started := shared.FormatArtifactTimestamp(runStartedAt)
	for _, c := range report.Cases {
		fields := artifactcontract.BuildCaseFields(c)
		caseScore := score(c)
		out = append(out, PairedOutcome{
			ConfigID:       configID,
			RunStartedAt:   started,
			Scale:          fields.Scale,
			ConversationID: fields.ConversationID,
			QID:            fields.QID,
			Ability:        fields.Ability,
			Question:       fields.Question,
			Score:          caseScore,
			Correct:        shared.PairedOutcomeCorrect(caseScore),
		})
	}
	return out
}

// BuildFailureAuditRows projects failed recall cases into the stable failure-audit contract.
func BuildFailureAuditRows(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) []FailureAuditRow {
	out := []FailureAuditRow{}
	started := shared.FormatArtifactTimestamp(runStartedAt)
	for _, c := range report.Cases {
		score := casecontract.Score(c)
		if score >= 1 {
			continue
		}
		fields := artifactcontract.BuildCaseFields(c)
		out = append(out, FailureAuditRow{
			ConfigID:              configID,
			RunStartedAt:          started,
			Scale:                 fields.Scale,
			ConversationID:        fields.ConversationID,
			QID:                   fields.QID,
			Ability:               fields.Ability,
			Question:              fields.Question,
			Score:                 score,
			FailureMode:           casecontract.FailureMode(c, score),
			Rank:                  casecontract.FirstRelevantRank(c.CandidateMemoryIDs, c.RelevantIDs),
			RelevantIDs:           append([]string(nil), c.RelevantIDs...),
			RequiredEvidenceKinds: append([]string(nil), c.RequiredEvidenceKinds...),
			ExpectedNoAnswer:      c.ExpectedNoAnswer,
			CandidateMemoryIDs:    fields.CandidateMemoryIDs,
			SelectedMemoryIDs:     fields.SelectedMemoryIDs,
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
