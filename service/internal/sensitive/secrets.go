package sensitive

import "strings"

var contentSecretNeedles = []string{"password", "api token", "secret", "private key", "sk-live", "bearer "}

var metadataKeySecretNeedles = []string{"secret", "token", "password", "api_key", "private_key", "authorization"}

// ContainsSecretLikeContent reports whether free-form content contains the
// stable secret/privacy needles Goncho uses to prevent proposed active memory
// from storing credentials or similarly sensitive values.
func ContainsSecretLikeContent(value string) bool {
	return containsAnyLower(value, contentSecretNeedles)
}

// MetadataKeySecretLike reports whether a metadata key conventionally names a
// secret-bearing value and should be redacted even if the value itself has no
// recognisable secret shape.
func MetadataKeySecretLike(key string) bool {
	return containsAnyLower(strings.TrimSpace(key), metadataKeySecretNeedles)
}

func containsAnyLower(value string, needles []string) bool {
	lower := strings.ToLower(value)
	for _, needle := range needles {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}
