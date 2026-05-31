package goncho

import (
	"context"
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
	seen := map[string]struct{}{}
	for _, query := range plannedRecallQueries(q, g.planner) {
		items, err := g.base.Generate(ctx, query)
		if err != nil {
			return nil, err
		}
		out = appendMergedRecallCandidates(out, seen, items)
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

func appendMergedRecallCandidates(out []RecallCandidate, seen map[string]struct{}, items []RecallCandidate) []RecallCandidate {
	for _, item := range items {
		if item.MemoryID == "" {
			out = append(out, item)
			continue
		}
		if _, ok := seen[item.MemoryID]; ok {
			continue
		}
		seen[item.MemoryID] = struct{}{}
		out = append(out, item)
	}
	return out
}

func recallQueryKey(q RecallQuery) string {
	return recallQueryStringKey(q.Query)
}

func recallQueryStringKey(query string) string {
	return strings.ToLower(strings.Join(strings.Fields(query), " "))
}
