package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/utf8limit"

// TruncateUTF8Bytes returns value truncated to at most limit bytes without
// splitting a UTF-8 encoded rune.
func TruncateUTF8Bytes(value string, limit int) string {
	return utf8limit.TruncateBytes(value, limit)
}
