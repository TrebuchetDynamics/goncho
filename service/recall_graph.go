package goncho

import (
	"context"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
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
	return expandGraphRecallCandidates(q, base, g.index), nil
}

func expandGraphRecallCandidates(q RecallQuery, base []RecallCandidate, index GraphExpansionIndex) []RecallCandidate {
	out := sliceutil.Clone(base)
	seen := make(map[string]bool, len(out)+len(index.Memories))
	active := make(map[string]RecallCandidate, len(out)+len(index.Memories))
	for _, candidate := range out {
		seen[candidate.MemoryID] = true
		active[candidate.MemoryID] = candidate
	}
	for {
		added := false
		for _, relation := range index.Relations {
			if !graphRelationCanExpand(q, relation, seen) {
				continue
			}
			source, ok := active[relation.FromMemoryID]
			if !ok {
				continue
			}
			target, ok := index.Memories[relation.ToMemoryID]
			if !ok || recallScopeMismatch(q, target) {
				continue
			}
			target.Provenance = graphExpandedProvenance(source, target, relation)
			out = append(out, target)
			seen[target.MemoryID] = true
			active[target.MemoryID] = target
			added = true
		}
		if !added {
			return out
		}
	}
}

func graphRelationCanExpand(q RecallQuery, relation GraphRelation, seen map[string]bool) bool {
	return graphRelationIsAccepted(relation) &&
		seen[relation.FromMemoryID] &&
		!seen[relation.ToMemoryID] &&
		graphRelationMatchesQuery(q.Query, relation.QueryTerms) &&
		graphRelationMatchesQuery(q.Query, relation.ActivationTerms)
}

func graphExpandedProvenance(source, target RecallCandidate, relation GraphRelation) []EvidenceItem {
	provenance := sliceutil.Clone(target.Provenance)
	provenance = append(provenance, graphPathEvidence(source.Provenance)...)
	provenance = append(provenance, EvidenceItem{
		Kind:   "graph",
		ID:     relation.EvidenceID,
		Source: relation.FromMemoryID,
		Note:   relation.FromMemoryID + " -> " + relation.Relation + " -> " + relation.ToMemoryID,
		Score:  relation.Score,
	})
	return provenance
}

func graphPathEvidence(provenance []EvidenceItem) []EvidenceItem {
	out := []EvidenceItem{}
	for _, item := range provenance {
		if item.Kind == "graph" {
			out = append(out, item)
		}
	}
	return out
}

func graphRelationIsAccepted(relation GraphRelation) bool {
	state := textutil.LowerTrimmed(relation.State)
	return state == "" || state == GraphRelationAccepted
}

func graphRelationMatchesQuery(query string, terms []string) bool {
	return textutil.ContainsAllSubstringsFold(query, terms)
}
