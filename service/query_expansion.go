package goncho

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/queryexpand"
)

type expandedQuery = queryexpand.Expanded

func expandSearchQuery(query string) expandedQuery {
	return queryexpand.Expand(query)
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
