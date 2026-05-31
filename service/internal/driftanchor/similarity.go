package driftanchor

import (
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textmatch"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

// Similarity scores prompt-memory overlap for negative drift-anchor warnings.
func Similarity(prompt, memory string) float64 {
	return textmatch.OverlapCoefficient(TokenSet(prompt), TokenSet(memory))
}

// TokenSet normalizes text for drift-anchor matching.
func TokenSet(value string) map[string]struct{} {
	return textutil.Set(tokenPattern.FindAllString(strings.ToLower(value), -1), func(token string) string {
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
