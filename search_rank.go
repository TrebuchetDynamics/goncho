package goncho

import (
	"math"
	"regexp"
	"slices"
	"strings"
)

var searchRankTokenPattern = regexp.MustCompile(`[a-z0-9]+`)

type searchTemporalDirection int

const (
	searchTemporalNone searchTemporalDirection = iota
	searchTemporalNewer
	searchTemporalOlder
)

type searchTemporalFeatures struct {
	Direction searchTemporalDirection
	Markers   []string
	Temporal  bool
}

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
		factScore := searchFactIntentScore(query, hit.Content)
		if score == 0 && factScore == 0 {
			continue
		}
		score += searchFactIntentBonus(factScore, maxScore)
		score += searchTemporalRerankBonus(temporal, hit.Content, i, len(hits), score, maxScore)
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
	q := strings.ToLower(query)
	features := searchTemporalFeatures{Markers: searchTemporalMarkers(q), Temporal: searchTemporalQuery(q)}
	if strings.Contains(q, "first") || strings.Contains(q, "earliest") || strings.Contains(q, "initial") || strings.Contains(q, "original") || strings.Contains(q, "started first") {
		features.Direction = searchTemporalOlder
		return features
	}
	if strings.Contains(q, "latest") || strings.Contains(q, "current") || strings.Contains(q, "currently") || strings.Contains(q, "most recently") {
		features.Direction = searchTemporalNewer
		return features
	}
	return features
}

func searchTemporalQuery(query string) bool {
	needles := []string{"when", "first", "earliest", "initial", "original", "latest", "current", "currently", "recent", "today", "yesterday", "tomorrow", "last ", "this ", "past ", "how many days", "how many weeks", "how many months", "how many years", "how long", "order of"}
	for _, needle := range needles {
		if strings.Contains(query, needle) {
			return true
		}
	}
	return false
}

func searchTemporalMarkers(query string) []string {
	candidates := []string{
		"today", "yesterday", "tomorrow", "most recently", "this weekend", "this week", "this month", "this year", "past few months", "past three months", "last week", "last month", "last year", "last friday", "last saturday", "last sunday",
		"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december",
	}
	markers := []string{}
	for _, candidate := range candidates {
		if strings.Contains(query, candidate) {
			markers = append(markers, candidate)
		}
	}
	return markers
}

func searchTemporalRerankBonus(features searchTemporalFeatures, content string, index, total int, score, maxScore float64) float64 {
	if total < 2 || maxScore <= 0 {
		return 0
	}
	if features.Temporal && searchGenericAssistantAnswer(content) && searchPersonalSignalCount(content) < 12 && score >= maxScore*0.70 {
		return -maxScore * 0.30
	}
	if features.Direction == searchTemporalNone || score < maxScore*0.78 {
		return 0
	}
	contentLower := strings.ToLower(content)
	markerMatches := 0
	for _, marker := range features.Markers {
		if strings.Contains(contentLower, marker) {
			markerMatches++
		}
	}
	alignment := float64(markerMatches)
	switch features.Direction {
	case searchTemporalNewer:
		// Newer/current phrasing is common in distractors (for example "new products"),
		// so only exact query temporal marker matches contribute positive evidence.
	case searchTemporalOlder:
		if strings.Contains(contentLower, "first") || strings.Contains(contentLower, "initial") || strings.Contains(contentLower, "original") || strings.Contains(contentLower, "earliest") || strings.Contains(contentLower, "started") || strings.Contains(contentLower, "began") {
			alignment += 0.5
		}
	}
	if alignment == 0 {
		return 0
	}
	position := 0.0
	if total > 1 {
		position = float64(total-1-index) / float64(total-1)
		if features.Direction == searchTemporalOlder {
			position = float64(index) / float64(total-1)
		}
	}
	return maxScore * 0.08 * alignment * (0.5 + 0.5*position)
}

func searchGenericAssistantAnswer(content string) bool {
	content = strings.ToLower(content)
	return strings.Contains(content, "as an ai language model") || strings.Contains(content, "i cannot provide") || strings.Contains(content, "i don't have personal experience")
}

func searchPersonalSignalCount(content string) int {
	count := 0
	for _, token := range searchRankTokenPattern.FindAllString(strings.ToLower(content), -1) {
		switch token {
		case "i", "my", "me", "mine", "myself":
			count++
		}
	}
	return count
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
