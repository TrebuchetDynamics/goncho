package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/trimmed"

// IsBlank reports whether value is empty after trimming Unicode whitespace.
func IsBlank(value string) bool {
	return trimmed.Blank(value)
}

// NonBlank reports whether value has non-whitespace content.
func NonBlank(value string) bool {
	return trimmed.NonBlank(value)
}

// UpperTrimmed trims surrounding whitespace and converts the value to upper case.
func UpperTrimmed(value string) string {
	return trimmed.Upper(value)
}

// TrimSentenceBoundary removes the sentence punctuation policy used by recall
// fact classifiers, then trims surrounding whitespace. It intentionally keeps
// punctuation inside the value unchanged.
func TrimSentenceBoundary(value string) string {
	return trimmed.SentenceBoundary(value)
}

// TrimQuestionPunctuation removes leading/trailing question punctuation before
// trimming whitespace, matching the policy used by fact-question classifiers.
func TrimQuestionPunctuation(value string) string {
	return trimmed.QuestionPunctuation(value)
}

// TrimQuestionPhraseBoundary removes question punctuation, dots, and spaces as
// boundary characters, matching classifiers that accept loosely spaced prompts.
func TrimQuestionPhraseBoundary(value string) string {
	return trimmed.QuestionPhraseBoundary(value)
}
