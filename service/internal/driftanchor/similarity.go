package driftanchor

import (
	"github.com/TrebuchetDynamics/goncho/service/internal/textmatch"
	"github.com/TrebuchetDynamics/goncho/service/internal/texttokens"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

// Similarity scores prompt-memory overlap for negative drift-anchor warnings.
func Similarity(prompt, memory string) float64 {
	return textmatch.OverlapCoefficient(TokenSet(prompt), TokenSet(memory))
}

// TokenSet normalizes text for drift-anchor matching.
func TokenSet(value string) map[string]struct{} {
	return textutil.Set(texttokens.LowerAlnum(value), func(token string) string {
		if len(token) < 4 || Stopword(token) {
			return ""
		}
		return token
	})
}

// Stopword reports whether token is too generic to anchor repeated-failure matching.
func Stopword(token string) bool {
	switch token {
	case "this", "that", "with", "from", "before", "after", "again", "should", "would", "could", "known":
		return true
	default:
		return false
	}
}
