package goncho

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/annotationgraph"
	"github.com/TrebuchetDynamics/goncho/service/internal/idutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

const (
	kgRelationUses      = annotationgraph.RelationUses
	kgRelationDependsOn = annotationgraph.RelationDependsOn
	kgRelationRunsOn    = annotationgraph.RelationRunsOn
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

type annotationGraphPreferenceTarget struct {
	Candidate      RecallCandidate
	PreferenceFact memoryFactAnnotation
	Entity         string
	Attribute      string
}

type annotationGraphInstructionTarget struct {
	Candidate       RecallCandidate
	InstructionFact memoryFactAnnotation
	Entity          string
}

type annotationGraphSequenceTarget struct {
	Candidate    RecallCandidate
	SequenceFact memoryFactAnnotation
	Entity       string
}

type annotationGraphDecisionTarget struct {
	Candidate    RecallCandidate
	DecisionFact memoryFactAnnotation
}

type annotationGraphNegationTarget struct {
	Candidate    RecallCandidate
	NegationFact memoryFactAnnotation
}

func (r retrievalModule) expandAnnotationGraphRecall(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope string, base []RecallCandidate) ([]RecallCandidate, error) {
	ownerQuery := annotationGraphOwnerQuery(q.Query)
	versionQuery := annotationGraphVersionQuery(q.Query)
	timelineQuery := annotationgraph.TimelineQuery(q.Query)
	metricQuery := annotationgraph.MetricQuery(q.Query)
	locationQuery := annotationgraph.LocationQuery(q.Query)
	preferenceQuery := annotationgraph.PreferenceQuery(q.Query)
	instructionQuery := annotationgraph.InstructionQuery(q.Query)
	sequenceQuery := annotationgraph.SequenceQuery(q.Query)
	decisionQuery := annotationgraph.DecisionQuery(q.Query)
	negationQuery := annotationgraph.NegationQuery(q.Query)
	if len(base) == 0 || (!ownerQuery && !versionQuery && !timelineQuery && !metricQuery && !locationQuery && !preferenceQuery && !instructionQuery && !sequenceQuery && !decisionQuery && !negationQuery) {
		return base, nil
	}
	out := sliceutil.Clone(base)
	indexByID := recallCandidateIndexByStableMemoryID(out)
	for _, source := range base {
		for _, evidence := range source.Provenance {
			if evidence.Kind != "fact" || evidence.Source != "goncho_memory_annotations" {
				continue
			}
			fact := strings.TrimPrefix(evidence.Note, "fact=")
			if timelineQuery {
				owner, entity, ok := annotationgraph.OwnerFactParts(fact)
				if ok && annotationgraph.QueryMatchesOwnerFact(q.Query, owner) {
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
			subject, relation, entity, ok := annotationgraph.KGRelationAnswerParts(fact)
			if !ok || !annotationgraph.QueryMatchesKGRelation(q.Query, subject, relation) {
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
			if preferenceQuery {
				targets, err := r.findAnnotationGraphPreferenceTargets(ctx, q, workspaceID, peer, memoryScope, entity, source.MemoryID)
				if err != nil {
					return nil, err
				}
				for _, target := range targets {
					graphEvidence := annotationGraphPreferenceEvidence(source.MemoryID, target.Candidate.MemoryID, relation, entity, target.Entity, target.Attribute, evidence.ID, target.PreferenceFact.ID)
					out = appendAnnotationGraphCandidate(out, indexByID, target.Candidate, graphEvidence)
				}
			}
			if instructionQuery {
				targets, err := r.findAnnotationGraphInstructionTargets(ctx, q, workspaceID, peer, memoryScope, entity, source.MemoryID)
				if err != nil {
					return nil, err
				}
				for _, target := range targets {
					graphEvidence := annotationGraphInstructionEvidence(source.MemoryID, target.Candidate.MemoryID, relation, entity, target.Entity, evidence.ID, target.InstructionFact.ID)
					out = appendAnnotationGraphCandidate(out, indexByID, target.Candidate, graphEvidence)
				}
			}
			if sequenceQuery {
				targets, err := r.findAnnotationGraphSequenceTargets(ctx, q, workspaceID, peer, memoryScope, entity, source.MemoryID)
				if err != nil {
					return nil, err
				}
				for _, target := range targets {
					graphEvidence := annotationGraphSequenceEvidence(source.MemoryID, target.Candidate.MemoryID, relation, entity, target.Entity, evidence.ID, target.SequenceFact.ID)
					out = appendAnnotationGraphCandidate(out, indexByID, target.Candidate, graphEvidence)
				}
			}
			if decisionQuery {
				targets, err := r.findAnnotationGraphDecisionTargets(ctx, q, workspaceID, peer, memoryScope, entity, source.MemoryID)
				if err != nil {
					return nil, err
				}
				for _, target := range targets {
					graphEvidence := annotationGraphDecisionEvidence(source.MemoryID, target.Candidate.MemoryID, relation, entity, evidence.ID, target.DecisionFact.ID)
					out = appendAnnotationGraphCandidate(out, indexByID, target.Candidate, graphEvidence)
				}
			}
			if negationQuery {
				targets, err := r.findAnnotationGraphNegationTargets(ctx, q, workspaceID, peer, memoryScope, entity, source.MemoryID)
				if err != nil {
					return nil, err
				}
				for _, target := range targets {
					graphEvidence := annotationGraphNegationEvidence(source.MemoryID, target.Candidate.MemoryID, relation, entity, evidence.ID, target.NegationFact.ID)
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
		memoryID := idutil.Decimal(fact.MemoryID)
		if memoryID == sourceMemoryID {
			continue
		}
		_, owned, ok := annotationgraph.OwnerFactParts(fact.Value)
		if !ok || !annotationgraph.EntityMatches(entity, owned) {
			continue
		}
		candidate := annotationGraphCandidateFromFact(q, fact, content, sessionKey, r.observer, memoryScope)
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
		memoryID := idutil.Decimal(timelineFact.MemoryID)
		if memoryID == sourceMemoryID {
			continue
		}
		event, _, ok := searchTimelineAnswerParts(timelineFact.Value)
		if !ok || !annotationgraph.EntityMatches(entity, event) {
			continue
		}
		candidate := annotationGraphCandidateFromFact(q, timelineFact.memoryFactAnnotation, timelineFact.Content, timelineFact.SessionKey, r.observer, memoryScope)
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
		memoryID := idutil.Decimal(locationFact.MemoryID)
		if memoryID == sourceMemoryID {
			continue
		}
		object, _, ok := searchLocationAnswerParts(locationFact.Value)
		if !ok || !annotationgraph.EntityMatches(entity, object) {
			continue
		}
		candidate := annotationGraphCandidateFromFact(q, locationFact.memoryFactAnnotation, locationFact.Content, locationFact.SessionKey, r.observer, memoryScope)
		out = append(out, annotationGraphLocationTarget{Candidate: candidate, LocationFact: locationFact.memoryFactAnnotation, Entity: object})
	}
	return out, nil
}

func (r retrievalModule) findAnnotationGraphPreferenceTargets(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope, entity, sourceMemoryID string) ([]annotationGraphPreferenceTarget, error) {
	facts, err := r.queryAnnotationGraphFacts(ctx, workspaceID, peer, memoryScope, q.SessionKey)
	if err != nil {
		return nil, err
	}
	_, queryAttribute, attributeOK := searchPreferenceQuestion(q.Query)
	attributeTokens := searchRankTokenSet(queryAttribute)
	out := []annotationGraphPreferenceTarget{}
	for _, preferenceFact := range facts {
		memoryID := idutil.Decimal(preferenceFact.MemoryID)
		if memoryID == sourceMemoryID {
			continue
		}
		subject, _, attribute, ok := searchPreferenceAnswerParts(preferenceFact.Value)
		if !ok || !annotationgraph.EntityMatches(entity, subject) {
			continue
		}
		if attributeOK && len(attributeTokens) > 0 && searchRankTokenCoverage(attributeTokens, attribute) < 0.80 {
			continue
		}
		candidate := annotationGraphCandidateFromFact(q, preferenceFact.memoryFactAnnotation, preferenceFact.Content, preferenceFact.SessionKey, r.observer, memoryScope)
		out = append(out, annotationGraphPreferenceTarget{Candidate: candidate, PreferenceFact: preferenceFact.memoryFactAnnotation, Entity: subject, Attribute: attribute})
	}
	return out, nil
}

func (r retrievalModule) findAnnotationGraphInstructionTargets(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope, entity, sourceMemoryID string) ([]annotationGraphInstructionTarget, error) {
	facts, err := r.queryAnnotationGraphFacts(ctx, workspaceID, peer, memoryScope, q.SessionKey)
	if err != nil {
		return nil, err
	}
	_, queryTopic, topicOK := searchInstructionQuestion(q.Query)
	topicTokens := searchRankTokenSet(queryTopic)
	out := []annotationGraphInstructionTarget{}
	for _, instructionFact := range facts {
		memoryID := idutil.Decimal(instructionFact.MemoryID)
		if memoryID == sourceMemoryID {
			continue
		}
		subject, instruction, ok := searchInstructionAnswerParts(instructionFact.Value)
		if !ok || !annotationgraph.EntityMatches(entity, subject) {
			continue
		}
		if topicOK && len(topicTokens) > 0 && searchRankTokenCoverage(topicTokens, instruction) < 0.80 {
			continue
		}
		candidate := annotationGraphCandidateFromFact(q, instructionFact.memoryFactAnnotation, instructionFact.Content, instructionFact.SessionKey, r.observer, memoryScope)
		out = append(out, annotationGraphInstructionTarget{Candidate: candidate, InstructionFact: instructionFact.memoryFactAnnotation, Entity: subject})
	}
	return out, nil
}

func (r retrievalModule) findAnnotationGraphSequenceTargets(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope, entity, sourceMemoryID string) ([]annotationGraphSequenceTarget, error) {
	facts, err := r.queryAnnotationGraphFacts(ctx, workspaceID, peer, memoryScope, q.SessionKey)
	if err != nil {
		return nil, err
	}
	out := []annotationGraphSequenceTarget{}
	for _, sequenceFact := range facts {
		memoryID := idutil.Decimal(sequenceFact.MemoryID)
		if memoryID == sourceMemoryID {
			continue
		}
		subject, _, ok := searchSequenceAnswerParts(sequenceFact.Value)
		if !ok || !annotationgraph.EntityMentionedInFact(entity, subject) {
			continue
		}
		candidate := annotationGraphCandidateFromFact(q, sequenceFact.memoryFactAnnotation, sequenceFact.Content, sequenceFact.SessionKey, r.observer, memoryScope)
		out = append(out, annotationGraphSequenceTarget{Candidate: candidate, SequenceFact: sequenceFact.memoryFactAnnotation, Entity: subject})
	}
	return out, nil
}

func (r retrievalModule) findAnnotationGraphDecisionTargets(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope, entity, sourceMemoryID string) ([]annotationGraphDecisionTarget, error) {
	facts, err := r.queryAnnotationGraphFacts(ctx, workspaceID, peer, memoryScope, q.SessionKey)
	if err != nil {
		return nil, err
	}
	out := []annotationGraphDecisionTarget{}
	for _, decisionFact := range facts {
		memoryID := idutil.Decimal(decisionFact.MemoryID)
		if memoryID == sourceMemoryID {
			continue
		}
		decision, ok := searchDecisionAnswerParts(decisionFact.Value)
		if !ok || !annotationgraph.EntityMentionedInFact(entity, decision) {
			continue
		}
		candidate := annotationGraphCandidateFromFact(q, decisionFact.memoryFactAnnotation, decisionFact.Content, decisionFact.SessionKey, r.observer, memoryScope)
		out = append(out, annotationGraphDecisionTarget{Candidate: candidate, DecisionFact: decisionFact.memoryFactAnnotation})
	}
	return out, nil
}

func (r retrievalModule) findAnnotationGraphNegationTargets(ctx context.Context, q RecallQuery, workspaceID, peer, memoryScope, entity, sourceMemoryID string) ([]annotationGraphNegationTarget, error) {
	facts, err := r.queryAnnotationGraphFacts(ctx, workspaceID, peer, memoryScope, q.SessionKey)
	if err != nil {
		return nil, err
	}
	out := []annotationGraphNegationTarget{}
	for _, negationFact := range facts {
		memoryID := idutil.Decimal(negationFact.MemoryID)
		if memoryID == sourceMemoryID {
			continue
		}
		object, ok := searchNegationAnswerParts(negationFact.Value)
		if !ok || !annotationgraph.EntityMentionedInFact(entity, object) {
			continue
		}
		candidate := annotationGraphCandidateFromFact(q, negationFact.memoryFactAnnotation, negationFact.Content, negationFact.SessionKey, r.observer, memoryScope)
		out = append(out, annotationGraphNegationTarget{Candidate: candidate, NegationFact: negationFact.memoryFactAnnotation})
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
		memoryID := idutil.Decimal(metricFact.MemoryID)
		if memoryID == sourceMemoryID {
			continue
		}
		key, _, ok := searchMetricAnswerParts(metricFact.Value)
		if !ok || !annotationgraph.EntityMentionedInFact(entity, key) {
			continue
		}
		candidate := annotationGraphCandidateFromFact(q, metricFact.memoryFactAnnotation, metricFact.Content, metricFact.SessionKey, r.observer, memoryScope)
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
		memoryID := idutil.Decimal(relationFact.MemoryID)
		if memoryID == sourceMemoryID {
			continue
		}
		subject, relation, nextEntity, ok := annotationgraph.KGRelationAnswerParts(relationFact.Value)
		if !ok || !annotationgraph.EntityMatches(entity, subject) {
			continue
		}
		for _, versionFact := range facts {
			versionSubject, _, ok := searchVersionAnswerParts(versionFact.Value)
			if !ok || !annotationgraph.EntityMatches(nextEntity, versionSubject) {
				continue
			}
			candidate := annotationGraphCandidateFromFact(q, versionFact.memoryFactAnnotation, versionFact.Content, versionFact.SessionKey, r.observer, memoryScope)
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

func annotationGraphCandidateFromFact(q RecallQuery, fact memoryFactAnnotation, content, sessionKey, observer, memoryScope string) RecallCandidate {
	return RecallCandidate{
		MemoryID:   idutil.Decimal(fact.MemoryID),
		SourceType: memoryAnnotationSourceConclusion,
		Content:    content,
		SessionID:  sessionKey,
		AgentID:    observer,
		ScopeID:    normalizeMemoryScope(memoryScope, ""),
		Provenance: []EvidenceItem{annotationFactEvidence(q.Query, fact)},
	}
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
		ID:     idutil.Decimal(fact.ID),
		Score:  roundRecallFloat(searchFactIntentScore(query, fact.Value)),
		Note:   "fact=" + strings.TrimSpace(fact.Value),
		Metadata: map[string]string{
			"memory_source": fact.MemorySource,
			"memory_id":     idutil.Decimal(fact.MemoryID),
			"source":        fact.Source,
			"confidence":    fmt.Sprintf("%.3f", fact.Confidence),
		},
	}
}

func annotationGraphEvidenceItem(sourceMemoryID string, details annotationgraph.EvidenceDetails) EvidenceItem {
	return EvidenceItem{Kind: "graph", Source: sourceMemoryID, ID: details.ID, Score: 1, Note: details.Note, Metadata: details.Metadata}
}

func annotationGraphEvidence(sourceMemoryID, targetMemoryID, relation, entity, sourceFactID string, targetFactID int64) EvidenceItem {
	return annotationGraphEvidenceItem(sourceMemoryID, annotationgraph.RelationEvidence(sourceMemoryID, targetMemoryID, relation, entity, sourceFactID, targetFactID))
}

func annotationGraphTimelineEvidence(sourceMemoryID, targetMemoryID, entity, sourceFactID string, timelineFactID int64) EvidenceItem {
	return annotationGraphEvidenceItem(sourceMemoryID, annotationgraph.TimelineEvidence(sourceMemoryID, targetMemoryID, entity, sourceFactID, timelineFactID))
}

func annotationGraphLocationEvidence(sourceMemoryID, targetMemoryID, relation, entity, locationEntity, sourceFactID string, locationFactID int64) EvidenceItem {
	return annotationGraphEvidenceItem(sourceMemoryID, annotationgraph.RelatedEvidence(sourceMemoryID, targetMemoryID, relation, entity, "location", "location_entity", locationEntity, sourceFactID, locationFactID))
}

func annotationGraphPreferenceEvidence(sourceMemoryID, targetMemoryID, relation, entity, preferenceEntity, attribute, sourceFactID string, preferenceFactID int64) EvidenceItem {
	return annotationGraphEvidenceItem(sourceMemoryID, annotationgraph.PreferenceEvidence(sourceMemoryID, targetMemoryID, relation, entity, preferenceEntity, attribute, sourceFactID, preferenceFactID))
}

func annotationGraphInstructionEvidence(sourceMemoryID, targetMemoryID, relation, entity, instructionEntity, sourceFactID string, instructionFactID int64) EvidenceItem {
	return annotationGraphEvidenceItem(sourceMemoryID, annotationgraph.RelatedEvidence(sourceMemoryID, targetMemoryID, relation, entity, "instruction", "instruction_entity", instructionEntity, sourceFactID, instructionFactID))
}

func annotationGraphSequenceEvidence(sourceMemoryID, targetMemoryID, relation, entity, sequenceEntity, sourceFactID string, sequenceFactID int64) EvidenceItem {
	return annotationGraphEvidenceItem(sourceMemoryID, annotationgraph.RelatedEvidence(sourceMemoryID, targetMemoryID, relation, entity, "sequence", "sequence_entity", sequenceEntity, sourceFactID, sequenceFactID))
}

func annotationGraphDecisionEvidence(sourceMemoryID, targetMemoryID, relation, entity, sourceFactID string, decisionFactID int64) EvidenceItem {
	return annotationGraphEvidenceItem(sourceMemoryID, annotationgraph.RelatedEvidence(sourceMemoryID, targetMemoryID, relation, entity, "decision", "", "", sourceFactID, decisionFactID))
}

func annotationGraphNegationEvidence(sourceMemoryID, targetMemoryID, relation, entity, sourceFactID string, negationFactID int64) EvidenceItem {
	return annotationGraphEvidenceItem(sourceMemoryID, annotationgraph.RelatedEvidence(sourceMemoryID, targetMemoryID, relation, entity, "negation", "", "", sourceFactID, negationFactID))
}

func annotationGraphMetricEvidence(sourceMemoryID, targetMemoryID, relation, entity, metricEntity, sourceFactID string, metricFactID int64) EvidenceItem {
	return annotationGraphEvidenceItem(sourceMemoryID, annotationgraph.RelatedEvidence(sourceMemoryID, targetMemoryID, relation, entity, "metric", "metric_entity", metricEntity, sourceFactID, metricFactID))
}

func annotationGraphVersionEvidence(sourceMemoryID, targetMemoryID, firstRelation, firstEntity, secondRelation, secondEntity, sourceFactID string, relationFactID, versionFactID int64) EvidenceItem {
	return annotationGraphEvidenceItem(sourceMemoryID, annotationgraph.VersionEvidence(sourceMemoryID, targetMemoryID, firstRelation, firstEntity, secondRelation, secondEntity, sourceFactID, relationFactID, versionFactID))
}

func appendAnnotationGraphCandidate(out []RecallCandidate, indexByID map[string]int, candidate RecallCandidate, evidence EvidenceItem) []RecallCandidate {
	memoryID, stable := recallCandidateStableMemoryID(candidate)
	if stable {
		if idx, exists := lookupRecallCandidateStableIndex(out, indexByID, memoryID); exists {
			if !recallCandidateHasEvidence(out[idx], evidence.Kind, evidence.ID) {
				out[idx].Provenance = append(out[idx].Provenance, evidence)
			}
			return out
		}
	}
	candidate.Provenance = append(candidate.Provenance, evidence)
	out = append(out, candidate)
	if stable {
		indexByID[memoryID] = len(out) - 1
	}
	return out
}

func recallCandidateHasEvidence(candidate RecallCandidate, kind, id string) bool {
	return evidenceListHas(candidate.Provenance, kind, id)
}

func annotationGraphOwnerQuery(query string) bool {
	if !textutil.ContainsAnySubstringFold(query, []string{"owner", "owns", "responsible", "accountable"}) {
		return false
	}
	return textutil.ContainsAnySubstringFold(query, []string{"who", "which", "what"})
}

func annotationGraphVersionQuery(query string) bool {
	return textutil.ContainsAnySubstringFold(query, []string{"version"}) && textutil.ContainsAnySubstringFold(query, []string{"what", "which"})
}
