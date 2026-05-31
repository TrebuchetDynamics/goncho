package stableid

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/hashutil"
)

// TrimmedNullSeparated hashes trimmed parts with NUL separators so adjacent
// fields cannot collide by concatenation.
func TrimmedNullSeparated(prefixBytes int, parts ...string) string {
	var seed strings.Builder
	for _, part := range parts {
		seed.WriteString(strings.TrimSpace(part))
		seed.WriteByte(0)
	}
	return hashutil.SHA256HexStringPrefix(seed.String(), prefixBytes)
}
