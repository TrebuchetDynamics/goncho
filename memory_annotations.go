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

func conclusionFactAnnotations(content string) []string {
	seen := map[string]struct{}{}
	facts := []string{}
	for _, sentence := range recallSentencePattern.FindAllString(content, -1) {
		for _, extractor := range []func(string) (string, bool){conclusionOwnerFactAnnotation, conclusionPreferenceFactAnnotation} {
			fact, ok := extractor(sentence)
			if !ok {
				continue
			}
			key := strings.ToLower(fact)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			facts = append(facts, fact)
		}
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
		SELECT memory_id, value
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
		var id int64
		var value string
		if err := rows.Scan(&id, &value); err != nil {
			return nil, fmt.Errorf("goncho: scan conclusion fact annotation: %w", err)
		}
		for _, index := range indexes[id] {
			hits[index].factAnnotations = append(hits[index].factAnnotations, value)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate conclusion fact annotations: %w", err)
	}
	return hits, nil
}
