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

type annotationGraphVersionTarget struct {
	Candidate    RecallCandidate
	RelationFact memoryFactAnnotation
	VersionFact  memoryFactAnnotation
	Relation     string
	Entity       string
}

type annotationGraphTimelineTarget struct {
	Candidate    RecallCandidate
	TimelineFact memoryFactAnnotation
	Entity       string
}

type annotationGraphMetricTarget struct {
	Candidate  RecallCandidate
	MetricFact memoryFactAnnotation
	Entity     string
}

type annotationGraphLocationTarget struct {
	Candidate    RecallCandidate
	LocationFact memoryFactAnnotation
	Entity       string
}

func (r retrievalModule) expandAnnotationGraphRecall(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope string, base []RecallCandidate) ([]RecallCandidate, error) {
	ownerQuery := annotationGraphOwnerQuery(q.Query)
	versionQuery := annotationGraphVersionQuery(q.Query)
	timelineQuery := annotationGraphTimelineQuery(q.Query)
	metricQuery := annotationGraphMetricQuery(q.Query)
	locationQuery := annotationGraphLocationQuery(q.Query)
	if len(base) == 0 || (!ownerQuery && !versionQuery && !timelineQuery && !metricQuery && !locationQuery) {
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
			if timelineQuery {
				owner, entity, ok := annotationGraphOwnerFactParts(fact)
				if ok && annotationGraphQueryMatchesOwnerFact(q.Query, owner) {
					targets, err := r.findAnnotationGraphTimelineTargets(ctx, q, workspaceID, peer, memoryScope, entity, source.MemoryID)
					if err != nil {
						return nil, err
					}
					for _, target := range targets {
						graphEvidence := annotationGraphTimelineEvidence(source.MemoryID, target.Candidate.MemoryID, entity, evidence.ID, target.TimelineFact.ID)
						out = appendAnnotationGraphCandidate(out, indexByID, target.Candidate, graphEvidence)
					}
				}
			}
			subject, relation, entity, ok := kgRelationAnswerParts(fact)
			if !ok || !annotationGraphQueryMatchesKGRelation(q.Query, subject, relation) {
				continue
			}
			if ownerQuery {
				targets, err := r.findAnnotationGraphOwnerTargets(ctx, q, workspaceID, peer, memoryScope, entity, source.MemoryID)
				if err != nil {
					return nil, err
				}
				for _, target := range targets {
					graphEvidence := annotationGraphEvidence(source.MemoryID, target.Candidate.MemoryID, relation, entity, evidence.ID, target.Fact.ID)
					out = appendAnnotationGraphCandidate(out, indexByID, target.Candidate, graphEvidence)
				}
			}
			if versionQuery {
				targets, err := r.findAnnotationGraphVersionTargets(ctx, q, workspaceID, peer, memoryScope, entity, source.MemoryID)
				if err != nil {
					return nil, err
				}
				for _, target := range targets {
					graphEvidence := annotationGraphVersionEvidence(source.MemoryID, target.Candidate.MemoryID, relation, entity, target.Relation, target.Entity, evidence.ID, target.RelationFact.ID, target.VersionFact.ID)
					out = appendAnnotationGraphCandidate(out, indexByID, target.Candidate, graphEvidence)
				}
			}
			if metricQuery {
				targets, err := r.findAnnotationGraphMetricTargets(ctx, q, workspaceID, peer, memoryScope, entity, source.MemoryID)
				if err != nil {
					return nil, err
				}
				for _, target := range targets {
					graphEvidence := annotationGraphMetricEvidence(source.MemoryID, target.Candidate.MemoryID, relation, entity, target.Entity, evidence.ID, target.MetricFact.ID)
					out = appendAnnotationGraphCandidate(out, indexByID, target.Candidate, graphEvidence)
				}
			}
			if locationQuery {
				targets, err := r.findAnnotationGraphLocationTargets(ctx, q, workspaceID, peer, memoryScope, entity, source.MemoryID)
				if err != nil {
					return nil, err
				}
				for _, target := range targets {
					graphEvidence := annotationGraphLocationEvidence(source.MemoryID, target.Candidate.MemoryID, relation, entity, target.Entity, evidence.ID, target.LocationFact.ID)
					out = appendAnnotationGraphCandidate(out, indexByID, target.Candidate, graphEvidence)
				}
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

func (r retrievalModule) findAnnotationGraphTimelineTargets(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope, entity, sourceMemoryID string) ([]annotationGraphTimelineTarget, error) {
	facts, err := r.queryAnnotationGraphFacts(ctx, workspaceID, peer, memoryScope, q.SessionKey)
	if err != nil {
		return nil, err
	}
	out := []annotationGraphTimelineTarget{}
	for _, timelineFact := range facts {
		memoryID := strconv.FormatInt(timelineFact.MemoryID, 10)
		if memoryID == sourceMemoryID {
			continue
		}
		event, _, ok := searchTimelineAnswerParts(timelineFact.Value)
		if !ok || !annotationGraphEntityMatches(entity, event) {
			continue
		}
		candidate := RecallCandidate{
			MemoryID:   memoryID,
			SourceType: memoryAnnotationSourceConclusion,
			Content:    timelineFact.Content,
			SessionID:  timelineFact.SessionKey,
			AgentID:    r.observer,
			ScopeID:    normalizeMemoryScope(memoryScope, ""),
			Provenance: []EvidenceItem{annotationFactEvidence(q.Query, timelineFact.memoryFactAnnotation)},
		}
		out = append(out, annotationGraphTimelineTarget{Candidate: candidate, TimelineFact: timelineFact.memoryFactAnnotation, Entity: event})
	}
	return out, nil
}

func (r retrievalModule) findAnnotationGraphLocationTargets(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope, entity, sourceMemoryID string) ([]annotationGraphLocationTarget, error) {
	facts, err := r.queryAnnotationGraphFacts(ctx, workspaceID, peer, memoryScope, q.SessionKey)
	if err != nil {
		return nil, err
	}
	out := []annotationGraphLocationTarget{}
	for _, locationFact := range facts {
		memoryID := strconv.FormatInt(locationFact.MemoryID, 10)
		if memoryID == sourceMemoryID {
			continue
		}
		object, _, ok := searchLocationAnswerParts(locationFact.Value)
		if !ok || !annotationGraphEntityMatches(entity, object) {
			continue
		}
		candidate := RecallCandidate{
			MemoryID:   memoryID,
			SourceType: memoryAnnotationSourceConclusion,
			Content:    locationFact.Content,
			SessionID:  locationFact.SessionKey,
			AgentID:    r.observer,
			ScopeID:    normalizeMemoryScope(memoryScope, ""),
			Provenance: []EvidenceItem{annotationFactEvidence(q.Query, locationFact.memoryFactAnnotation)},
		}
		out = append(out, annotationGraphLocationTarget{Candidate: candidate, LocationFact: locationFact.memoryFactAnnotation, Entity: object})
	}
	return out, nil
}

func (r retrievalModule) findAnnotationGraphMetricTargets(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope, entity, sourceMemoryID string) ([]annotationGraphMetricTarget, error) {
	facts, err := r.queryAnnotationGraphFacts(ctx, workspaceID, peer, memoryScope, q.SessionKey)
	if err != nil {
		return nil, err
	}
	out := []annotationGraphMetricTarget{}
	for _, metricFact := range facts {
		memoryID := strconv.FormatInt(metricFact.MemoryID, 10)
		if memoryID == sourceMemoryID {
			continue
		}
		key, _, ok := searchMetricAnswerParts(metricFact.Value)
		if !ok || !annotationGraphEntityMentionedInFact(entity, key) {
			continue
		}
		candidate := RecallCandidate{
			MemoryID:   memoryID,
			SourceType: memoryAnnotationSourceConclusion,
			Content:    metricFact.Content,
			SessionID:  metricFact.SessionKey,
			AgentID:    r.observer,
			ScopeID:    normalizeMemoryScope(memoryScope, ""),
			Provenance: []EvidenceItem{annotationFactEvidence(q.Query, metricFact.memoryFactAnnotation)},
		}
		out = append(out, annotationGraphMetricTarget{Candidate: candidate, MetricFact: metricFact.memoryFactAnnotation, Entity: key})
	}
	return out, nil
}

func (r retrievalModule) findAnnotationGraphVersionTargets(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope, entity, sourceMemoryID string) ([]annotationGraphVersionTarget, error) {
	facts, err := r.queryAnnotationGraphFacts(ctx, workspaceID, peer, memoryScope, q.SessionKey)
	if err != nil {
		return nil, err
	}
	out := []annotationGraphVersionTarget{}
	for _, relationFact := range facts {
		memoryID := strconv.FormatInt(relationFact.MemoryID, 10)
		if memoryID == sourceMemoryID {
			continue
		}
		subject, relation, nextEntity, ok := kgRelationAnswerParts(relationFact.Value)
		if !ok || !annotationGraphEntityMatches(entity, subject) {
			continue
		}
		for _, versionFact := range facts {
			versionSubject, _, ok := searchVersionAnswerParts(versionFact.Value)
			if !ok || !annotationGraphEntityMatches(nextEntity, versionSubject) {
				continue
			}
			candidate := RecallCandidate{
				MemoryID:   strconv.FormatInt(versionFact.MemoryID, 10),
				SourceType: memoryAnnotationSourceConclusion,
				Content:    versionFact.Content,
				SessionID:  versionFact.SessionKey,
				AgentID:    r.observer,
				ScopeID:    normalizeMemoryScope(memoryScope, ""),
				Provenance: []EvidenceItem{annotationFactEvidence(q.Query, versionFact.memoryFactAnnotation)},
			}
			out = append(out, annotationGraphVersionTarget{Candidate: candidate, RelationFact: relationFact.memoryFactAnnotation, VersionFact: versionFact.memoryFactAnnotation, Relation: relation, Entity: versionSubject})
		}
	}
	return out, nil
}

type annotationGraphFactRow struct {
	memoryFactAnnotation
	Content    string
	SessionKey string
}

func (r retrievalModule) queryAnnotationGraphFacts(ctx context.Context, workspaceID, peer, memoryScope, sessionKey string) ([]annotationGraphFactRow, error) {
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
	if sessionKey := strings.TrimSpace(sessionKey); sessionKey != "" {
		query += ` AND (c.session_key = ? OR c.session_key IS NULL)`
		args = append(args, sessionKey)
	}
	query += ` ORDER BY a.id ASC LIMIT 200`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("goncho: query annotation graph facts: %w", err)
	}
	defer rows.Close()

	out := []annotationGraphFactRow{}
	for rows.Next() {
		var fact annotationGraphFactRow
		if err := rows.Scan(&fact.ID, &fact.MemorySource, &fact.MemoryID, &fact.Value, &fact.Source, &fact.Confidence, &fact.Content, &fact.SessionKey); err != nil {
			return nil, fmt.Errorf("goncho: scan annotation graph fact: %w", err)
		}
		out = append(out, fact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate annotation graph facts: %w", err)
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

func annotationGraphTimelineEvidence(sourceMemoryID, targetMemoryID, entity, sourceFactID string, timelineFactID int64) EvidenceItem {
	timelineFactIDText := strconv.FormatInt(timelineFactID, 10)
	return EvidenceItem{
		Kind:   "graph",
		Source: sourceMemoryID,
		ID:     "annotation:" + sourceFactID + "->annotation:" + timelineFactIDText,
		Score:  1,
		Note:   sourceMemoryID + " -> owned_entity -> " + entity + " -> timeline -> " + targetMemoryID,
		Metadata: map[string]string{
			"entity":          entity,
			"relation":        "owned_entity",
			"source_fact_id":  sourceFactID,
			"target_fact_id":  timelineFactIDText,
			"target_relation": "timeline",
		},
	}
}

func annotationGraphLocationEvidence(sourceMemoryID, targetMemoryID, relation, entity, locationEntity, sourceFactID string, locationFactID int64) EvidenceItem {
	locationFactIDText := strconv.FormatInt(locationFactID, 10)
	return EvidenceItem{
		Kind:   "graph",
		Source: sourceMemoryID,
		ID:     "annotation:" + sourceFactID + "->annotation:" + locationFactIDText,
		Score:  1,
		Note:   sourceMemoryID + " -> " + kgRelationPhrase(relation) + " -> " + entity + " -> location -> " + targetMemoryID,
		Metadata: map[string]string{
			"entity":          entity,
			"location_entity": locationEntity,
			"relation":        relation,
			"source_fact_id":  sourceFactID,
			"target_fact_id":  locationFactIDText,
			"target_relation": "location",
		},
	}
}

func annotationGraphMetricEvidence(sourceMemoryID, targetMemoryID, relation, entity, metricEntity, sourceFactID string, metricFactID int64) EvidenceItem {
	metricFactIDText := strconv.FormatInt(metricFactID, 10)
	return EvidenceItem{
		Kind:   "graph",
		Source: sourceMemoryID,
		ID:     "annotation:" + sourceFactID + "->annotation:" + metricFactIDText,
		Score:  1,
		Note:   sourceMemoryID + " -> " + kgRelationPhrase(relation) + " -> " + entity + " -> metric -> " + targetMemoryID,
		Metadata: map[string]string{
			"entity":          entity,
			"metric_entity":   metricEntity,
			"relation":        relation,
			"source_fact_id":  sourceFactID,
			"target_fact_id":  metricFactIDText,
			"target_relation": "metric",
		},
	}
}

func annotationGraphVersionEvidence(sourceMemoryID, targetMemoryID, firstRelation, firstEntity, secondRelation, secondEntity, sourceFactID string, relationFactID, versionFactID int64) EvidenceItem {
	relationFactIDText := strconv.FormatInt(relationFactID, 10)
	versionFactIDText := strconv.FormatInt(versionFactID, 10)
	return EvidenceItem{
		Kind:   "graph",
		Source: sourceMemoryID,
		ID:     "annotation:" + sourceFactID + "->annotation:" + relationFactIDText + "->annotation:" + versionFactIDText,
		Score:  1,
		Note:   sourceMemoryID + " -> " + kgRelationPhrase(firstRelation) + " -> " + firstEntity + " -> " + kgRelationPhrase(secondRelation) + " -> " + secondEntity + " -> version -> " + targetMemoryID,
		Metadata: map[string]string{
			"entity":               firstEntity,
			"relation":             firstRelation,
			"source_fact_id":       sourceFactID,
			"intermediate_fact_id": relationFactIDText,
			"target_fact_id":       versionFactIDText,
			"target_relation":      "version",
		},
	}
}

func appendAnnotationGraphCandidate(out []RecallCandidate, indexByID map[string]int, candidate RecallCandidate, evidence EvidenceItem) []RecallCandidate {
	if idx, exists := indexByID[candidate.MemoryID]; exists {
		if !recallCandidateHasEvidence(out[idx], evidence.Kind, evidence.ID) {
			out[idx].Provenance = append(out[idx].Provenance, evidence)
		}
		return out
	}
	candidate.Provenance = append(candidate.Provenance, evidence)
	out = append(out, candidate)
	indexByID[candidate.MemoryID] = len(out) - 1
	return out
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

func annotationGraphVersionQuery(query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(query, "version") && (strings.Contains(query, "what") || strings.Contains(query, "which"))
}

func annotationGraphTimelineQuery(query string) bool {
	query = strings.ToLower(query)
	if !(strings.Contains(query, "when") || strings.Contains(query, "deadline") || strings.Contains(query, "scheduled") || strings.Contains(query, "date")) {
		return false
	}
	return strings.Contains(query, "owner") || strings.Contains(query, "owned") || strings.Contains(query, "responsible") || strings.Contains(query, "accountable")
}

func annotationGraphMetricQuery(query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(query, "how fast") || strings.Contains(query, "how many") || strings.Contains(query, "how much") || strings.Contains(query, "latency") || strings.Contains(query, "metric") || strings.Contains(query, "measurement")
}

func annotationGraphLocationQuery(query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(query, "where") || strings.Contains(query, "location") || strings.Contains(query, "located")
}

func annotationGraphQueryMatchesOwnerFact(query, owner string) bool {
	ownerTokens := searchRankTokenSet(owner)
	return len(ownerTokens) > 0 && searchRankTokenCoverage(ownerTokens, query) >= 0.80
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

func annotationGraphEntityMentionedInFact(entity, factKey string) bool {
	entityTokens := searchRankTokenSet(cleanFactObject(entity))
	return len(entityTokens) > 0 && searchRankTokenCoverage(entityTokens, factKey) >= 0.80
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
