package goncho

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

const (
	kgRelationUses      = "uses"
	kgRelationDependsOn = "depends_on"
	kgRelationRunsOn    = "runs_on"
)

type annotationGraphOwnerTarget struct {
	Candidate RecallCandidate
	Fact      memoryFactAnnotation
}

func (r retrievalModule) expandAnnotationGraphRecall(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope string, base []RecallCandidate) ([]RecallCandidate, error) {
	if len(base) == 0 || !annotationGraphOwnerQuery(q.Query) {
		return base, nil
	}
	out := make([]RecallCandidate, len(base))
	copy(out, base)
	indexByID := make(map[string]int, len(out))
	for i, candidate := range out {
		indexByID[candidate.MemoryID] = i
	}
	for _, source := range base {
		for _, evidence := range source.Provenance {
			if evidence.Kind != "fact" || evidence.Source != "goncho_memory_annotations" {
				continue
			}
			fact := strings.TrimPrefix(evidence.Note, "fact=")
			subject, relation, entity, ok := kgRelationAnswerParts(fact)
			if !ok || !annotationGraphQueryMatchesKGRelation(q.Query, subject, relation) {
				continue
			}
			targets, err := r.findAnnotationGraphOwnerTargets(ctx, q, workspaceID, peer, memoryScope, entity, source.MemoryID)
			if err != nil {
				return nil, err
			}
			for _, target := range targets {
				graphEvidence := annotationGraphEvidence(source.MemoryID, target.Candidate.MemoryID, relation, entity, evidence.ID, target.Fact.ID)
				if idx, exists := indexByID[target.Candidate.MemoryID]; exists {
					if !recallCandidateHasEvidence(out[idx], graphEvidence.Kind, graphEvidence.ID) {
						out[idx].Provenance = append(out[idx].Provenance, graphEvidence)
					}
					continue
				}
				target.Candidate.Provenance = append(target.Candidate.Provenance, graphEvidence)
				out = append(out, target.Candidate)
				indexByID[target.Candidate.MemoryID] = len(out) - 1
			}
		}
	}
	return out, nil
}

func (r retrievalModule) findAnnotationGraphOwnerTargets(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope, entity, sourceMemoryID string) ([]annotationGraphOwnerTarget, error) {
	present, err := sqliteTableExists(ctx, r.db, "goncho_memory_annotations")
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, nil
	}

	query := `
		SELECT a.id, a.memory_source, a.memory_id, a.value, a.source, a.confidence,
		       c.content, COALESCE(c.session_key, '')
		FROM goncho_memory_annotations a
		JOIN goncho_conclusions c ON c.id = a.memory_id
		WHERE a.workspace_id = ?
		  AND a.profile_id = ''
		  AND a.observer_peer_id = ?
		  AND a.peer_id = ?
		  AND a.memory_source = 'conclusion'
		  AND a.kind = 'fact'
	`
	args := []any{workspaceID, r.observer, peer}
	switch normalizeMemoryScope(memoryScope, "") {
	case MemoryScopeWorkspace:
		query += ` AND ((c.workspace_id = ? AND c.scope = 'workspace') OR c.scope = 'global')`
		args = append(args, workspaceID)
	case MemoryScopeShared:
		query += ` AND c.workspace_id = ? AND c.scope = 'shared'`
		args = append(args, workspaceID)
	case MemoryScopeSession:
		query += ` AND c.workspace_id = ? AND c.profile_id = '' AND c.scope = 'session'`
		args = append(args, workspaceID)
	case MemoryScopeGlobal:
		query += ` AND c.scope = 'global'`
	case MemoryScopeProfile:
		query += ` AND c.workspace_id = ? AND c.profile_id = '' AND c.scope = 'profile'`
		args = append(args, workspaceID)
	}
	if sessionKey := strings.TrimSpace(q.SessionKey); sessionKey != "" {
		query += ` AND (c.session_key = ? OR c.session_key IS NULL)`
		args = append(args, sessionKey)
	}
	query += ` ORDER BY a.id ASC LIMIT 200`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("goncho: query annotation graph owner targets: %w", err)
	}
	defer rows.Close()

	out := []annotationGraphOwnerTarget{}
	for rows.Next() {
		var fact memoryFactAnnotation
		var content, sessionKey string
		if err := rows.Scan(&fact.ID, &fact.MemorySource, &fact.MemoryID, &fact.Value, &fact.Source, &fact.Confidence, &content, &sessionKey); err != nil {
			return nil, fmt.Errorf("goncho: scan annotation graph owner target: %w", err)
		}
		memoryID := strconv.FormatInt(fact.MemoryID, 10)
		if memoryID == sourceMemoryID {
			continue
		}
		_, owned, ok := annotationGraphOwnerFactParts(fact.Value)
		if !ok || !annotationGraphEntityMatches(entity, owned) {
			continue
		}
		candidate := RecallCandidate{
			MemoryID:   memoryID,
			SourceType: memoryAnnotationSourceConclusion,
			Content:    content,
			SessionID:  sessionKey,
			AgentID:    r.observer,
			ScopeID:    normalizeMemoryScope(memoryScope, ""),
			Provenance: []EvidenceItem{annotationFactEvidence(q.Query, fact)},
		}
		out = append(out, annotationGraphOwnerTarget{Candidate: candidate, Fact: fact})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate annotation graph owner targets: %w", err)
	}
	return out, nil
}

func annotationFactEvidence(query string, fact memoryFactAnnotation) EvidenceItem {
	return EvidenceItem{
		Kind:   "fact",
		Source: "goncho_memory_annotations",
		ID:     strconv.FormatInt(fact.ID, 10),
		Score:  roundRecallFloat(searchFactIntentScore(query, fact.Value)),
		Note:   "fact=" + strings.TrimSpace(fact.Value),
		Metadata: map[string]string{
			"memory_source": fact.MemorySource,
			"memory_id":     strconv.FormatInt(fact.MemoryID, 10),
			"source":        fact.Source,
			"confidence":    fmt.Sprintf("%.3f", fact.Confidence),
		},
	}
}

func annotationGraphEvidence(sourceMemoryID, targetMemoryID, relation, entity, sourceFactID string, targetFactID int64) EvidenceItem {
	targetFactIDText := strconv.FormatInt(targetFactID, 10)
	return EvidenceItem{
		Kind:   "graph",
		Source: sourceMemoryID,
		ID:     "annotation:" + sourceFactID + "->annotation:" + targetFactIDText,
		Score:  1,
		Note:   sourceMemoryID + " -> " + kgRelationPhrase(relation) + " -> " + entity + " -> owned_by -> " + targetMemoryID,
		Metadata: map[string]string{
			"entity":          entity,
			"relation":        relation,
			"source_fact_id":  sourceFactID,
			"target_fact_id":  targetFactIDText,
			"target_relation": "owned_by",
		},
	}
}

func recallCandidateHasEvidence(candidate RecallCandidate, kind, id string) bool {
	for _, item := range candidate.Provenance {
		if item.Kind == kind && item.ID == id {
			return true
		}
	}
	return false
}

func annotationGraphOwnerQuery(query string) bool {
	query = strings.ToLower(query)
	if !(strings.Contains(query, "owner") || strings.Contains(query, "owns") || strings.Contains(query, "responsible") || strings.Contains(query, "accountable")) {
		return false
	}
	return strings.Contains(query, "who") || strings.Contains(query, "which") || strings.Contains(query, "what")
}

func annotationGraphQueryMatchesKGRelation(query, subject, relation string) bool {
	subjectTokens := searchRankTokenSet(subject)
	if len(subjectTokens) == 0 || searchRankTokenCoverage(subjectTokens, query) < 0.80 {
		return false
	}
	query = strings.ToLower(query)
	switch relation {
	case kgRelationUses:
		return strings.Contains(query, "use") || strings.Contains(query, "used") || strings.Contains(query, "using")
	case kgRelationDependsOn:
		return strings.Contains(query, "depend") || strings.Contains(query, "dependency")
	case kgRelationRunsOn:
		return strings.Contains(query, "runs on") || strings.Contains(query, "running on")
	default:
		return false
	}
}

func annotationGraphEntityMatches(a, b string) bool {
	a = cleanFactObject(a)
	b = cleanFactObject(b)
	if strings.EqualFold(a, b) {
		return true
	}
	aTokens := searchRankTokenSet(a)
	bTokens := searchRankTokenSet(b)
	return len(aTokens) > 0 && searchRankTokenCoverage(aTokens, b) >= 0.80 && searchRankTokenCoverage(bTokens, a) >= 0.80
}

func annotationGraphOwnerFactParts(fact string) (owner, entity string, ok bool) {
	fact = strings.TrimSpace(strings.Trim(fact, ".!?"))
	lower := strings.ToLower(fact)
	idx := strings.Index(lower, " owns ")
	if idx <= 0 {
		return "", "", false
	}
	owner = cleanFactValue(fact[:idx])
	entity = cleanFactObject(fact[idx+len(" owns "):])
	return owner, entity, searchFactSubjectLooksAssertive(owner) && searchFactObjectLooksAssertive(entity)
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
