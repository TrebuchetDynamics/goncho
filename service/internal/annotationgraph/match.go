package annotationgraph

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/searchintent"
	"github.com/TrebuchetDynamics/goncho/service/internal/searchtokens"
)

const (
	RelationUses      = "uses"
	RelationDependsOn = "depends_on"
	RelationRunsOn    = "runs_on"
)

func TimelineQuery(query string) bool {
	query = strings.ToLower(query)
	if !(strings.Contains(query, "when") || strings.Contains(query, "deadline") || strings.Contains(query, "scheduled") || strings.Contains(query, "date")) {
		return false
	}
	return strings.Contains(query, "owner") || strings.Contains(query, "owned") || strings.Contains(query, "responsible") || strings.Contains(query, "accountable")
}

func MetricQuery(query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(query, "how fast") || strings.Contains(query, "how many") || strings.Contains(query, "how much") || strings.Contains(query, "latency") || strings.Contains(query, "metric") || strings.Contains(query, "measurement")
}

func LocationQuery(query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(query, "where") || strings.Contains(query, "location") || strings.Contains(query, "located")
}

func PreferenceQuery(query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(query, "prefer") || strings.Contains(query, "preference")
}

func InstructionQuery(query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(query, "instruction") || strings.Contains(query, "rule")
}

func SequenceQuery(query string) bool {
	if _, ok := searchintent.SequenceQuestionSubject(query); ok {
		return true
	}
	query = strings.ToLower(query)
	return strings.Contains(query, "sequence") || strings.Contains(query, "order")
}

func DecisionQuery(query string) bool {
	if _, ok := searchintent.DecisionQuestionTopic(query); ok {
		return true
	}
	query = strings.ToLower(query)
	return strings.Contains(query, "decision") || strings.Contains(query, "decide")
}

func NegationQuery(query string) bool {
	if _, ok := searchintent.NegationQuestionObject(query); ok {
		return true
	}
	query = strings.ToLower(query)
	return strings.Contains(query, "never") || strings.Contains(query, " not ")
}

func QueryMatchesOwnerFact(query, owner string) bool {
	ownerTokens := searchtokens.TokenSet(owner)
	return len(ownerTokens) > 0 && searchtokens.Coverage(ownerTokens, query) >= 0.80
}

func QueryMatchesKGRelation(query, subject, relation string) bool {
	subjectTokens := searchtokens.TokenSet(subject)
	if len(subjectTokens) == 0 || searchtokens.Coverage(subjectTokens, query) < 0.80 {
		return false
	}
	query = strings.ToLower(query)
	switch relation {
	case RelationUses:
		return strings.Contains(query, "use") || strings.Contains(query, "used") || strings.Contains(query, "using")
	case RelationDependsOn:
		return strings.Contains(query, "depend") || strings.Contains(query, "dependency")
	case RelationRunsOn:
		return strings.Contains(query, "runs on") || strings.Contains(query, "running on")
	default:
		return false
	}
}

func EntityMatches(a, b string) bool {
	a = searchintent.CleanFactObject(a)
	b = searchintent.CleanFactObject(b)
	if strings.EqualFold(a, b) {
		return true
	}
	aTokens := searchtokens.TokenSet(a)
	bTokens := searchtokens.TokenSet(b)
	return len(aTokens) > 0 && searchtokens.Coverage(aTokens, b) >= 0.80 && searchtokens.Coverage(bTokens, a) >= 0.80
}

func EntityMentionedInFact(entity, factKey string) bool {
	entityTokens := searchtokens.TokenSet(searchintent.CleanFactObject(entity))
	return len(entityTokens) > 0 && searchtokens.Coverage(entityTokens, factKey) >= 0.80
}

func OwnerFactParts(fact string) (owner, entity string, ok bool) {
	fact = strings.TrimSpace(strings.Trim(fact, ".!?"))
	lower := strings.ToLower(fact)
	idx := strings.Index(lower, " owns ")
	if idx <= 0 {
		return "", "", false
	}
	owner = searchintent.CleanFactValue(fact[:idx])
	entity = searchintent.CleanFactObject(fact[idx+len(" owns "):])
	return owner, entity, searchintent.FactSubjectLooksAssertive(owner) && searchintent.FactObjectLooksAssertive(entity)
}

func KGRelationAnswerParts(sentence string) (subject, relation, object string, ok bool) {
	sentence = strings.TrimSpace(strings.Trim(sentence, ".!?"))
	if sentence == "" || strings.Contains(sentence, "?") {
		return "", "", "", false
	}
	lower := strings.ToLower(sentence)
	for _, marker := range []struct {
		text     string
		relation string
	}{
		{text: " depends on ", relation: RelationDependsOn},
		{text: " runs on ", relation: RelationRunsOn},
		{text: " uses ", relation: RelationUses},
	} {
		idx := strings.Index(lower, marker.text)
		if idx <= 0 {
			continue
		}
		subject = searchintent.CleanFactObject(sentence[:idx])
		object = searchintent.CleanFactValue(sentence[idx+len(marker.text):])
		if searchintent.FactObjectLooksAssertive(subject) && searchintent.FactObjectLooksAssertive(object) {
			return subject, marker.relation, object, true
		}
	}
	return "", "", "", false
}

func KGRelationPhrase(relation string) string {
	switch relation {
	case RelationUses:
		return "uses"
	case RelationDependsOn:
		return "depends on"
	case RelationRunsOn:
		return "runs on"
	default:
		return ""
	}
}
