package goncho

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/queryexpand"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

type expandedQuery = queryexpand.Expanded

func expandSearchQuery(query string) expandedQuery {
	return queryexpand.Expand(query)
}

func expandSearchQueryWithAliases(query string, aliases map[string][]string) expandedQuery {
	return queryexpand.ExpandWithAliases(query, aliases)
}

func cloneQueryAliases(in map[string][]string) map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]string, len(in))
	for key, values := range in {
		key = strings.TrimSpace(strings.ToLower(key))
		if key == "" {
			continue
		}
		for _, value := range values {
			value = strings.TrimSpace(strings.ToLower(value))
			if value != "" {
				out[key] = append(out[key], value)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func queryExpansionEvidenceID(expansion expandedQuery) string {
	return textutil.LowerTrimmed(expansion.Original)
}

func queryExpansionEvidence(expansion expandedQuery) EvidenceItem {
	return EvidenceItem{
		Kind:     "query_expansion",
		Source:   "goncho_query_expansion",
		ID:       queryExpansionEvidenceID(expansion),
		Score:    1,
		Note:     "expanded query with transparent synonyms",
		Metadata: queryExpansionEvidenceMetadata(expansion),
	}
}

func queryExpansionEvidenceMetadata(expansion expandedQuery) map[string]string {
	metadata := map[string]string{
		"original_query": strings.TrimSpace(expansion.Original),
		"expanded_terms": strings.Join(expansion.Terms, ","),
	}
	if len(expansion.ConfiguredTerms) > 0 {
		metadata["configured_terms"] = strings.Join(expansion.ConfiguredTerms, ",")
		metadata["alias_source"] = "configured"
	} else if len(expansion.Terms) > 0 {
		metadata["alias_source"] = "built_in"
	}
	return metadata
}
