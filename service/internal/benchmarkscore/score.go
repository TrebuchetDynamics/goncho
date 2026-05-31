package benchmarkscore

import (
	"strings"
	"unicode"

	"github.com/TrebuchetDynamics/goncho/service/internal/recallscore"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

var rubricStopWords = map[string]struct{}{
	"about": {}, "answer": {}, "contain": {}, "contains": {}, "correct": {}, "correctly": {}, "from": {}, "identify": {}, "identifies": {}, "include": {}, "includes": {}, "mention": {}, "mentions": {}, "name": {}, "names": {}, "note": {}, "project": {}, "say": {}, "says": {}, "state": {}, "states": {}, "that": {}, "the": {}, "with": {},
}

// RecallAtK reports the fraction of relevant IDs found in the top-k candidates.
func RecallAtK(candidateIDs, relevantIDs []string, k int) float64 {
	if len(relevantIDs) == 0 || k <= 0 {
		return 0
	}
	relevant := textutil.TrimmedSet(relevantIDs)
	if len(relevant) == 0 {
		return 0
	}
	limit := k
	if len(candidateIDs) < limit {
		limit = len(candidateIDs)
	}
	found := make(map[string]struct{}, len(relevant))
	for _, id := range candidateIDs[:limit] {
		if _, ok := relevant[id]; ok {
			found[id] = struct{}{}
		}
	}
	return RoundFloat(float64(len(found)) / float64(len(relevant)))
}

// RubricCoverage scores how many rubric items have all significant tokens in
// the selected recall context strings.
func RubricCoverage(contexts, rubric []string) (float64, []string) {
	if len(rubric) == 0 {
		return 0, nil
	}
	contextTokens := map[string]struct{}{}
	for _, context := range contexts {
		for _, token := range RubricTokens(context) {
			contextTokens[token] = struct{}{}
		}
	}
	matched := []string{}
	denom := 0
	for _, item := range rubric {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		tokens := RubricTokens(item)
		if len(tokens) == 0 {
			continue
		}
		denom++
		allPresent := true
		for _, token := range tokens {
			if _, ok := contextTokens[token]; !ok {
				allPresent = false
				break
			}
		}
		if allPresent {
			matched = append(matched, item)
		}
	}
	if denom == 0 {
		return 0, nil
	}
	return RoundFloat(float64(len(matched)) / float64(denom)), matched
}

// RubricTokens extracts significant lower-cased tokens for benchmark rubric matching.
func RubricTokens(text string) []string {
	tokens := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	return textutil.NormalizeUnique(tokens, func(token string) string {
		token = strings.TrimSpace(token)
		if len(token) < 2 {
			return ""
		}
		if _, skip := rubricStopWords[token]; skip {
			return ""
		}
		return token
	}, false)
}

func RoundFloat(value float64) float64 {
	return recallscore.Round(value)
}
