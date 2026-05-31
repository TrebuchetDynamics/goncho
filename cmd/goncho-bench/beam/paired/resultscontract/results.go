package resultscontract

import (
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
)

const DefaultConfigID = "beam-results"

type File struct {
	Metadata struct {
		ConfigID     string `json:"config_id"`
		RunStartedAt string `json:"run_started_at"`
		Date         string `json:"date"`
	} `json:"metadata"`
	Results []struct {
		ConversationID string `json:"conversation_id"`
		Scale          string `json:"scale"`
		Results        []struct {
			QID      string  `json:"qid"`
			Ability  string  `json:"ability"`
			Question string  `json:"question"`
			Score    float64 `json:"score"`
		} `json:"results"`
	} `json:"results"`
}

func OutcomesFromResults(results File, overrideConfigID, sourcePath, sourceSHA256 string) ([]shared.PairedOutcome, error) {
	configID := shared.FirstNonEmptyTrimmed(overrideConfigID, results.Metadata.ConfigID, DefaultConfigID)
	runStartedAt := shared.FirstNonEmptyTrimmed(results.Metadata.RunStartedAt, results.Metadata.Date)
	if runStartedAt == "" {
		runStartedAt = shared.FormatArtifactTimestamp(time.Now())
	}
	out := []shared.PairedOutcome{}
	for conversationIndex, conv := range results.Results {
		conversationID := strings.TrimSpace(conv.ConversationID)
		scale := strings.TrimSpace(conv.Scale)
		for resultIndex, result := range conv.Results {
			qid := strings.TrimSpace(result.QID)
			if qid == "" {
				return nil, fmt.Errorf("goncho-bench: BEAM paired result conversation %d result %d missing qid", conversationIndex+1, resultIndex+1)
			}
			score := shared.RoundMetric(result.Score)
			out = append(out, shared.PairedOutcome{
				ConfigID:       configID,
				RunStartedAt:   runStartedAt,
				Scale:          scale,
				ConversationID: conversationID,
				QID:            qid,
				Ability:        shared.NormalizeAbility(result.Ability),
				Question:       strings.TrimSpace(result.Question),
				SourcePath:     strings.TrimSpace(sourcePath),
				SourceSHA256:   strings.TrimSpace(sourceSHA256),
				Score:          score,
				Correct:        shared.PairedOutcomeCorrect(score),
			})
		}
	}
	return out, nil
}
