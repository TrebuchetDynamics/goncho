package searchtokens

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textmatch"
	"github.com/TrebuchetDynamics/goncho/service/internal/texttokens"
)

func Tokens(value string) []string {
	out := []string{}
	for _, token := range texttokens.LowerAlnum(value) {
		token = Stem(token)
		if len(token) < 3 || Stopword(token) {
			continue
		}
		out = append(out, token)
	}
	return out
}

func TokenSet(value string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, token := range Tokens(value) {
		out[token] = struct{}{}
	}
	return out
}

func TermFrequency(value string) map[string]int {
	out := map[string]int{}
	for _, token := range Tokens(value) {
		out[token]++
	}
	return out
}

func Coverage(want map[string]struct{}, value string) float64 {
	return textmatch.Coverage(want, TokenSet(value))
}

func Stem(token string) string {
	for _, suffix := range []string{"ing", "edly", "edly", "ed", "es", "s"} {
		if len(token) > len(suffix)+3 && strings.HasSuffix(token, suffix) {
			return strings.TrimSuffix(token, suffix)
		}
	}
	return token
}

func Stopword(token string) bool {
	switch token {
	case "the", "and", "for", "who", "what", "when", "where", "which", "should", "not", "did", "does", "with", "that", "this", "from", "are", "was", "were", "has", "have", "had", "you", "your", "about", "can", "could", "would", "there", "their", "they", "them", "then", "than":
		return true
	default:
		return false
	}
}
