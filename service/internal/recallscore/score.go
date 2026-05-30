package recallscore

import (
	"math"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

func Keyword(content, query string) float64 {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return 0
	}
	content = strings.ToLower(content)
	if strings.Contains(content, query) {
		return 1
	}
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return 0
	}
	tokens = textutil.UniqueTrimmed(tokens, false)
	hits := 0
	for _, token := range tokens {
		if strings.Contains(content, token) {
			hits++
		}
	}
	if len(tokens) == 0 {
		return 0
	}
	return Clamp(float64(hits) / float64(len(tokens)))
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
