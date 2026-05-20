package goncho

import (
	"regexp"
	"slices"
	"strings"
)

var searchRankTokenPattern = regexp.MustCompile(`[a-z0-9]+`)

func rankConclusionHitsByLexicalOverlap(query string, hits []SearchHit) []SearchHit {
	queryTokens := searchRankTokenSet(query)
	if len(queryTokens) == 0 || len(hits) < 2 {
		return hits
	}
	type scoredHit struct {
		hit   SearchHit
		score int
		index int
	}
	scored := make([]scoredHit, 0, len(hits))
	for i, hit := range hits {
		score := searchRankOverlapScore(queryTokens, hit.Content)
		if score == 0 {
			continue
		}
		scored = append(scored, scoredHit{hit: hit, score: score, index: i})
	}
	slices.SortStableFunc(scored, func(a, b scoredHit) int {
		if a.score > b.score {
			return -1
		}
		if a.score < b.score {
			return 1
		}
		if a.index < b.index {
			return -1
		}
		if a.index > b.index {
			return 1
		}
		return 0
	})
	out := make([]SearchHit, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.hit)
	}
	return out
}

func searchRankOverlapScore(queryTokens map[string]struct{}, content string) int {
	contentTokens := searchRankTokenSet(content)
	score := 0
	for token := range queryTokens {
		if _, ok := contentTokens[token]; ok {
			score++
		}
	}
	return score
}

func searchRankTokenSet(value string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, token := range searchRankTokenPattern.FindAllString(strings.ToLower(value), -1) {
		if len(token) < 3 || searchRankStopword(token) {
			continue
		}
		out[token] = struct{}{}
	}
	return out
}

func searchRankStopword(token string) bool {
	switch token {
	case "the", "and", "for", "who", "what", "when", "where", "which", "should", "not", "did", "does", "with", "that", "this", "from", "are", "was", "were", "has", "have", "had":
		return true
	default:
		return false
	}
}
