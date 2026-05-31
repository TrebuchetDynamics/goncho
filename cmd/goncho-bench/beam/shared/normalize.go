package shared

import "strings"

// NormalizeAbility returns the canonical BEAM ability code used in artifacts and match keys.
func NormalizeAbility(ability string) string {
	return strings.ToUpper(strings.TrimSpace(ability))
}

// NormalizeRecordType returns the canonical BEAM JSONL record type.
func NormalizeRecordType(recordType string) string {
	return strings.ToLower(strings.TrimSpace(recordType))
}

// NormalizeEvidenceKind returns the canonical BEAM evidence-kind token.
func NormalizeEvidenceKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}

// NormalizeQuestionText returns the canonical question text used for BEAM question-key matching.
func NormalizeQuestionText(question string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(question))), " ")
}
