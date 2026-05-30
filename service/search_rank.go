package goncho

import (
	"slices"

	"github.com/TrebuchetDynamics/goncho/service/internal/searchrank"
	"github.com/TrebuchetDynamics/goncho/service/internal/searchtokens"
)

type searchTemporalDirection = searchrank.TemporalDirection

const (
	searchTemporalNone  = searchrank.TemporalNone
	searchTemporalNewer = searchrank.TemporalNewer
	searchTemporalOlder = searchrank.TemporalOlder
)

type searchTemporalFeatures = searchrank.TemporalFeatures

func rankConclusionHitsByLexicalOverlap(query string, hits []SearchHit) []SearchHit {
	expansion := expandSearchQuery(query)
	queryTokens := searchRankTokenSet(expansion.Expanded)
	if len(queryTokens) == 0 {
		return hits
	}
	if len(hits) < 2 {
		if len(hits) == 1 && expansion.Applied() && keywordRecallScore(hits[0].Content, expansion.Expanded) > 0 {
			hits[0].Provenance = append(hits[0].Provenance, queryExpansionEvidence(expansion))
		}
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
		hit       SearchHit
		score     float64
		baseScore float64
		index     int
	}
	baseScores := make([]float64, len(hits))
	maxScore := 0.0
	for i := range hits {
		baseScores[i] = searchRankBM25Score(queryTokens, docs[i], df, len(hits), docLengths[i], avgLength)
		if baseScores[i] > maxScore {
			maxScore = baseScores[i]
		}
	}
	temporal := searchTemporalIntent(query)
	scored := make([]scoredHit, 0, len(hits))
	for i, hit := range hits {
		score := baseScores[i]
		factScore := searchHitFactIntentScore(query, hit)
		if score == 0 && factScore == 0 {
			continue
		}
		score += searchFactIntentBonus(factScore, maxScore)
		score += searchTemporalRerankBonus(temporal, hit.Content, i, len(hits), score, maxScore)
		if expansion.Applied() {
			hit.Provenance = append(hit.Provenance, queryExpansionEvidence(expansion))
		}
		scored = append(scored, scoredHit{hit: hit, score: score, baseScore: baseScores[i], index: i})
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

func searchTemporalIntent(query string) searchTemporalFeatures {
	return searchrank.TemporalIntent(query)
}

func searchTemporalQuery(query string) bool {
	return searchrank.TemporalQuery(query)
}

func searchTemporalMarkers(query string) []string {
	return searchrank.TemporalMarkers(query)
}

func searchTemporalRerankBonus(features searchTemporalFeatures, content string, index, total int, score, maxScore float64) float64 {
	return searchrank.TemporalRerankBonus(features, content, index, total, score, maxScore)
}

func searchGenericAssistantAnswer(content string) bool {
	return searchrank.GenericAssistantAnswer(content)
}

func searchPersonalSignalCount(content string) int {
	return searchrank.PersonalSignalCount(content)
}

func searchRankBM25Score(queryTokens map[string]struct{}, tf map[string]int, df map[string]int, docCount, docLength int, avgLength float64) float64 {
	return searchrank.BM25Score(queryTokens, tf, df, docCount, docLength, avgLength)
}

func searchRankTermFrequency(value string) map[string]int {
	return searchtokens.TermFrequency(value)
}

func searchRankTokenSet(value string) map[string]struct{} {
	return searchtokens.TokenSet(value)
}

func searchRankTokens(value string) []string {
	return searchtokens.Tokens(value)
}
