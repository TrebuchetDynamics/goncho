package memproposal

import (
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/sensitive"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

var durableFactPattern = regexp.MustCompile(`(?i)^(.+?)\s+(?:is|are|lives in|uses|owns)\s+(.+)$`)

// SplitMarker parses the user-facing memory proposal prefix from content.
func SplitMarker(content string) (string, string, bool) {
	prefix, body, ok := strings.Cut(content, ":")
	if !ok {
		return "", "", false
	}
	prefix = textutil.LowerTrimmed(prefix)
	switch prefix {
	case "remember", "update", "supersede", "forget", "delete", "preference", "procedure", "lesson":
		return prefix, strings.TrimSpace(body), true
	default:
		return "", "", false
	}
}

// Subject derives a stable review/search subject from proposal content.
func Subject(content string) string {
	content = strings.Trim(strings.TrimSpace(content), ".")
	matches := durableFactPattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return textutil.FirstWords(content, 5)
}

// IsLowConfidence detects hedged claims that should remain review-only.
func IsLowConfidence(content string) bool {
	return textutil.ContainsAnySubstringFold(content, []string{"maybe ", " might ", "not sure", "i think"})
}

// IsPrivacySensitive detects secret-like content that should remain review-only.
func IsPrivacySensitive(content string) bool {
	return sensitive.ContainsSecretLikeContent(content)
}
