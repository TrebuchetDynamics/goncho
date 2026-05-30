package textutil

import "strings"

// IsBlank reports whether value is empty after trimming Unicode whitespace.
func IsBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}

// NonBlank reports whether value has non-whitespace content.
func NonBlank(value string) bool {
	return !IsBlank(value)
}

// TrimSentenceBoundary removes the sentence punctuation policy used by recall
// fact classifiers, then trims surrounding whitespace. It intentionally keeps
// punctuation inside the value unchanged.
func TrimSentenceBoundary(value string) string {
	return strings.TrimSpace(strings.Trim(value, ".!?"))
}

// TrimQuestionPunctuation removes leading/trailing question punctuation before
// trimming whitespace, matching the policy used by fact-question classifiers.
func TrimQuestionPunctuation(value string) string {
	return strings.TrimSpace(strings.Trim(value, "?!."))
}

// TrimQuestionPhraseBoundary removes question punctuation, dots, and spaces as
// boundary characters, matching classifiers that accept loosely spaced prompts.
func TrimQuestionPhraseBoundary(value string) string {
	return strings.TrimSpace(strings.Trim(value, "?! ."))
}
