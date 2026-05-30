package codeclaim

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var pathPattern = regexp.MustCompile(`\b[[:alnum:]_./-]+\.(?:go|ts|tsx|js|jsx|py|rs|cs|java|cpp|c|h|hpp)\b`)

func ExtractPaths(content string) []string {
	return pathPattern.FindAllString(content, -1)
}

func PathExists(repoRoot, rel string) bool {
	rel = filepath.Clean(strings.TrimSpace(rel))
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false
	}
	info, err := os.Stat(filepath.Join(repoRoot, rel))
	return err == nil && !info.IsDir()
}

func Representation(peer string, claims []Claim) string {
	if len(claims) == 0 {
		return "No live-verified code claims for " + peer + "."
	}
	var b strings.Builder
	b.WriteString("Live-verified code claims for ")
	b.WriteString(peer)
	b.WriteString(":")
	for _, claim := range claims {
		b.WriteString("\n- ")
		b.WriteString(claim.Path)
		b.WriteString(": ")
		b.WriteString(claim.Content)
	}
	return b.String()
}

type Claim struct {
	Path    string
	Content string
}
