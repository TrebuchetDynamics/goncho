package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/memoryannotations"
)

const memoryAnnotationSourceConclusion = "conclusion"

var memoryAnnotationDDL = memoryannotations.DDL

type memoryFactAnnotation struct {
	ID           int64
	MemorySource string
	MemoryID     int64
	Value        string
	Source       string
	Confidence   float64
}

func conclusionFactAnnotations(content string) []string {
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
		for _, extractor := range []func(string) (string, bool){conclusionOwnerFactAnnotation, conclusionPreferenceFactAnnotation, conclusionLocationFactAnnotation, conclusionInstructionFactAnnotation, conclusionTimelineFactAnnotation, conclusionMetricFactAnnotation, conclusionVersionFactAnnotation, conclusionSequenceFactAnnotation, conclusionNegationFactAnnotation, conclusionDecisionFactAnnotation} {
			addFact(extractor(sentence))
		}
	}
	for _, extractor := range []func(string) (string, bool){conclusionMetricFactAnnotation, conclusionVersionFactAnnotation} {
		addFact(extractor(content))
	}
	return facts
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

func storeConclusionFactAnnotations(ctx context.Context, db *sql.DB, workspaceID, profileID, observer, peer string, conclusionID int64, facts []string) error {
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
		`, workspaceID, profileID, observer, peer, memoryAnnotationSourceConclusion, conclusionID, fact, now); err != nil {
			return fmt.Errorf("goncho: store conclusion fact annotation: %w", err)
		}
	}
	return nil
}

func attachConclusionFactAnnotations(ctx context.Context, db *sql.DB, hits []SearchHit) ([]SearchHit, error) {
	if len(hits) == 0 {
		return hits, nil
	}
	present, err := sqliteTableExists(ctx, db, "goncho_memory_annotations")
	if err != nil {
		return nil, err
	}
	if !present {
		return hits, nil
	}
	ids := make([]string, 0, len(hits))
	indexes := map[int64][]int{}
	for i, hit := range hits {
		if hit.Source != memoryAnnotationSourceConclusion || hit.ID <= 0 {
			continue
		}
		if _, ok := indexes[hit.ID]; !ok {
			ids = append(ids, fmt.Sprint(hit.ID))
		}
		indexes[hit.ID] = append(indexes[hit.ID], i)
	}
	if len(ids) == 0 {
		return hits, nil
	}

	var b strings.Builder
	b.WriteString(`
		SELECT id, memory_source, memory_id, value, source, confidence
		FROM goncho_memory_annotations
		WHERE memory_source = 'conclusion'
		  AND kind = 'fact'
		  AND `)
	args := []any{}
	appendInClause(&b, "memory_id", ids, &args)
	b.WriteString(`
		ORDER BY id ASC
	`)
	rows, err := db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("goncho: query conclusion fact annotations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var annotation memoryFactAnnotation
		if err := rows.Scan(&annotation.ID, &annotation.MemorySource, &annotation.MemoryID, &annotation.Value, &annotation.Source, &annotation.Confidence); err != nil {
			return nil, fmt.Errorf("goncho: scan conclusion fact annotation: %w", err)
		}
		for _, index := range indexes[annotation.MemoryID] {
			hits[index].factAnnotations = append(hits[index].factAnnotations, annotation)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate conclusion fact annotations: %w", err)
	}
	return hits, nil
}
