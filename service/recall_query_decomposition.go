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
		return sliceutil.FilterMap(queries, func(query string) (RecallQuery, bool) {
			query = strings.TrimSpace(query)
			if query == "" || query == q.Query {
				return RecallQuery{}, false
			}
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
