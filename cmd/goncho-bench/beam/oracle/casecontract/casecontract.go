package casecontract

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

const (
	DefaultConfigID       = "goncho-service-beam-v1"
	DefaultScale          = "100K"
	DefaultConversationID = "goncho-service-memoria-fixtures"
	ModelName             = "goncho-service-recall"
	JudgeModelName        = "none"
)

func NormalizeConfigID(configID string) string {
	return shared.FirstNonEmptyTrimmed(configID, DefaultConfigID)
}

func Scale(c goncho.RecallBenchmarkCaseReport) string {
	return shared.FirstNonEmptyTrimmed(c.Scale, DefaultScale)
}

func ConversationID(c goncho.RecallBenchmarkCaseReport) string {
	return shared.FirstNonEmptyTrimmed(c.ConversationID, DefaultConversationID)
}

func Score(c goncho.RecallBenchmarkCaseReport) float64 {
	if c.RecallAt5 <= 0 || !c.ContextSatisfied || !c.TokenBudgetWithin {
		return 0
	}
	if len(c.RequiredEvidenceKinds) > 0 && !c.ProvenanceSatisfied {
		return 0
	}
	return shared.RoundMetric(c.RecallAt5)
}

func FirstRelevantRank(candidateIDs, relevantIDs []string) int {
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

func FailureMode(c goncho.RecallBenchmarkCaseReport, score float64) string {
	if len(c.RelevantIDs) == 0 && !c.ExpectedNoAnswer {
		return "unscorable_missing_relevant_ids"
	}
	if c.ExpectedNoAnswer && len(c.SelectedMemoryIDs) > 0 {
		return "abstention_failed"
	}
	rank := FirstRelevantRank(c.CandidateMemoryIDs, c.RelevantIDs)
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

func Assessment(c goncho.RecallBenchmarkCaseReport, score float64) string {
	if score >= 1 {
		return "pure-recall context selected the required memory and provenance gates passed"
	}
	if len(c.WarningCodes) > 0 {
		return "pure-recall context did not satisfy benchmark gates; see warning_codes in the service report"
	}
	return "pure-recall context did not satisfy benchmark gates"
}

func VoiceMap(kinds []string) map[string]float64 {
	out := map[string]float64{}
	for _, kind := range kinds {
		kind = shared.NormalizeEvidenceKind(kind)
		if kind != "" {
			out[kind]++
		}
	}
	return out
}

func TopResultTier(kinds []string) string {
	for _, kind := range kinds {
		switch shared.NormalizeEvidenceKind(kind) {
		case "graph", "fact":
			return "structured"
		}
	}
	if len(kinds) > 0 {
		return "episodic"
	}
	return "unknown"
}
