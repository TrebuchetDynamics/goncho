package goncho

import (
	"regexp"
	"slices"
	"strings"
)

const searchMetricUnitPattern = `ms|sec|seconds?|minutes?|hours?|days?|weeks?|months?|%|kb|mb|gb|tb|rows?|columns?|roles?|features?|bugs?|commits?|cards?|users?|items?|tests?|apis?|endpoints?|tickets?`

var (
	searchOwnerQuestionPattern = regexp.MustCompile(`(?i)\bwho\s+(?:currently\s+|now\s+)?owns?\s+([^?!.]+)`)
	searchOwnerAnswerPattern   = regexp.MustCompile(`(?i)^\s*([a-z][a-z0-9 _.'-]{0,80}?)\s+(?:currently\s+|now\s+)?owns?\s+(.+?)\s*$`)
	searchMetricValuePattern   = regexp.MustCompile(`(?i)^\d+(?:[.,]\d+)?\s*(?:` + searchMetricUnitPattern + `)\s*$`)
	searchMetricAnswerPattern  = regexp.MustCompile(`(?i)^\s*(.+?)\s+(?:is|was|=)\s+(\d+(?:[.,]\d+)?\s*(?:` + searchMetricUnitPattern + `))\s*$`)
	searchVersionValuePattern  = regexp.MustCompile(`(?i)^v?\d+\.\d+(?:\.\d+)?\s*$`)
	searchVersionIsPattern     = regexp.MustCompile(`(?i)^\s*(.+?)\s+version\s+(?:is|was|=)\s+(v?\d+\.\d+(?:\.\d+)?)\s*$`)
	searchVersionShortPattern  = regexp.MustCompile(`(?i)^\s*(.+?)\s+v(\d+\.\d+(?:\.\d+)?)\s*$`)
	searchNegationPattern      = regexp.MustCompile(`(?i)^\s*(?:project note:\s*)?(?:i|we|user)\s+(?:(?:have|has|had|did)\s+)?(?:never|not)\s+(.+?)\s*$`)
	searchDecisionPattern      = regexp.MustCompile(`(?i)^\s*(?:project note:\s*)?(?:i|we|user)\s+(?:decided to|chose to|opted for|selected|picked|switching to)\s+(.+?)\s*$`)
	searchSequenceMarkers      = []string{"first", "second", "third", "fourth", "fifth", "finally", "next", "then", "after that"}
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
	if score := searchTimelineFactIntentScore(query, content); score > 0 {
		return score
	}
	if score := searchMetricFactIntentScore(query, content); score > 0 {
		return score
	}
	if score := searchVersionFactIntentScore(query, content); score > 0 {
		return score
	}
	if score := searchSequenceFactIntentScore(query, content); score > 0 {
		return score
	}
	if score := searchNegationFactIntentScore(query, content); score > 0 {
		return score
	}
	if score := searchDecisionFactIntentScore(query, content); score > 0 {
		return score
	}
	return 0
}

func searchHitFactIntentScore(query string, hit SearchHit) float64 {
	score := searchFactIntentScore(query, hit.Content)
	for _, fact := range hit.factAnnotations {
		if factScore := searchFactIntentScore(query, fact.Value); factScore > score {
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

func searchTimelineFactIntentScore(query, content string) float64 {
	queryEvent, ok := searchTimelineQuestionEvent(query)
	if !ok {
		return 0
	}
	queryTokens := searchRankTokenSet(queryEvent)
	if len(queryTokens) == 0 {
		return 0
	}
	for _, sentence := range recallSentencePattern.FindAllString(content, -1) {
		if strings.Contains(sentence, "?") {
			continue
		}
		event, date, ok := searchTimelineAnswerParts(sentence)
		if !ok || !searchFactObjectLooksAssertive(date) {
			continue
		}
		if searchRankTokenCoverage(queryTokens, event) >= 0.80 {
			return 1
		}
	}
	return 0
}

func searchTimelineQuestionEvent(query string) (string, bool) {
	query = strings.TrimSpace(strings.Trim(query, "?!."))
	lower := strings.ToLower(query)
	for _, prefix := range []string{"when is ", "when are "} {
		if strings.HasPrefix(lower, prefix) {
			event := cleanFactObject(query[len(prefix):])
			return event, event != ""
		}
	}
	return "", false
}

func searchTimelineAnswerParts(sentence string) (event, date string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	lower := strings.ToLower(sentence)
	for _, marker := range []string{" occurs on ", " is scheduled for ", " deadline is ", " is on "} {
		idx := strings.Index(lower, marker)
		if idx <= 0 {
			continue
		}
		event = cleanFactObject(sentence[:idx])
		date = cleanFactValue(sentence[idx+len(marker):])
		return event, date, searchFactObjectLooksAssertive(event) && date != ""
	}
	return "", "", false
}

func searchMetricFactIntentScore(query, content string) float64 {
	queryKey, ok := searchMetricQuestionKey(query)
	if !ok {
		return 0
	}
	queryTokens := searchRankTokenSet(queryKey)
	if len(queryTokens) == 0 {
		return 0
	}
	if !strings.Contains(content, "?") {
		key, value, ok := searchMetricAnswerParts(content)
		if ok && searchMetricValueLooksAssertive(value) && searchRankTokenCoverage(queryTokens, key) >= 0.80 {
			return 1
		}
	}
	for _, sentence := range recallSentencePattern.FindAllString(content, -1) {
		if strings.Contains(sentence, "?") {
			continue
		}
		key, value, ok := searchMetricAnswerParts(sentence)
		if !ok || !searchMetricValueLooksAssertive(value) {
			continue
		}
		if searchRankTokenCoverage(queryTokens, key) >= 0.80 {
			return 1
		}
	}
	return 0
}

func searchMetricQuestionKey(query string) (string, bool) {
	query = strings.TrimSpace(strings.Trim(query, "?! ."))
	lower := strings.ToLower(query)
	for _, prefix := range []string{"what is ", "what was ", "what are ", "what were ", "how fast is ", "how many ", "how much "} {
		if strings.HasPrefix(lower, prefix) {
			key := cleanFactObject(query[len(prefix):])
			return key, key != ""
		}
	}
	return "", false
}

func searchMetricAnswerParts(sentence string) (key, value string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	match := searchMetricAnswerPattern.FindStringSubmatch(sentence)
	if len(match) != 3 {
		return "", "", false
	}
	key = cleanFactObject(match[1])
	value = cleanFactValue(match[2])
	return key, value, searchFactObjectLooksAssertive(key) && searchMetricValueLooksAssertive(value)
}

func searchVersionFactIntentScore(query, content string) float64 {
	querySubject, ok := searchVersionQuestionSubject(query)
	if !ok {
		return 0
	}
	queryTokens := searchRankTokenSet(querySubject)
	if len(queryTokens) == 0 {
		return 0
	}
	if !strings.Contains(content, "?") {
		subject, version, ok := searchVersionAnswerParts(content)
		if ok && searchVersionValueLooksAssertive(version) && searchRankTokenCoverage(queryTokens, subject) >= 0.80 {
			return 1
		}
	}
	for _, sentence := range recallSentencePattern.FindAllString(content, -1) {
		if strings.Contains(sentence, "?") {
			continue
		}
		subject, version, ok := searchVersionAnswerParts(sentence)
		if !ok || !searchVersionValueLooksAssertive(version) {
			continue
		}
		if searchRankTokenCoverage(queryTokens, subject) >= 0.80 {
			return 1
		}
	}
	return 0
}

func searchVersionQuestionSubject(query string) (string, bool) {
	query = strings.TrimSpace(strings.Trim(query, "?! ."))
	lower := strings.ToLower(query)
	for _, prefix := range []string{"what version is ", "which version is ", "what version does ", "which version does "} {
		if strings.HasPrefix(lower, prefix) {
			subject := cleanFactObject(query[len(prefix):])
			return subject, subject != ""
		}
	}
	for _, prefix := range []string{"what ", "which "} {
		if strings.HasPrefix(lower, prefix) && strings.HasSuffix(lower, " version") {
			subject := cleanFactObject(query[len(prefix) : len(query)-len(" version")])
			return subject, subject != ""
		}
	}
	for _, prefix := range []string{"what is ", "what was ", "which is "} {
		if strings.HasPrefix(lower, prefix) && strings.HasSuffix(lower, " version") {
			subject := cleanFactObject(query[len(prefix) : len(query)-len(" version")])
			return subject, subject != ""
		}
	}
	return "", false
}

func searchVersionAnswerParts(sentence string) (subject, version string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	for _, pattern := range []*regexp.Regexp{searchVersionIsPattern, searchVersionShortPattern} {
		match := pattern.FindStringSubmatch(sentence)
		if len(match) != 3 {
			continue
		}
		subject = cleanFactObject(match[1])
		version = cleanFactValue(match[2])
		return subject, version, searchFactObjectLooksAssertive(subject) && searchVersionValueLooksAssertive(version)
	}
	return "", "", false
}

func searchSequenceFactIntentScore(query, content string) float64 {
	querySubject, ok := searchSequenceQuestionSubject(query)
	if !ok {
		return 0
	}
	queryTokens := searchRankTokenSet(querySubject)
	if len(queryTokens) == 0 {
		return 0
	}
	if !strings.Contains(content, "?") {
		subject, steps, ok := searchSequenceAnswerParts(content)
		if ok && searchSequenceValueLooksAssertive(steps) && searchRankTokenCoverage(queryTokens, subject) >= 0.80 {
			return 1
		}
	}
	for _, sentence := range recallSentencePattern.FindAllString(content, -1) {
		if strings.Contains(sentence, "?") {
			continue
		}
		subject, steps, ok := searchSequenceAnswerParts(sentence)
		if !ok || !searchSequenceValueLooksAssertive(steps) {
			continue
		}
		if searchRankTokenCoverage(queryTokens, subject) >= 0.80 {
			return 1
		}
	}
	return 0
}

func searchSequenceQuestionSubject(query string) (string, bool) {
	query = strings.TrimSpace(strings.Trim(query, "?!."))
	lower := strings.ToLower(query)
	for _, prefix := range []string{"walk me through the ", "walk me through ", "list the order of the ", "list the order of ", "what is the order of the ", "what is the order of ", "what was the order of the ", "what was the order of "} {
		if strings.HasPrefix(lower, prefix) {
			subject := cleanFactObject(query[len(prefix):])
			return subject, subject != ""
		}
	}
	if strings.HasPrefix(lower, "in what order did ") {
		subject := cleanFactObject(query[len("in what order did "):])
		return subject, subject != ""
	}
	return "", false
}

func searchSequenceAnswerParts(sentence string) (subject, steps string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", "", false
	}
	firstIdx := searchSequenceFirstMarkerIndex(sentence)
	if firstIdx <= 0 || searchSequenceMarkerCount(sentence) < 2 {
		return "", "", false
	}
	subject = sequenceSubjectBeforeMarker(sentence[:firstIdx])
	steps = cleanSequenceValue(sentence[firstIdx:])
	if subject != "" && searchSequenceValueLooksAssertive(steps) {
		return subject, steps, true
	}
	lower := strings.ToLower(sentence)
	for _, marker := range []string{" sequence is ", " order is "} {
		idx := strings.Index(lower, marker)
		if idx <= 0 {
			continue
		}
		subject = cleanFactObject(sentence[:idx] + strings.TrimSpace(marker))
		steps = cleanSequenceValue(sentence[idx+len(marker):])
		return subject, steps, searchFactObjectLooksAssertive(subject) && searchSequenceValueLooksAssertive(steps)
	}
	return "", "", false
}

func sequenceSubjectBeforeMarker(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.TrimSpace(strings.TrimRight(prefix, ":;"))
	if idx := strings.LastIndexAny(prefix, ":;"); idx >= 0 {
		prefix = strings.TrimSpace(prefix[idx+1:])
	}
	return cleanFactObject(prefix)
}

func searchSequenceFirstMarkerIndex(value string) int {
	lower := strings.ToLower(value)
	best := -1
	for _, marker := range searchSequenceMarkers {
		idx := strings.Index(lower, marker)
		if idx < 0 {
			continue
		}
		if best < 0 || idx < best {
			best = idx
		}
	}
	return best
}

func searchSequenceMarkerCount(value string) int {
	lower := strings.ToLower(value)
	count := 0
	for _, marker := range searchSequenceMarkers {
		if strings.Contains(lower, marker) {
			count++
		}
	}
	return count
}

func cleanSequenceValue(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "\"'`“”‘’")
	for _, sep := range []string{" because ", " but "} {
		idx := strings.Index(strings.ToLower(value), sep)
		if idx > 0 {
			value = strings.TrimSpace(value[:idx])
		}
	}
	return value
}

func searchNegationFactIntentScore(query, content string) float64 {
	queryObject, ok := searchNegationQuestionObject(query)
	if !ok {
		return 0
	}
	queryTokens := searchRankTokenSet(queryObject)
	if len(queryTokens) == 0 {
		return 0
	}
	if !strings.Contains(content, "?") {
		object, ok := searchNegationAnswerParts(content)
		if ok && searchRankTokenCoverage(queryTokens, object) >= 0.80 {
			return 1
		}
	}
	for _, sentence := range recallSentencePattern.FindAllString(content, -1) {
		if strings.Contains(sentence, "?") {
			continue
		}
		object, ok := searchNegationAnswerParts(sentence)
		if !ok {
			continue
		}
		if searchRankTokenCoverage(queryTokens, object) >= 0.80 {
			return 1
		}
	}
	return 0
}

func searchNegationQuestionObject(query string) (string, bool) {
	query = strings.TrimSpace(strings.Trim(query, "?!."))
	lower := strings.ToLower(query)
	for _, prefix := range []string{"have i ever ", "have i ", "did i ever ", "did i ", "have we ever ", "have we ", "did we ever ", "did we ", "has this ", "am i "} {
		if strings.HasPrefix(lower, prefix) {
			object := cleanFactObject(query[len(prefix):])
			return object, object != ""
		}
	}
	return "", false
}

func searchNegationAnswerParts(sentence string) (object string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	match := searchNegationPattern.FindStringSubmatch(sentence)
	if len(match) != 2 {
		return "", false
	}
	object = cleanFactValue(match[1])
	return object, searchFactObjectLooksAssertive(object)
}

func searchDecisionFactIntentScore(query, content string) float64 {
	queryTopic, ok := searchDecisionQuestionTopic(query)
	if !ok {
		return 0
	}
	queryTokens := searchRankTokenSet(queryTopic)
	if len(queryTokens) == 0 {
		return 0
	}
	if !strings.Contains(content, "?") {
		decision, ok := searchDecisionAnswerParts(content)
		if ok && searchRankTokenCoverage(queryTokens, decision) >= 0.80 {
			return 1
		}
	}
	for _, sentence := range recallSentencePattern.FindAllString(content, -1) {
		if strings.Contains(sentence, "?") {
			continue
		}
		decision, ok := searchDecisionAnswerParts(sentence)
		if !ok {
			continue
		}
		if searchRankTokenCoverage(queryTokens, decision) >= 0.80 {
			return 1
		}
	}
	return 0
}

func searchDecisionQuestionTopic(query string) (string, bool) {
	query = strings.TrimSpace(strings.Trim(query, "?!."))
	lower := strings.ToLower(query)
	for _, prefix := range []string{"what decision did i make about ", "which decision did i make about ", "what decision did we make about ", "which decision did we make about ", "what did i decide about ", "what did we decide about "} {
		if strings.HasPrefix(lower, prefix) {
			topic := cleanFactObject(query[len(prefix):])
			return topic, topic != ""
		}
	}
	return "", false
}

func searchDecisionAnswerParts(sentence string) (decision string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	match := searchDecisionPattern.FindStringSubmatch(sentence)
	if len(match) != 2 {
		return "", false
	}
	decision = cleanFactValue(match[1])
	return decision, searchFactObjectLooksAssertive(decision)
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

func cleanFactObject(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.LastIndexAny(value, ":;"); idx >= 0 {
		value = strings.TrimSpace(value[idx+1:])
	}
	value = strings.Trim(strings.TrimSpace(value), "\"'`“”‘’")
	for _, prefix := range []string{"the ", "a ", "an "} {
		if strings.HasPrefix(strings.ToLower(value), prefix) {
			value = strings.TrimSpace(value[len(prefix):])
		}
	}
	return value
}

func cleanFactValue(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "\"'`“”‘’")
	for _, sep := range []string{";", ",", " because ", " but ", " and "} {
		idx := strings.Index(strings.ToLower(value), sep)
		if idx > 0 {
			value = strings.TrimSpace(value[:idx])
		}
	}
	return value
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

func searchMetricValueLooksAssertive(value string) bool {
	return searchMetricValuePattern.MatchString(strings.TrimSpace(value))
}

func searchVersionValueLooksAssertive(value string) bool {
	return searchVersionValuePattern.MatchString(strings.TrimSpace(value))
}

func searchSequenceValueLooksAssertive(value string) bool {
	return searchSequenceMarkerCount(value) >= 2 && searchFactObjectLooksAssertive(value)
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
