package goncho

import (
	"context"
	"strings"
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
		out := make([]RecallQuery, 0, len(queries))
		for _, query := range queries {
			query = strings.TrimSpace(query)
			if query == "" || query == q.Query {
				continue
			}
			sub := q
			sub.Query = query
			out = append(out, sub)
		}
		return out
	}
}

func (g queryDecomposingRecallGenerator) Generate(ctx context.Context, q RecallQuery) ([]RecallCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	queries := []RecallQuery{q}
	if g.planner != nil {
		queries = append(queries, g.planner(q)...)
	}
	seen := map[string]bool{}
	out := []RecallCandidate{}
	for _, query := range queries {
		if strings.TrimSpace(query.Query) == "" {
			continue
		}
		items, err := g.base.Generate(ctx, query)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.MemoryID == "" {
				out = append(out, item)
				continue
			}
			if seen[item.MemoryID] {
				continue
			}
			seen[item.MemoryID] = true
			out = append(out, item)
		}
	}
	return out, nil
}
