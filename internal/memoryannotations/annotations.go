package memoryannotations

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

const SourceConclusion = "conclusion"

const searchMetricUnitPattern = `ms|sec|seconds?|minutes?|hours?|days?|weeks?|months?|%|kb|mb|gb|tb|rows?|columns?|roles?|features?|bugs?|commits?|cards?|users?|items?|tests?|apis?|endpoints?|tickets?`

const (
	kgRelationUses      = "uses"
	kgRelationDependsOn = "depends_on"
	kgRelationRunsOn    = "runs_on"
)

var (
	searchMetricValuePattern  = regexp.MustCompile(`(?i)^\d+(?:[.,]\d+)?\s*(?:` + searchMetricUnitPattern + `)\s*$`)
	searchMetricAnswerPattern = regexp.MustCompile(`(?i)^\s*(.+?)\s+(?:is|was|=)\s+(\d+(?:[.,]\d+)?\s*(?:` + searchMetricUnitPattern + `))\s*$`)
	searchVersionValuePattern = regexp.MustCompile(`(?i)^v?\d+\.\d+(?:\.\d+)?\s*$`)
	searchVersionIsPattern    = regexp.MustCompile(`(?i)^\s*(.+?)\s+version\s+(?:is|was|=)\s+(v?\d+\.\d+(?:\.\d+)?)\s*$`)
	searchVersionShortPattern = regexp.MustCompile(`(?i)^\s*(.+?)\s+v(\d+\.\d+(?:\.\d+)?)\s*$`)
	searchNegationPattern     = regexp.MustCompile(`(?i)^\s*(?:project note:\s*)?(?:i|we|user)\s+(?:(?:have|has|had|did)\s+)?(?:never|not)\s+(.+?)\s*$`)
	searchDecisionPattern     = regexp.MustCompile(`(?i)^\s*(?:project note:\s*)?(?:i|we|user)\s+(?:decided to|chose to|opted for|selected|picked|switching to)\s+(.+?)\s*$`)
	searchSequenceMarkers     = []string{"first", "second", "third", "fourth", "fifth", "finally", "next", "then", "after that"}
	recallSentencePattern     = regexp.MustCompile(`[^.!?]+[.!?]?`)
	searchRankTokenPattern    = regexp.MustCompile(`[a-z0-9]+`)
)

var DDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_memory_annotations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id TEXT NOT NULL,
		profile_id TEXT NOT NULL DEFAULT '',
		observer_peer_id TEXT NOT NULL,
		peer_id TEXT NOT NULL,
		memory_source TEXT NOT NULL,
		memory_id INTEGER NOT NULL,
		kind TEXT NOT NULL,
		value TEXT NOT NULL,
		source TEXT NOT NULL DEFAULT '',
		confidence REAL NOT NULL DEFAULT 1.0,
		created_at INTEGER NOT NULL,
		FOREIGN KEY(memory_id) REFERENCES goncho_conclusions(id) ON DELETE CASCADE,
		UNIQUE(workspace_id, profile_id, observer_peer_id, peer_id, memory_source, memory_id, kind, value)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_memory_annotations_memory_kind ON goncho_memory_annotations(workspace_id, profile_id, observer_peer_id, peer_id, memory_source, memory_id, kind)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_memory_annotations_kind_value ON goncho_memory_annotations(kind, value)`,
}

type FactAnnotation struct {
	ID           int64
	MemorySource string
	MemoryID     int64
	Value        string
	Source       string
	Confidence   float64
}

func ConclusionFacts(content string) []string {
	seen := map[string]struct{}{}
	facts := []string{}
	addFact := func(fact string, ok bool) {
		if !ok {
			return
		}
		key := strings.ToLower(fact)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		facts = append(facts, fact)
	}
	for _, sentence := range recallSentencePattern.FindAllString(content, -1) {
		for _, extractor := range []func(string) (string, bool){conclusionOwnerFactAnnotation, conclusionPreferenceFactAnnotation, conclusionLocationFactAnnotation, conclusionInstructionFactAnnotation, conclusionTimelineFactAnnotation, conclusionMetricFactAnnotation, conclusionVersionFactAnnotation, conclusionSequenceFactAnnotation, conclusionNegationFactAnnotation, conclusionDecisionFactAnnotation, conclusionKGRelationFactAnnotation} {
			addFact(extractor(sentence))
		}
	}
	for _, extractor := range []func(string) (string, bool){conclusionMetricFactAnnotation, conclusionVersionFactAnnotation} {
		addFact(extractor(content))
	}
	return facts
}

func StoreConclusionFacts(ctx context.Context, db *sql.DB, workspaceID, profileID, observer, peer string, conclusionID int64, facts []string) error {
	if len(facts) == 0 {
		return nil
	}
	now := time.Now().Unix()
	for _, fact := range facts {
		fact = strings.TrimSpace(fact)
		if fact == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, `
			INSERT OR IGNORE INTO goncho_memory_annotations(
				workspace_id, profile_id, observer_peer_id, peer_id, memory_source, memory_id,
				kind, value, source, confidence, created_at
			)
			VALUES(?, ?, ?, ?, ?, ?, 'fact', ?, 'deterministic_owner_extractor', 0.8, ?)
		`, workspaceID, profileID, observer, peer, SourceConclusion, conclusionID, fact, now); err != nil {
			return fmt.Errorf("goncho: store conclusion fact annotation: %w", err)
		}
	}
	return nil
}

func ConclusionFactsByMemoryID(ctx context.Context, db *sql.DB, ids []int64) (map[int64][]FactAnnotation, error) {
	out := map[int64][]FactAnnotation{}
	if len(ids) == 0 {
		return out, nil
	}
	present, err := sqliteTableExists(ctx, db, "goncho_memory_annotations")
	if err != nil {
		return nil, err
	}
	if !present {
		return out, nil
	}
	idStrings := make([]string, 0, len(ids))
	seen := map[int64]struct{}{}
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		idStrings = append(idStrings, fmt.Sprint(id))
	}
	if len(idStrings) == 0 {
		return out, nil
	}

	var b strings.Builder
	b.WriteString(`
		SELECT id, memory_source, memory_id, value, source, confidence
		FROM goncho_memory_annotations
		WHERE memory_source = 'conclusion'
		  AND kind = 'fact'
		  AND `)
	args := []any{}
	appendInClause(&b, "memory_id", idStrings, &args)
	b.WriteString(`
		ORDER BY id ASC
	`)
	rows, err := db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("goncho: query conclusion fact annotations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var annotation FactAnnotation
		if err := rows.Scan(&annotation.ID, &annotation.MemorySource, &annotation.MemoryID, &annotation.Value, &annotation.Source, &annotation.Confidence); err != nil {
			return nil, fmt.Errorf("goncho: scan conclusion fact annotation: %w", err)
		}
		out[annotation.MemoryID] = append(out[annotation.MemoryID], annotation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate conclusion fact annotations: %w", err)
	}
	return out, nil
}

func conclusionOwnerFactAnnotation(sentence string) (string, bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", false
	}
	lower := strings.ToLower(sentence)
	if fact, ok := conclusionPossessiveOwnerFact(sentence, lower); ok {
		return fact, true
	}
	return conclusionOwnerOfFact(sentence, lower)
}

func conclusionPossessiveOwnerFact(sentence, lower string) (string, bool) {
	marker := "'s owner is "
	idx := strings.Index(lower, marker)
	if idx < 0 {
		return "", false
	}
	object := cleanFactObject(sentence[:idx])
	owner := cleanFactValue(sentence[idx+len(marker):])
	return ownerFactAnnotation(owner, object)
}

func conclusionOwnerOfFact(sentence, lower string) (string, bool) {
	prefix := "owner of "
	idx := strings.Index(lower, prefix)
	if idx < 0 {
		return "", false
	}
	rest := sentence[idx+len(prefix):]
	restLower := strings.ToLower(rest)
	isIdx := strings.Index(restLower, " is ")
	if isIdx < 0 {
		return "", false
	}
	object := cleanFactObject(rest[:isIdx])
	owner := cleanFactValue(rest[isIdx+len(" is "):])
	return ownerFactAnnotation(owner, object)
}

func conclusionPreferenceFactAnnotation(sentence string) (string, bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", false
	}
	lower := strings.ToLower(sentence)
	possessive := "'s "
	idx := strings.Index(lower, possessive)
	if idx < 0 {
		return "", false
	}
	subject := cleanFactObject(sentence[:idx])
	rest := sentence[idx+len(possessive):]
	restLower := strings.ToLower(rest)
	marker := " preference is "
	prefIdx := strings.Index(restLower, marker)
	if prefIdx < 0 {
		return "", false
	}
	attribute := cleanFactObject(rest[:prefIdx])
	value := cleanFactValue(rest[prefIdx+len(marker):])
	return preferenceFactAnnotation(subject, value, attribute)
}

func conclusionLocationFactAnnotation(sentence string) (string, bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", false
	}
	lower := strings.ToLower(sentence)
	for _, marker := range []string{" location is ", " location: "} {
		idx := strings.Index(lower, marker)
		if idx <= 0 {
			continue
		}
		object := cleanFactObject(sentence[:idx])
		location := cleanFactValue(sentence[idx+len(marker):])
		return locationFactAnnotation(object, location)
	}
	for _, marker := range []string{" is located at ", " is located in ", " is in ", " lives in "} {
		idx := strings.Index(lower, marker)
		if idx <= 0 {
			continue
		}
		object := cleanFactObject(sentence[:idx])
		location := cleanFactValue(sentence[idx+len(marker):])
		return locationFactAnnotation(object, location)
	}
	return "", false
}

func conclusionInstructionFactAnnotation(sentence string) (string, bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", false
	}
	lower := strings.ToLower(sentence)
	possessive := "'s "
	idx := strings.Index(lower, possessive)
	if idx < 0 {
		return "", false
	}
	subject := cleanFactObject(sentence[:idx])
	rest := sentence[idx+len(possessive):]
	restLower := strings.ToLower(rest)
	for _, marker := range []string{"instruction is ", "rule is ", " instruction is ", " rule is "} {
		instructionIdx := strings.Index(restLower, marker)
		if instructionIdx < 0 {
			continue
		}
		instruction := cleanFactValue(rest[instructionIdx+len(marker):])
		return instructionFactAnnotation(subject, instruction)
	}
	return "", false
}

func conclusionTimelineFactAnnotation(sentence string) (string, bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", false
	}
	lower := strings.ToLower(sentence)
	for _, marker := range []string{" deadline is ", " is scheduled for ", " occurs on ", " is on "} {
		idx := strings.Index(lower, marker)
		if idx <= 0 {
			continue
		}
		event := cleanFactObject(sentence[:idx])
		date := cleanFactValue(sentence[idx+len(marker):])
		return timelineFactAnnotation(event, date)
	}
	return "", false
}

func conclusionMetricFactAnnotation(sentence string) (string, bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", false
	}
	key, value, ok := searchMetricAnswerParts(sentence)
	if !ok {
		return "", false
	}
	return metricFactAnnotation(key, value)
}

func conclusionVersionFactAnnotation(sentence string) (string, bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", false
	}
	subject, version, ok := searchVersionAnswerParts(sentence)
	if !ok {
		return "", false
	}
	return versionFactAnnotation(subject, version)
}

func conclusionSequenceFactAnnotation(sentence string) (string, bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", false
	}
	subject, steps, ok := searchSequenceAnswerParts(sentence)
	if !ok {
		return "", false
	}
	return sequenceFactAnnotation(subject, steps)
}

func conclusionKGRelationFactAnnotation(sentence string) (string, bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", false
	}
	subject, relation, object, ok := kgRelationAnswerParts(sentence)
	if !ok {
		return "", false
	}
	return kgRelationFactAnnotation(subject, relation, object)
}

func conclusionNegationFactAnnotation(sentence string) (string, bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", false
	}
	object, ok := searchNegationAnswerParts(sentence)
	if !ok {
		return "", false
	}
	return negationFactAnnotation(object)
}

func conclusionDecisionFactAnnotation(sentence string) (string, bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", false
	}
	decision, ok := searchDecisionAnswerParts(sentence)
	if !ok {
		return "", false
	}
	return decisionFactAnnotation(decision)
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

func ownerFactAnnotation(owner, object string) (string, bool) {
	owner = cleanFactValue(owner)
	object = cleanFactObject(object)
	if !searchFactSubjectLooksAssertive(owner) || !searchFactObjectLooksAssertive(object) {
		return "", false
	}
	if searchRankTokenCoverage(searchRankTokenSet(object), owner) > 0 {
		return "", false
	}
	return fmt.Sprintf("%s owns %s", owner, object), true
}

func preferenceFactAnnotation(subject, value, attribute string) (string, bool) {
	subject = cleanFactValue(subject)
	value = cleanFactValue(value)
	attribute = cleanFactObject(attribute)
	if !searchFactSubjectLooksAssertive(subject) || !searchFactObjectLooksAssertive(value) || !searchFactObjectLooksAssertive(attribute) {
		return "", false
	}
	return fmt.Sprintf("%s prefers %s for %s", subject, value, attribute), true
}

func locationFactAnnotation(object, location string) (string, bool) {
	object = cleanFactObject(object)
	location = cleanFactValue(location)
	if !searchFactObjectLooksAssertive(object) || !searchFactObjectLooksAssertive(location) {
		return "", false
	}
	return fmt.Sprintf("%s is located at %s", object, location), true
}

func instructionFactAnnotation(subject, instruction string) (string, bool) {
	subject = cleanFactValue(subject)
	instruction = cleanFactValue(instruction)
	if !searchFactSubjectLooksAssertive(subject) || !searchFactObjectLooksAssertive(instruction) {
		return "", false
	}
	return fmt.Sprintf("%s instructed %s", subject, instruction), true
}

func timelineFactAnnotation(event, date string) (string, bool) {
	event = cleanFactObject(event)
	date = cleanFactValue(date)
	if !searchFactObjectLooksAssertive(event) || !searchFactObjectLooksAssertive(date) {
		return "", false
	}
	return fmt.Sprintf("%s occurs on %s", event, date), true
}

func metricFactAnnotation(key, value string) (string, bool) {
	key = cleanFactObject(key)
	value = cleanFactValue(value)
	if !searchFactObjectLooksAssertive(key) || !searchMetricValueLooksAssertive(value) {
		return "", false
	}
	return fmt.Sprintf("%s is %s", key, value), true
}

func versionFactAnnotation(subject, version string) (string, bool) {
	subject = cleanFactObject(subject)
	version = cleanFactValue(version)
	if !searchFactObjectLooksAssertive(subject) || !searchVersionValueLooksAssertive(version) {
		return "", false
	}
	return fmt.Sprintf("%s version is %s", subject, version), true
}

func sequenceFactAnnotation(subject, steps string) (string, bool) {
	subject = cleanFactObject(subject)
	steps = cleanSequenceValue(steps)
	if !searchFactObjectLooksAssertive(subject) || !searchSequenceValueLooksAssertive(steps) {
		return "", false
	}
	return fmt.Sprintf("%s is %s", subject, steps), true
}

func kgRelationFactAnnotation(subject, relation, object string) (string, bool) {
	subject = cleanFactObject(subject)
	object = cleanFactValue(object)
	if !searchFactObjectLooksAssertive(subject) || !searchFactObjectLooksAssertive(object) {
		return "", false
	}
	phrase := kgRelationPhrase(relation)
	if phrase == "" {
		return "", false
	}
	return fmt.Sprintf("%s %s %s", subject, phrase, object), true
}

func negationFactAnnotation(object string) (string, bool) {
	object = cleanFactValue(object)
	if !searchFactObjectLooksAssertive(object) {
		return "", false
	}
	return fmt.Sprintf("user never %s", object), true
}

func decisionFactAnnotation(decision string) (string, bool) {
	decision = cleanFactValue(decision)
	if !searchFactObjectLooksAssertive(decision) {
		return "", false
	}
	return fmt.Sprintf("user decided to %s", decision), true
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

func searchNegationAnswerParts(sentence string) (object string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	match := searchNegationPattern.FindStringSubmatch(sentence)
	if len(match) != 2 {
		return "", false
	}
	object = cleanFactValue(match[1])
	return object, searchFactObjectLooksAssertive(object)
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

func kgRelationAnswerParts(sentence string) (subject, relation, object string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", "", "", false
	}
	lower := strings.ToLower(sentence)
	for _, marker := range []struct {
		text     string
		relation string
	}{
		{text: " depends on ", relation: kgRelationDependsOn},
		{text: " runs on ", relation: kgRelationRunsOn},
		{text: " uses ", relation: kgRelationUses},
	} {
		idx := strings.Index(lower, marker.text)
		if idx <= 0 {
			continue
		}
		subject = cleanFactObject(sentence[:idx])
		object = cleanFactValue(sentence[idx+len(marker.text):])
		if searchFactObjectLooksAssertive(subject) && searchFactObjectLooksAssertive(object) {
			return subject, marker.relation, object, true
		}
	}
	return "", "", "", false
}

func kgRelationPhrase(relation string) string {
	switch relation {
	case kgRelationUses:
		return "uses"
	case kgRelationDependsOn:
		return "depends on"
	case kgRelationRunsOn:
		return "runs on"
	default:
		return ""
	}
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

func sqliteTableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&count); err != nil {
		return false, fmt.Errorf("goncho: check sqlite table %s: %w", name, err)
	}
	return count > 0, nil
}

func appendInClause(b *strings.Builder, column string, values []string, args *[]any) {
	b.WriteString(column)
	b.WriteString(" IN (")
	for i, value := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("?")
		*args = append(*args, value)
	}
	b.WriteString(")")
}
