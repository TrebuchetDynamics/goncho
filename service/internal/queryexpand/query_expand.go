package queryexpand

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/searchtokens"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

type Expanded struct {
	Original string
	Expanded string
	Terms    []string
}

var synonyms = map[string][]string{
	"auth":     {"authentication", "login", "credentials", "oauth"},
	"login":    {"auth", "authentication", "credentials", "signin"},
	"signin":   {"login", "authentication", "credentials"},
	"db":       {"database", "postgres", "postgresql", "sqlite"},
	"database": {"db", "postgres", "postgresql", "sqlite"},
	"err":      {"error", "failure", "exception"},
	"error":    {"failure", "exception", "err"},
	"failure":  {"error", "exception", "failed"},
	"owner":    {"owns", "owned", "responsible"},
	"pref":     {"preference", "prefers", "prefer"},
}

func Expand(query string) Expanded {
	original := strings.TrimSpace(query)
	if original == "" {
		return Expanded{}
	}
	terms := []string{}
	for _, token := range searchtokens.Tokens(original) {
		terms = append(terms, synonyms[token]...)
	}
	terms = textutil.UniqueLowerTrimmed(terms, false)
	if len(terms) == 0 {
		return Expanded{Original: original, Expanded: original}
	}
	return Expanded{Original: original, Expanded: original + " " + strings.Join(terms, " "), Terms: terms}
}

func (e Expanded) Applied() bool {
	return len(e.Terms) > 0 && strings.TrimSpace(e.Expanded) != strings.TrimSpace(e.Original)
}
