package goncho

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

type recallSubqueryPlanner func(RecallQuery) []RecallQuery

type queryDecomposingRecallGenerator struct {
	base    recallCandidateGenerator
	planner recallSubqueryPlanner
}

func newQueryDecomposingRecallGenerator(base recallCandidateGenerator, planner recallSubqueryPlanner) recallCandidateGenerator {
	return queryDecomposingRecallGenerator{base: base, planner: planner}
}

func fixedRecallSubqueries(queries ...string) recallSubqueryPlanner {
	return func(q RecallQuery) []RecallQuery {
		original := recallQueryKey(q)
		seen := map[string]struct{}{original: {}}
		return sliceutil.FilterMap(queries, func(query string) (RecallQuery, bool) {
			query = strings.TrimSpace(query)
			key := recallQueryStringKey(query)
			if key == "" {
				return RecallQuery{}, false
			}
			if _, ok := seen[key]; ok {
				return RecallQuery{}, false
			}
			seen[key] = struct{}{}
			sub := q
			sub.Query = query
			return sub, true
		})
	}
}

func (g queryDecomposingRecallGenerator) Generate(ctx context.Context, q RecallQuery) ([]RecallCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := []RecallCandidate{}
	indexByMemoryID := map[string]int{}
	for _, query := range plannedRecallQueries(q, g.planner) {
		items, err := g.base.Generate(ctx, query)
		if err != nil {
			return nil, err
		}
		out = appendMergedRecallCandidates(out, indexByMemoryID, items)
	}
	return out, nil
}

func plannedRecallQueries(q RecallQuery, planner recallSubqueryPlanner) []RecallQuery {
	queries := []RecallQuery{}
	seen := map[string]struct{}{}
	appendQuery := func(query RecallQuery) {
		key := recallQueryKey(query)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		queries = append(queries, query)
	}
	appendQuery(q)
	if planner != nil {
		for _, query := range planner(q) {
			appendQuery(query)
		}
	}
	return queries
}

func appendMergedRecallCandidates(out []RecallCandidate, indexByMemoryID map[string]int, items []RecallCandidate) []RecallCandidate {
	for _, item := range items {
		if item.MemoryID == "" {
			out = append(out, item)
			continue
		}
		if idx, ok := indexByMemoryID[item.MemoryID]; ok {
			out[idx] = mergeRecallCandidateEvidence(out[idx], item)
			continue
		}
		indexByMemoryID[item.MemoryID] = len(out)
		out = append(out, item)
	}
	return out
}

func recallCandidateEvidenceKey(evidence EvidenceItem) string {
	return evidence.Kind + "\x00" + evidence.Source + "\x00" + evidence.ID + "\x00" + evidence.Note
}

func recallQueryKey(q RecallQuery) string {
	query := recallQueryStringKey(q.Query)
	if query == "" {
		return ""
	}
	parts := []string{
		query,
		recallQueryKeyText(q.WorkspaceID),
		recallQueryKeyText(q.Peer),
		recallQueryKeyText(q.SessionKey),
		recallQueryKeyText(q.ScopeID),
		recallQuerySourcesKey(q.Sources),
		strconv.Itoa(q.Limit),
		strconv.Itoa(q.MaxTokens),
	}
	return strings.Join(parts, "\x00")
}

func recallQueryStringKey(query string) string {
	return recallQueryKeyText(query)
}

func recallQueryKeyText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func recallQuerySourcesKey(sources []string) string {
	if len(sources) == 0 {
		return "*"
	}
	out := make([]string, 0, len(sources))
	seen := map[string]struct{}{}
	for _, source := range sources {
		key := recallQueryKeyText(source)
		if key == "" || key == "*" {
			return "*"
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}
