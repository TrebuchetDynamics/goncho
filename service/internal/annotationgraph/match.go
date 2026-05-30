package annotationgraph

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/searchintent"
	"github.com/TrebuchetDynamics/goncho/service/internal/searchtokens"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

const (
	RelationUses      = "uses"
	RelationDependsOn = "depends_on"
	RelationRunsOn    = "runs_on"
)

func TimelineQuery(query string) bool {
	return queryHasAny(query, "when", "deadline", "scheduled", "date") && queryHasAny(query, "owner", "owned", "responsible", "accountable")
}

func MetricQuery(query string) bool {
	return queryHasAny(query, "how fast", "how many", "how much", "latency", "metric", "measurement")
}

func LocationQuery(query string) bool {
	return queryHasAny(query, "where", "location", "located")
}

func PreferenceQuery(query string) bool {
	return queryHasAny(query, "prefer", "preference")
}

func InstructionQuery(query string) bool {
	return queryHasAny(query, "instruction", "rule")
}

func SequenceQuery(query string) bool {
	if _, ok := searchintent.SequenceQuestionSubject(query); ok {
		return true
	}
	return queryHasAny(query, "sequence", "order")
}

func DecisionQuery(query string) bool {
	if _, ok := searchintent.DecisionQuestionTopic(query); ok {
		return true
	}
	return queryHasAny(query, "decision", "decide")
}

func NegationQuery(query string) bool {
	if _, ok := searchintent.NegationQuestionObject(query); ok {
		return true
	}
	return queryHasAny(query, "never", " not ")
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
	switch relation {
	case RelationUses:
		return queryHasAny(query, "use", "used", "using")
	case RelationDependsOn:
		return queryHasAny(query, "depend", "dependency")
	case RelationRunsOn:
		return queryHasAny(query, "runs on", "running on")
	default:
		return false
	}
}

func queryHasAny(query string, markers ...string) bool {
	return textutil.ContainsAnySubstringFold(query, markers)
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
	fact = textutil.TrimSentenceBoundary(fact)
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
	sentence = textutil.TrimSentenceBoundary(sentence)
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
