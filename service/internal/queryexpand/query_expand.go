package queryexpand

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/searchtokens"
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
	seen := map[string]struct{}{}
	terms := []string{}
	for _, token := range searchtokens.Tokens(original) {
		for _, synonym := range synonyms[token] {
			synonym = strings.TrimSpace(strings.ToLower(synonym))
			if synonym == "" {
				continue
			}
			if _, ok := seen[synonym]; ok {
				continue
			}
			seen[synonym] = struct{}{}
			terms = append(terms, synonym)
		}
	}
	if len(terms) == 0 {
		return Expanded{Original: original, Expanded: original}
	}
	return Expanded{Original: original, Expanded: original + " " + strings.Join(terms, " "), Terms: terms}
}

func (e Expanded) Applied() bool {
	return len(e.Terms) > 0 && strings.TrimSpace(e.Expanded) != strings.TrimSpace(e.Original)
}
