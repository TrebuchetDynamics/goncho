package goncho

import (
	"regexp"
	"slices"
	"strings"
)

var (
	searchOwnerQuestionPattern = regexp.MustCompile(`(?i)\bwho\s+(?:currently\s+|now\s+)?owns?\s+([^?!.]+)`)
	searchOwnerAnswerPattern   = regexp.MustCompile(`(?i)^\s*([a-z][a-z0-9 _.'-]{0,80}?)\s+(?:currently\s+|now\s+)?owns?\s+(.+?)\s*$`)
	recallSentencePattern      = regexp.MustCompile(`[^.!?]+[.!?]?`)
)

func searchFactIntentScore(query, content string) float64 {
	if score := searchOwnerFactIntentScore(query, content); score > 0 {
		return score
	}
	return 0
}

func searchOwnerFactIntentScore(query, content string) float64 {
	queryObject, ok := searchOwnerQuestionObject(query)
	if !ok {
		return 0
	}
	queryTokens := searchRankTokenSet(queryObject)
	if len(queryTokens) == 0 {
		return 0
	}
	for _, sentence := range recallSentencePattern.FindAllString(content, -1) {
		if strings.Contains(sentence, "?") {
			continue
		}
		match := searchOwnerAnswerPattern.FindStringSubmatch(sentence)
		if len(match) != 3 {
			continue
		}
		subject := strings.TrimSpace(match[1])
		object := strings.TrimSpace(match[2])
		if !searchFactSubjectLooksAssertive(subject) || searchRankTokenCoverage(queryTokens, subject) > 0 {
			continue
		}
		if !searchFactObjectLooksAssertive(object) {
			continue
		}
		if searchRankTokenCoverage(queryTokens, object) >= 0.80 {
			return 1
		}
	}
	return 0
}

func searchOwnerQuestionObject(query string) (string, bool) {
	match := searchOwnerQuestionPattern.FindStringSubmatch(query)
	if len(match) != 2 {
		return "", false
	}
	object := strings.TrimSpace(match[1])
	return object, object != ""
}

func searchFactSubjectLooksAssertive(subject string) bool {
	tokens := searchRankTokens(subject)
	if len(tokens) == 0 {
		return false
	}
	if len(tokens) > 6 {
		return false
	}
	for _, token := range tokens {
		if slices.Contains([]string{"who", "what", "which", "ask", "checklist", "question", "answer", "own"}, token) {
			return false
		}
	}
	return true
}

func searchFactObjectLooksAssertive(object string) bool {
	tokens := searchRankTokens(object)
	if len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		if slices.Contains([]string{"own", "ask", "question", "answer", "checklist"}, token) {
			return false
		}
	}
	return true
}

func searchRankTokenCoverage(want map[string]struct{}, value string) float64 {
	if len(want) == 0 {
		return 0
	}
	got := searchRankTokenSet(value)
	if len(got) == 0 {
		return 0
	}
	hits := 0
	for token := range want {
		if _, ok := got[token]; ok {
			hits++
		}
	}
	return float64(hits) / float64(len(want))
}

func searchFactIntentBonus(factScore, maxBaseScore float64) float64 {
	if factScore <= 0 {
		return 0
	}
	if maxBaseScore <= 0 {
		return factScore
	}
	return maxBaseScore * 1.10 * factScore
}
