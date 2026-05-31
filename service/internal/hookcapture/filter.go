package hookcapture

import (
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/maputil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sensitive"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

// Payload carries host-hook text fields that share the same redaction and
// truncation policy before they are persisted as observations or messages.
type Payload struct {
	Content  string
	Input    string
	Output   string
	Error    string
	Summary  string
	Metadata map[string]string
}

// Result reports filtering evidence for hook metadata.
type Result struct {
	Payload        Payload
	Redacted       bool
	RedactionCount int
	Truncated      bool
}

var redactionRules = []struct {
	kind string
	re   *regexp.Regexp
}{
	{kind: "private", re: regexp.MustCompile(`(?is)<private>.*?</private>`)},
	{kind: "pem_private_key", re: regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)},
	{kind: "authorization", re: regexp.MustCompile(`(?i)Authorization:\s*Bearer\s+[^\s\r\n]+`)},
	{kind: "json_secret", re: regexp.MustCompile(`(?i)"([^"]*(?:secret|token|password|api_key|private_key|authorization)[^"]*)"\s*:\s*"[^"]*"`)},
	{kind: "env_secret", re: regexp.MustCompile(`(?im)\b[A-Z0-9_]*(?:SECRET|TOKEN|PASSWORD|API_KEY|PRIVATE_KEY)[A-Z0-9_]*\s*=\s*[^\s\r\n]+`)},
	{kind: "api_key", re: regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]+|ghp_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+)\b`)},
}

// Filter applies Goncho's host-hook safety policy: force valid UTF-8, redact
// known secret shapes and sensitive metadata keys, then truncate payload fields.
func Filter(payload Payload, maxBytes int) Result {
	out := Result{Payload: payload}
	var filtered string
	filtered, out = FilterString(payload.Content, out, maxBytes)
	out.Payload.Content = filtered
	filtered, out = FilterString(payload.Input, out, maxBytes)
	out.Payload.Input = filtered
	filtered, out = FilterString(payload.Output, out, maxBytes)
	out.Payload.Output = filtered
	filtered, out = FilterString(payload.Error, out, maxBytes)
	out.Payload.Error = filtered
	filtered, out = FilterString(payload.Summary, out, maxBytes)
	out.Payload.Summary = filtered
	if payload.Metadata != nil {
		out.Payload.Metadata = maputil.CloneStringString(payload.Metadata)
		for key, value := range out.Payload.Metadata {
			filteredValue, next := FilterString(value, out, maxBytes)
			out = next
			if SensitiveMetadataKey(key) && filteredValue == value && textutil.NonBlank(value) {
				filteredValue = "[REDACTED:metadata_secret]"
				out.Redacted = true
				out.RedactionCount++
			}
			out.Payload.Metadata[key] = filteredValue
		}
	}
	return out
}

// FilterString applies the same string-level safety policy to one field.
func FilterString(value string, state Result, maxBytes int) (string, Result) {
	value = strings.ToValidUTF8(value, "\uFFFD")
	for _, rule := range redactionRules {
		count := 0
		value = rule.re.ReplaceAllStringFunc(value, func(match string) string {
			count++
			if rule.kind == "json_secret" {
				parts := strings.SplitN(match, ":", 2)
				if len(parts) == 2 {
					return parts[0] + `:"[REDACTED:json_secret]"`
				}
			}
			return "[REDACTED:" + rule.kind + "]"
		})
		if count > 0 {
			state.Redacted = true
			state.RedactionCount += count
		}
	}
	if maxBytes > 0 && len([]byte(value)) > maxBytes {
		value = textutil.TruncateUTF8Bytes(value, maxBytes)
		state.Truncated = true
	}
	return value, state
}

func SensitiveMetadataKey(key string) bool {
	return sensitive.MetadataKeySecretLike(key)
}
