package goncho

import (
	"math"
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
	docs := make([]map[string]int, len(hits))
	docLengths := make([]int, len(hits))
	df := map[string]int{}
	totalLength := 0
	for i, hit := range hits {
		tf := searchRankTermFrequency(hit.Content)
		docs[i] = tf
		for _, count := range tf {
			docLengths[i] += count
		}
		totalLength += docLengths[i]
		for token := range queryTokens {
			if tf[token] > 0 {
				df[token]++
			}
		}
	}
	avgLength := 1.0
	if len(hits) > 0 && totalLength > 0 {
		avgLength = float64(totalLength) / float64(len(hits))
	}
	type scoredHit struct {
		hit   SearchHit
		score float64
		index int
	}
	scored := make([]scoredHit, 0, len(hits))
	for i, hit := range hits {
		score := searchRankBM25Score(queryTokens, docs[i], df, len(hits), docLengths[i], avgLength)
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

func searchRankBM25Score(queryTokens map[string]struct{}, tf map[string]int, df map[string]int, docCount, docLength int, avgLength float64) float64 {
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

func searchRankTermFrequency(value string) map[string]int {
	out := map[string]int{}
	for _, token := range searchRankTokens(value) {
		out[token]++
	}
	return out
}

func searchRankTokenSet(value string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, token := range searchRankTokens(value) {
		out[token] = struct{}{}
	}
	return out
}

func searchRankTokens(value string) []string {
	out := []string{}
	for _, token := range searchRankTokenPattern.FindAllString(strings.ToLower(value), -1) {
		token = searchRankStem(token)
		if len(token) < 3 || searchRankStopword(token) {
			continue
		}
		out = append(out, token)
	}
	return out
}

func searchRankStem(token string) string {
	for _, suffix := range []string{"ing", "edly", "edly", "ed", "es", "s"} {
		if len(token) > len(suffix)+3 && strings.HasSuffix(token, suffix) {
			return strings.TrimSuffix(token, suffix)
		}
	}
	return token
}

func searchRankStopword(token string) bool {
	switch token {
	case "the", "and", "for", "who", "what", "when", "where", "which", "should", "not", "did", "does", "with", "that", "this", "from", "are", "was", "were", "has", "have", "had", "you", "your", "about", "can", "could", "would", "there", "their", "they", "them", "then", "than":
		return true
	default:
		return false
	}
}
