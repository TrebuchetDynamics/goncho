package shared

import "strings"

// OutcomeKey identifies a BEAM row by the exact scale/conversation/qid match contract.
type OutcomeKey struct {
	Scale          string
	ConversationID string
	QID            string
}

// NewOutcomeKey normalizes BEAM row identity fields for maps and deterministic sorting.
func NewOutcomeKey(scale, conversationID, qid string) OutcomeKey {
	return OutcomeKey{
		Scale:          strings.TrimSpace(scale),
		ConversationID: strings.TrimSpace(conversationID),
		QID:            strings.TrimSpace(qid),
	}
}

func (k OutcomeKey) Empty() bool {
	return k.Scale == "" && k.ConversationID == "" && k.QID == ""
}

func (k OutcomeKey) Less(other OutcomeKey) bool {
	if k.Scale != other.Scale {
		return k.Scale < other.Scale
	}
	if k.ConversationID != other.ConversationID {
		return k.ConversationID < other.ConversationID
	}
	return k.QID < other.QID
}

// QuestionKey identifies a BEAM row by normalized question fallback identity.
type QuestionKey struct {
	Scale          string
	ConversationID string
	Ability        string
	Question       string
}

// NewQuestionKey normalizes BEAM question fallback fields for oracle judgment and paired-outcome matching.
func NewQuestionKey(scale, conversationID, ability, question string) QuestionKey {
	return QuestionKey{
		Scale:          strings.TrimSpace(scale),
		ConversationID: strings.TrimSpace(conversationID),
		Ability:        NormalizeAbility(ability),
		Question:       NormalizeQuestionText(question),
	}
}

func (k QuestionKey) Empty() bool {
	return k.Scale == "" && k.ConversationID == "" && k.Ability == "" && k.Question == ""
}
