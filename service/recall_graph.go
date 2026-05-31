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

type graphExpansionState struct {
	seen   map[string]bool
	active map[string]RecallCandidate
	index  map[string]int
}

func expandGraphRecallCandidates(q RecallQuery, base []RecallCandidate, index GraphExpansionIndex) []RecallCandidate {
	out := sliceutil.Clone(base)
	state := newGraphExpansionState(out, len(index.Memories))
	for {
		enriched := enrichExistingGraphRecallCandidates(q, index, state, out)
		added := false
		for _, relation := range index.Relations {
			if !graphRelationCanExpand(q, relation, state.seen) {
				continue
			}
			source, ok := state.active[relation.FromMemoryID]
			if !ok {
				continue
			}
			target, ok := graphRelationTargetCandidate(relation, index)
			if !ok || recallScopeMismatch(q, target) {
				continue
			}
			target.Provenance = graphExpandedProvenance(source, target, relation)
			out = append(out, target)
			state.addAt(target, len(out)-1)
			added = true
		}
		if !added && !enriched {
			return out
		}
	}
}

func enrichExistingGraphRecallCandidates(q RecallQuery, index GraphExpansionIndex, state graphExpansionState, candidates []RecallCandidate) bool {
	enriched := false
	for _, relation := range index.Relations {
		if !graphRelationCanAnnotateExisting(q, relation, state) {
			continue
		}
		source, sourceOK := state.active[relation.FromMemoryID]
		target, targetOK := state.active[relation.ToMemoryID]
		targetIndex, indexOK := state.index[relation.ToMemoryID]
		indexedTarget, indexedOK := graphRelationTargetCandidate(relation, index)
		if !sourceOK || !targetOK || !indexOK || !indexedOK || recallScopeMismatch(q, indexedTarget) || graphCandidateHasRelationEvidence(target, relation) {
			continue
		}
		target.Provenance = graphExpandedProvenance(source, target, relation)
		candidates[targetIndex] = target
		state.active[target.MemoryID] = target
		enriched = true
	}
	return enriched
}

func newGraphExpansionState(candidates []RecallCandidate, extraCapacity int) graphExpansionState {
	state := graphExpansionState{
		seen:   make(map[string]bool, len(candidates)+extraCapacity),
		active: make(map[string]RecallCandidate, len(candidates)+extraCapacity),
		index:  make(map[string]int, len(candidates)+extraCapacity),
	}
	for i, candidate := range candidates {
		state.addAt(candidate, i)
	}
	return state
}

func (s graphExpansionState) addAt(candidate RecallCandidate, index int) {
	if candidate.MemoryID == "" {
		return
	}
	s.seen[candidate.MemoryID] = true
	s.active[candidate.MemoryID] = candidate
	s.index[candidate.MemoryID] = index
}

func graphRelationCanExpand(q RecallQuery, relation GraphRelation, seen map[string]bool) bool {
	return graphRelationHasEndpoints(relation) &&
		graphRelationIsAccepted(relation) &&
		seen[relation.FromMemoryID] &&
		!seen[relation.ToMemoryID] &&
		graphRelationMatchesQuery(q.Query, relation.QueryTerms) &&
		graphRelationMatchesQuery(q.Query, relation.ActivationTerms)
}

func graphRelationCanAnnotateExisting(q RecallQuery, relation GraphRelation, state graphExpansionState) bool {
	return graphRelationHasEndpoints(relation) &&
		graphRelationIsAccepted(relation) &&
		state.seen[relation.FromMemoryID] &&
		state.seen[relation.ToMemoryID] &&
		graphRelationMatchesQuery(q.Query, relation.QueryTerms) &&
		graphRelationMatchesQuery(q.Query, relation.ActivationTerms)
}

func graphRelationHasEndpoints(relation GraphRelation) bool {
	return relation.FromMemoryID != "" && relation.ToMemoryID != ""
}

func graphRelationTargetCandidate(relation GraphRelation, index GraphExpansionIndex) (RecallCandidate, bool) {
	if !graphRelationHasEndpoints(relation) {
		return RecallCandidate{}, false
	}
	target, ok := index.Memories[relation.ToMemoryID]
	if !ok || target.MemoryID != relation.ToMemoryID {
		return RecallCandidate{}, false
	}
	return target, true
}

func graphExpandedProvenance(source, target RecallCandidate, relation GraphRelation) []EvidenceItem {
	provenance := sliceutil.Clone(target.Provenance)
	provenance = append(provenance, graphPathEvidence(source.Provenance)...)
	provenance = append(provenance, EvidenceItem{
		Kind:   "graph",
		ID:     relation.EvidenceID,
		Source: relation.FromMemoryID,
		Note:   graphRelationNote(relation),
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

func graphCandidateHasRelationEvidence(candidate RecallCandidate, relation GraphRelation) bool {
	note := graphRelationNote(relation)
	for _, item := range candidate.Provenance {
		if item.Kind != "graph" {
			continue
		}
		if relation.EvidenceID != "" && item.ID == relation.EvidenceID {
			return true
		}
		if item.Source == relation.FromMemoryID && item.Note == note {
			return true
		}
	}
	return false
}

func graphRelationNote(relation GraphRelation) string {
	return relation.FromMemoryID + " -> " + relation.Relation + " -> " + relation.ToMemoryID
}

func graphRelationIsAccepted(relation GraphRelation) bool {
	state := textutil.LowerTrimmed(relation.State)
	return state == "" || state == GraphRelationAccepted
}

func graphRelationMatchesQuery(query string, terms []string) bool {
	return textutil.ContainsAllSubstringsFold(query, terms)
}
