package queryexpand

import (
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/searchtokens"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

type Expanded struct {
	Original string
	Expanded string
	Terms    []string
}

var explicitSynonyms = map[string][]string{
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

var synonyms = reciprocalLexicon(explicitSynonyms)

func reciprocalLexicon(entries map[string][]string) map[string][]string {
	out := map[string][]string{}
	for term, aliases := range entries {
		termKeys := synonymLookupKeys(term)
		for _, termKey := range termKeys {
			out[termKey] = appendUnique(out[termKey], aliases...)
		}
		for _, alias := range aliases {
			for _, aliasKey := range synonymLookupKeys(alias) {
				out[aliasKey] = appendUnique(out[aliasKey], term)
			}
		}
	}
	makeLexiconReciprocal(out)
	sortLexiconAliases(out)
	return out
}

func makeLexiconReciprocal(lexicon map[string][]string) {
	pairs := []struct {
		term    string
		aliases []string
	}{}
	for term, aliases := range lexicon {
		pairs = append(pairs, struct {
			term    string
			aliases []string
		}{term: term, aliases: aliases})
	}
	for _, pair := range pairs {
		for _, alias := range pair.aliases {
			lexicon[alias] = appendUnique(lexicon[alias], pair.term)
		}
	}
}

func synonymLookupKeys(value string) []string {
	keys := []string{strings.TrimSpace(strings.ToLower(value))}
	keys = append(keys, searchtokens.Tokens(value)...)
	return textutil.UniqueLowerTrimmed(keys, false)
}

func sortLexiconAliases(lexicon map[string][]string) {
	for term := range lexicon {
		sort.Strings(lexicon[term])
	}
}

func appendUnique(values []string, candidates ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(candidates))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(strings.ToLower(candidate))
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		values = append(values, candidate)
	}
	return values
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
	sort.Strings(terms)
	if len(terms) == 0 {
		return Expanded{Original: original, Expanded: original}
	}
	return Expanded{Original: original, Expanded: original + " " + strings.Join(terms, " "), Terms: terms}
}

func (e Expanded) Applied() bool {
	return len(e.Terms) > 0 && strings.TrimSpace(e.Expanded) != strings.TrimSpace(e.Original)
}
