package sensitive

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil"

var contentSecretNeedles = []string{"password", "api token", "secret", "private key", "sk-live", "bearer "}

var metadataKeySecretNeedles = []string{"secret", "token", "password", "api_key", "private_key", "authorization"}

// ContainsSecretLikeContent reports whether free-form content contains the
// stable secret/privacy needles Goncho uses to prevent proposed active memory
// from storing credentials or similarly sensitive values.
func ContainsSecretLikeContent(value string) bool {
	return textutil.ContainsAnySubstringFold(value, contentSecretNeedles)
}

// MetadataKeySecretLike reports whether a metadata key conventionally names a
// secret-bearing value and should be redacted even if the value itself has no
// recognisable secret shape.
func MetadataKeySecretLike(key string) bool {
	return textutil.ContainsAnySubstringFold(key, metadataKeySecretNeedles)
}
