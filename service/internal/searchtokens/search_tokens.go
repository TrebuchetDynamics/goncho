package searchtokens

import (
	"regexp"
	"strings"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

func Tokens(value string) []string {
	out := []string{}
	for _, token := range tokenPattern.FindAllString(strings.ToLower(value), -1) {
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
	if len(want) == 0 {
		return 0
	}
	got := TokenSet(value)
	if len(got) == 0 {
		return 0
	}
	hits := 0
	for token := range want {
		if _, ok := got[token]; ok {
			hits++
		}
	}
	return float64(hits) / float64(len(want))
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
