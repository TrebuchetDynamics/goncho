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
	if score := searchPreferenceFactIntentScore(query, content); score > 0 {
		return score
	}
	if score := searchLocationFactIntentScore(query, content); score > 0 {
		return score
	}
	if score := searchInstructionFactIntentScore(query, content); score > 0 {
		return score
	}
	return 0
}

func searchHitFactIntentScore(query string, hit SearchHit) float64 {
	score := searchFactIntentScore(query, hit.Content)
	for _, fact := range hit.factAnnotations {
		if factScore := searchFactIntentScore(query, fact); factScore > score {
			score = factScore
		}
	}
	return score
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

func searchPreferenceFactIntentScore(query, content string) float64 {
	querySubject, queryAttribute, ok := searchPreferenceQuestion(query)
	if !ok {
		return 0
	}
	subjectTokens := searchRankTokenSet(querySubject)
	attributeTokens := searchRankTokenSet(queryAttribute)
	if len(subjectTokens) == 0 || len(attributeTokens) == 0 {
		return 0
	}
	for _, sentence := range recallSentencePattern.FindAllString(content, -1) {
		if strings.Contains(sentence, "?") {
			continue
		}
		subject, _, attribute, ok := searchPreferenceAnswerParts(sentence)
		if !ok {
			continue
		}
		if searchRankTokenCoverage(subjectTokens, subject) < 0.80 {
			continue
		}
		if searchRankTokenCoverage(attributeTokens, attribute) >= 0.80 {
			return 1
		}
	}
	return 0
}

func searchLocationFactIntentScore(query, content string) float64 {
	queryObject, ok := searchLocationQuestionObject(query)
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
		object, location, ok := searchLocationAnswerParts(sentence)
		if !ok || !searchFactObjectLooksAssertive(location) {
			continue
		}
		if searchRankTokenCoverage(queryTokens, object) >= 0.80 {
			return 1
		}
	}
	return 0
}

func searchLocationQuestionObject(query string) (string, bool) {
	query = strings.TrimSpace(strings.Trim(query, "?!."))
	lower := strings.ToLower(query)
	for _, prefix := range []string{"where is ", "where are ", "where's "} {
		if strings.HasPrefix(lower, prefix) {
			object := cleanFactObject(query[len(prefix):])
			return object, object != ""
		}
	}
	return "", false
}

func searchLocationAnswerParts(sentence string) (object, location string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	lower := strings.ToLower(sentence)
	for _, marker := range []string{" is located at ", " is located in ", " is in ", " lives in "} {
		idx := strings.Index(lower, marker)
		if idx <= 0 {
			continue
		}
		object = cleanFactObject(sentence[:idx])
		location = cleanFactValue(sentence[idx+len(marker):])
		return object, location, searchFactObjectLooksAssertive(object) && location != ""
	}
	return "", "", false
}

func searchInstructionFactIntentScore(query, content string) float64 {
	querySubject, queryTopic, ok := searchInstructionQuestion(query)
	if !ok {
		return 0
	}
	subjectTokens := searchRankTokenSet(querySubject)
	topicTokens := searchRankTokenSet(queryTopic)
	if len(subjectTokens) == 0 || len(topicTokens) == 0 {
		return 0
	}
	for _, sentence := range recallSentencePattern.FindAllString(content, -1) {
		if strings.Contains(sentence, "?") {
			continue
		}
		subject, instruction, ok := searchInstructionAnswerParts(sentence)
		if !ok {
			continue
		}
		if searchRankTokenCoverage(subjectTokens, subject) < 0.80 {
			continue
		}
		if searchRankTokenCoverage(topicTokens, instruction) >= 0.80 {
			return 1
		}
	}
	return 0
}

func searchInstructionQuestion(query string) (subject, topic string, ok bool) {
	query = strings.TrimSpace(strings.Trim(query, "?!."))
	lower := strings.ToLower(query)
	for _, prefix := range []string{"what instruction did ", "what rule did "} {
		if !strings.HasPrefix(lower, prefix) {
			continue
		}
		rest := query[len(prefix):]
		restLower := strings.ToLower(rest)
		giveIdx := strings.Index(restLower, " give")
		if giveIdx <= 0 {
			return "", "", false
		}
		subject = cleanFactValue(rest[:giveIdx])
		afterGive := strings.TrimSpace(rest[giveIdx+len(" give"):])
		afterLower := strings.ToLower(afterGive)
		aboutIdx := strings.LastIndex(afterLower, " about ")
		if aboutIdx >= 0 {
			topic = cleanFactObject(afterGive[aboutIdx+len(" about "):])
			return subject, topic, subject != "" && topic != ""
		}
		if strings.HasPrefix(afterLower, "about ") {
			topic = cleanFactObject(afterGive[len("about "):])
			return subject, topic, subject != "" && topic != ""
		}
		return "", "", false
		return subject, topic, subject != "" && topic != ""
	}
	return "", "", false
}

func searchInstructionAnswerParts(sentence string) (subject, instruction string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	lower := strings.ToLower(sentence)
	idx := strings.Index(lower, " instructed ")
	if idx <= 0 {
		return "", "", false
	}
	subject = cleanFactValue(sentence[:idx])
	instruction = cleanFactValue(sentence[idx+len(" instructed "):])
	if !searchFactSubjectLooksAssertive(subject) || !searchFactObjectLooksAssertive(instruction) {
		return "", "", false
	}
	return subject, instruction, true
}

func searchOwnerQuestionObject(query string) (string, bool) {
	match := searchOwnerQuestionPattern.FindStringSubmatch(query)
	if len(match) != 2 {
		return "", false
	}
	object := strings.TrimSpace(match[1])
	return object, object != ""
}

func searchPreferenceQuestion(query string) (subject, attribute string, ok bool) {
	query = strings.TrimSpace(strings.Trim(query, "?!."))
	lower := strings.ToLower(query)
	if strings.HasPrefix(lower, "what does ") {
		rest := query[len("what does "):]
		restLower := strings.ToLower(rest)
		preferIdx := strings.Index(restLower, " prefer")
		if preferIdx <= 0 {
			return "", "", false
		}
		subject = cleanFactValue(rest[:preferIdx])
		after := strings.TrimSpace(rest[preferIdx+len(" prefer"):])
		for _, prefix := range []string{"for ", "as "} {
			if strings.HasPrefix(strings.ToLower(after), prefix) {
				attribute = cleanFactObject(after[len(prefix):])
				return subject, attribute, subject != "" && attribute != ""
			}
		}
		return "", "", false
	}
	if !strings.HasPrefix(lower, "what ") {
		return "", "", false
	}
	rest := query[len("what "):]
	restLower := strings.ToLower(rest)
	doesIdx := strings.Index(restLower, " does ")
	if doesIdx <= 0 {
		return "", "", false
	}
	attribute = cleanFactObject(rest[:doesIdx])
	afterDoes := rest[doesIdx+len(" does "):]
	afterLower := strings.ToLower(afterDoes)
	preferIdx := strings.Index(afterLower, " prefer")
	if preferIdx <= 0 {
		return "", "", false
	}
	subject = cleanFactValue(afterDoes[:preferIdx])
	return subject, attribute, subject != "" && attribute != ""
}

func searchPreferenceAnswerParts(sentence string) (subject, value, attribute string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	lower := strings.ToLower(sentence)
	idx := strings.Index(lower, " prefers ")
	verbLen := len(" prefers ")
	if idx < 0 {
		idx = strings.Index(lower, " prefer ")
		verbLen = len(" prefer ")
	}
	if idx <= 0 {
		return "", "", "", false
	}
	subject = cleanFactValue(sentence[:idx])
	rest := sentence[idx+verbLen:]
	restLower := strings.ToLower(rest)
	forIdx := strings.LastIndex(restLower, " for ")
	if forIdx <= 0 {
		return "", "", "", false
	}
	value = cleanFactValue(rest[:forIdx])
	attribute = cleanFactObject(rest[forIdx+len(" for "):])
	if !searchFactSubjectLooksAssertive(subject) || !searchFactObjectLooksAssertive(value) || !searchFactObjectLooksAssertive(attribute) {
		return "", "", "", false
	}
	return subject, value, attribute, true
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
