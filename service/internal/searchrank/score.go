package searchrank

import (
	"github.com/TrebuchetDynamics/goncho/service/internal/searchrank/scoring"
	"github.com/TrebuchetDynamics/goncho/service/internal/searchrank/signals"
)

func GenericAssistantAnswer(content string) bool {
	return signals.GenericAssistantAnswer(content)
}

func PersonalSignalCount(content string) int {
	return signals.PersonalSignalCount(content)
}

func BM25Score(queryTokens map[string]struct{}, tf map[string]int, df map[string]int, docCount, docLength int, avgLength float64) float64 {
	return scoring.BM25(queryTokens, tf, df, docCount, docLength, avgLength)
}
