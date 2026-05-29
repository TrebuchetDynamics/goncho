package annotationgraph

import "strconv"

type EvidenceDetails struct {
	ID       string
	Note     string
	Metadata map[string]string
}

func RelationEvidence(sourceMemoryID, targetMemoryID, relation, entity, sourceFactID string, targetFactID int64) EvidenceDetails {
	targetFactIDText := strconv.FormatInt(targetFactID, 10)
	return EvidenceDetails{
		ID:   "annotation:" + sourceFactID + "->annotation:" + targetFactIDText,
		Note: sourceMemoryID + " -> " + KGRelationPhrase(relation) + " -> " + entity + " -> owned_by -> " + targetMemoryID,
		Metadata: map[string]string{
			"entity":          entity,
			"relation":        relation,
			"source_fact_id":  sourceFactID,
			"target_fact_id":  targetFactIDText,
			"target_relation": "owned_by",
		},
	}
}

func TimelineEvidence(sourceMemoryID, targetMemoryID, entity, sourceFactID string, timelineFactID int64) EvidenceDetails {
	timelineFactIDText := strconv.FormatInt(timelineFactID, 10)
	return EvidenceDetails{
		ID:   "annotation:" + sourceFactID + "->annotation:" + timelineFactIDText,
		Note: sourceMemoryID + " -> owned_entity -> " + entity + " -> timeline -> " + targetMemoryID,
		Metadata: map[string]string{
			"entity":          entity,
			"relation":        "owned_entity",
			"source_fact_id":  sourceFactID,
			"target_fact_id":  timelineFactIDText,
			"target_relation": "timeline",
		},
	}
}

func RelatedEvidence(sourceMemoryID, targetMemoryID, relation, entity, targetRelation, relatedEntityKey, relatedEntity, sourceFactID string, targetFactID int64) EvidenceDetails {
	targetFactIDText := strconv.FormatInt(targetFactID, 10)
	metadata := map[string]string{
		"entity":          entity,
		"relation":        relation,
		"source_fact_id":  sourceFactID,
		"target_fact_id":  targetFactIDText,
		"target_relation": targetRelation,
	}
	if relatedEntityKey != "" {
		metadata[relatedEntityKey] = relatedEntity
	}
	return EvidenceDetails{
		ID:       "annotation:" + sourceFactID + "->annotation:" + targetFactIDText,
		Note:     sourceMemoryID + " -> " + KGRelationPhrase(relation) + " -> " + entity + " -> " + targetRelation + " -> " + targetMemoryID,
		Metadata: metadata,
	}
}

func PreferenceEvidence(sourceMemoryID, targetMemoryID, relation, entity, preferenceEntity, attribute, sourceFactID string, preferenceFactID int64) EvidenceDetails {
	details := RelatedEvidence(sourceMemoryID, targetMemoryID, relation, entity, "preference", "preference_entity", preferenceEntity, sourceFactID, preferenceFactID)
	details.Metadata["attribute"] = attribute
	return details
}

func VersionEvidence(sourceMemoryID, targetMemoryID, firstRelation, firstEntity, secondRelation, secondEntity, sourceFactID string, relationFactID, versionFactID int64) EvidenceDetails {
	relationFactIDText := strconv.FormatInt(relationFactID, 10)
	versionFactIDText := strconv.FormatInt(versionFactID, 10)
	return EvidenceDetails{
		ID:   "annotation:" + sourceFactID + "->annotation:" + relationFactIDText + "->annotation:" + versionFactIDText,
		Note: sourceMemoryID + " -> " + KGRelationPhrase(firstRelation) + " -> " + firstEntity + " -> " + KGRelationPhrase(secondRelation) + " -> " + secondEntity + " -> version -> " + targetMemoryID,
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
