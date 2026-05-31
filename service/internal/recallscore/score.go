package recallscore

import (
	"math"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/texttokens"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

func Keyword(content, query string) float64 {
	query = textutil.LowerTrimmed(query)
	if query == "" {
		return 0
	}
	contentTokensList := texttokens.LowerAlnum(content)
	queryTokenSequence := texttokens.LowerAlnum(query)
	if tokenSequenceContains(contentTokensList, queryTokenSequence) {
		return 1
	}
	queryTokens := textutil.UniqueTrimmed(queryTokenSequence, false)
	if len(queryTokens) == 0 {
		return 0
	}
	contentTokens := tokenSet(contentTokensList)
	hits := 0
	for _, token := range queryTokens {
		if _, ok := contentTokens[token]; ok {
			hits++
		}
	}
	return Clamp(float64(hits) / float64(len(queryTokens)))
}

func tokenSequenceContains(contentTokens, queryTokens []string) bool {
	if len(queryTokens) == 0 || len(queryTokens) > len(contentTokens) {
		return false
	}
	for start := 0; start <= len(contentTokens)-len(queryTokens); start++ {
		matched := true
		for offset, token := range queryTokens {
			if contentTokens[start+offset] != token {
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

func tokenSet(tokens []string) map[string]struct{} {
	out := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		out[token] = struct{}{}
	}
	return out
}

func Recency(createdAt, now time.Time, halfLife time.Duration) float64 {
	if createdAt.IsZero() || now.IsZero() || halfLife <= 0 {
		return 0
	}
	age := now.Sub(createdAt.UTC())
	if age <= 0 {
		return 1
	}
	halfLives := float64(age) / float64(halfLife)
	return Clamp(math.Exp2(-halfLives))
}

func Clamp(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func Round(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}
