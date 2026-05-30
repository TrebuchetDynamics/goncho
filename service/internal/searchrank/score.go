package searchrank

import (
	"math"

	"github.com/TrebuchetDynamics/goncho/service/internal/searchtokens"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

func GenericAssistantAnswer(content string) bool {
	return textutil.ContainsAnySubstringFold(content, []string{"as an ai language model", "i cannot provide", "i don't have personal experience"})
}

func PersonalSignalCount(content string) int {
	count := 0
	for _, token := range searchtokens.Tokens(content) {
		switch token {
		case "i", "my", "me", "mine", "myself":
			count++
		}
	}
	return count
}

func BM25Score(queryTokens map[string]struct{}, tf map[string]int, df map[string]int, docCount, docLength int, avgLength float64) float64 {
	const k1 = 1.2
	const b = 0.75
	if docCount == 0 || docLength == 0 || avgLength <= 0 {
		return 0
	}
	score := 0.0
	for token := range queryTokens {
		freq := tf[token]
		if freq == 0 {
			continue
		}
		docFreq := df[token]
		idf := math.Log(1 + (float64(docCount)-float64(docFreq)+0.5)/(float64(docFreq)+0.5))
		denom := float64(freq) + k1*(1-b+b*(float64(docLength)/avgLength))
		score += idf * (float64(freq) * (k1 + 1) / denom)
	}
	return score
}
