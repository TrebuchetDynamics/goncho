package goncho

import "strings"

type expandedQuery struct {
	Original string
	Expanded string
	Terms    []string
}

var queryExpansionSynonyms = map[string][]string{
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

func expandSearchQuery(query string) expandedQuery {
	original := strings.TrimSpace(query)
	if original == "" {
		return expandedQuery{}
	}
	seen := map[string]struct{}{}
	terms := []string{}
	for _, token := range searchRankTokens(original) {
		for _, synonym := range queryExpansionSynonyms[token] {
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
		return expandedQuery{Original: original, Expanded: original}
	}
	return expandedQuery{Original: original, Expanded: original + " " + strings.Join(terms, " "), Terms: terms}
}

func (e expandedQuery) Applied() bool {
	return len(e.Terms) > 0 && strings.TrimSpace(e.Expanded) != strings.TrimSpace(e.Original)
}

func evidenceListHas(items []EvidenceItem, kind, id string) bool {
	for _, item := range items {
		if item.Kind == kind && item.ID == id {
			return true
		}
	}
	return false
}

func queryExpansionEvidence(expansion expandedQuery) EvidenceItem {
	return EvidenceItem{
		Kind:   "query_expansion",
		Source: "goncho_query_expansion",
		ID:     strings.ToLower(strings.TrimSpace(expansion.Original)),
		Score:  1,
		Note:   "expanded query with transparent synonyms",
		Metadata: map[string]string{
			"original_query": strings.TrimSpace(expansion.Original),
			"expanded_terms": strings.Join(expansion.Terms, ","),
		},
	}
}
