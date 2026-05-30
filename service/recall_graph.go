package goncho

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

type GraphExpansionIndex struct {
	Memories  map[string]RecallCandidate
	Relations []GraphRelation
}

const (
	GraphRelationAccepted = "accepted"
	GraphRelationPending  = "pending"
)

type GraphRelation struct {
	FromMemoryID    string
	ToMemoryID      string
	Relation        string
	QueryTerms      []string
	ActivationTerms []string
	EvidenceID      string
	Score           float64
	State           string
}

type graphExpandingRecallGenerator struct {
	base  recallCandidateGenerator
	index GraphExpansionIndex
}

func newGraphExpandingRecallGenerator(base recallCandidateGenerator, index GraphExpansionIndex) recallCandidateGenerator {
	return graphExpandingRecallGenerator{base: base, index: index}
}

func (g graphExpandingRecallGenerator) Generate(ctx context.Context, q RecallQuery) ([]RecallCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	base, err := g.base.Generate(ctx, q)
	if err != nil {
		return nil, err
	}
	out := sliceutil.Clone(base)
	seen := make(map[string]bool, len(out))
	for _, candidate := range out {
		seen[candidate.MemoryID] = true
	}
	for _, relation := range g.index.Relations {
		if !graphRelationIsAccepted(relation) || !seen[relation.FromMemoryID] || seen[relation.ToMemoryID] || !graphRelationMatchesQuery(q.Query, relation.QueryTerms) || !graphRelationMatchesQuery(q.Query, relation.ActivationTerms) {
			continue
		}
		target, ok := g.index.Memories[relation.ToMemoryID]
		if !ok || recallScopeMismatch(q, target) {
			continue
		}
		target.Provenance = append(sliceutil.Clone(target.Provenance), EvidenceItem{
			Kind:   "graph",
			ID:     relation.EvidenceID,
			Source: relation.FromMemoryID,
			Note:   relation.FromMemoryID + " -> " + relation.Relation + " -> " + relation.ToMemoryID,
			Score:  relation.Score,
		})
		out = append(out, target)
		seen[target.MemoryID] = true
	}
	return out, nil
}

func graphRelationIsAccepted(relation GraphRelation) bool {
	state := strings.ToLower(strings.TrimSpace(relation.State))
	return state == "" || state == GraphRelationAccepted
}

func graphRelationMatchesQuery(query string, terms []string) bool {
	query = strings.ToLower(query)
	for _, term := range terms {
		if !strings.Contains(query, strings.ToLower(strings.TrimSpace(term))) {
			return false
		}
	}
	return true
}
