package goncho

import (
	"slices"

	"github.com/TrebuchetDynamics/goncho/service/internal/recallscore"
	"github.com/TrebuchetDynamics/goncho/service/internal/searchrank"
	"github.com/TrebuchetDynamics/goncho/service/internal/searchtokens"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

type searchTemporalDirection = searchrank.TemporalDirection

const (
	searchTemporalNone  = searchrank.TemporalNone
	searchTemporalNewer = searchrank.TemporalNewer
	searchTemporalOlder = searchrank.TemporalOlder
)

type searchTemporalFeatures = searchrank.TemporalFeatures

type searchRankScoredHit struct {
	hit       SearchHit
	score     float64
	baseScore float64
	index     int
}

type searchRankCandidateDecision struct {
	hit       SearchHit
	score     float64
	baseScore float64
	keep      bool
}

func rankConclusionHitsByLexicalOverlap(query string, hits []SearchHit) []SearchHit {
	return rankConclusionHitsByLexicalOverlapWithAliases(query, hits, nil)
}

func rankConclusionHitsByLexicalOverlapWithAliases(query string, hits []SearchHit, aliases map[string][]string) []SearchHit {
	expansion := expandSearchQueryWithAliases(query, aliases)
	queryTokens := searchRankTokenSet(expansion.Expanded)
	if len(queryTokens) == 0 {
		return hits
	}
	if len(hits) == 0 {
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
	baseScores := make([]float64, len(hits))
	maxScore := 0.0
	for i := range hits {
		baseScores[i] = searchRankBM25Score(queryTokens, docs[i], df, len(hits), docLengths[i], avgLength)
		if baseScores[i] > maxScore {
			maxScore = baseScores[i]
		}
	}
	temporal := searchTemporalIntent(query)
	scored := make([]searchRankScoredHit, 0, len(hits))
	for i, hit := range hits {
		decision := searchRankCandidateDecisionFor(query, expansion, temporal, hit, i, len(hits), baseScores[i], maxScore)
		if !decision.keep {
			continue
		}
		scored = append(scored, searchRankScoredHit{hit: decision.hit, score: decision.score, baseScore: decision.baseScore, index: i})
	}
	slices.SortStableFunc(scored, func(a, b searchRankScoredHit) int {
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
	return sliceutil.Map(scored, func(item searchRankScoredHit) SearchHit {
		return item.hit
	})
}

func searchRankCandidateDecisionFor(query string, expansion expandedQuery, temporal searchTemporalFeatures, hit SearchHit, index, total int, baseScore, maxScore float64) searchRankCandidateDecision {
	decision := searchRankCandidateDecision{hit: hit, score: baseScore, baseScore: baseScore}
	factScore := searchHitFactIntentScore(query, hit)
	if baseScore == 0 && factScore == 0 {
		return decision
	}
	decision.score += searchFactIntentBonus(factScore, maxScore)
	decision.score += searchTemporalRerankBonus(temporal, hit.Content, index, total, decision.score, maxScore)
	if searchHitExpansionImproves(expansion, hit) {
		decision.hit.Provenance = appendSearchHitQueryExpansionEvidence(hit.Provenance, expansion)
	}
	decision.keep = true
	return decision
}

func searchHitExpansionImproves(expansion expandedQuery, hit SearchHit) bool {
	if !expansion.Applied() {
		return false
	}
	originalScore := recallscore.Keyword(hit.Content, expansion.Original)
	expandedScore := recallscore.Keyword(hit.Content, expansion.Expanded)
	return expandedScore > originalScore
}

func appendSearchHitQueryExpansionEvidence(provenance []EvidenceItem, expansion expandedQuery) []EvidenceItem {
	if evidenceListHas(provenance, "query_expansion", queryExpansionEvidenceID(expansion)) {
		return provenance
	}
	return append(provenance, queryExpansionEvidence(expansion))
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
