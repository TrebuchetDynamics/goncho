package shared

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/identity"

// OutcomeKey identifies a BEAM row by the exact scale/conversation/qid match contract.
type OutcomeKey = identity.OutcomeKey

// NewOutcomeKey normalizes BEAM row identity fields for maps and deterministic sorting.
func NewOutcomeKey(scale, conversationID, qid string) OutcomeKey {
	return identity.NewOutcomeKey(scale, conversationID, qid)
}

// QuestionKey identifies a BEAM row by normalized question fallback identity.
type QuestionKey = identity.QuestionKey

// NewQuestionKey normalizes BEAM question fallback fields for oracle judgment and paired-outcome matching.
func NewQuestionKey(scale, conversationID, ability, question string) QuestionKey {
	return identity.NewQuestionKey(scale, conversationID, ability, question)
}
