package goncho

import (
	"context"
	"strings"

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

type graphRelationEndpoints struct {
	FromMemoryID string
	ToMemoryID   string
}

func expandGraphRecallCandidates(q RecallQuery, base []RecallCandidate, index GraphExpansionIndex) []RecallCandidate {
	out := sliceutil.Clone(base)
	state := newGraphExpansionState(out, len(index.Memories))
	for {
		enriched := enrichExistingGraphRecallCandidates(q, index, state, out)
		added := false
		for _, relation := range index.Relations {
			endpoints, ok := graphRelationStableEndpoints(relation)
			if !ok || !graphRelationCanExpand(q, relation, endpoints, state.seen) {
				continue
			}
			source, ok := state.active[endpoints.FromMemoryID]
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
		endpoints, ok := graphRelationStableEndpoints(relation)
		if !ok || !graphRelationCanAnnotateExisting(q, relation, endpoints, state) {
			continue
		}
		source, sourceOK := state.active[endpoints.FromMemoryID]
		target, targetOK := state.active[endpoints.ToMemoryID]
		targetIndex, indexOK := state.index[endpoints.ToMemoryID]
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
	memoryID, stable := recallCandidateStableMemoryID(candidate)
	if !stable {
		return
	}
	s.seen[memoryID] = true
	s.active[memoryID] = candidate
	s.index[memoryID] = index
}

func graphRelationCanExpand(q RecallQuery, relation GraphRelation, endpoints graphRelationEndpoints, seen map[string]bool) bool {
	return graphRelationIsAccepted(relation) &&
		seen[endpoints.FromMemoryID] &&
		!seen[endpoints.ToMemoryID] &&
		graphRelationMatchesQuery(q.Query, relation.QueryTerms) &&
		graphRelationMatchesQuery(q.Query, relation.ActivationTerms)
}

func graphRelationCanAnnotateExisting(q RecallQuery, relation GraphRelation, endpoints graphRelationEndpoints, state graphExpansionState) bool {
	return graphRelationIsAccepted(relation) &&
		state.seen[endpoints.FromMemoryID] &&
		state.seen[endpoints.ToMemoryID] &&
		graphRelationMatchesQuery(q.Query, relation.QueryTerms) &&
		graphRelationMatchesQuery(q.Query, relation.ActivationTerms)
}

func graphRelationStableEndpoints(relation GraphRelation) (graphRelationEndpoints, bool) {
	endpoints := graphRelationEndpoints{
		FromMemoryID: strings.TrimSpace(relation.FromMemoryID),
		ToMemoryID:   strings.TrimSpace(relation.ToMemoryID),
	}
	return endpoints, endpoints.FromMemoryID != "" && endpoints.ToMemoryID != ""
}

func graphRelationTargetCandidate(relation GraphRelation, index GraphExpansionIndex) (RecallCandidate, bool) {
	endpoints, ok := graphRelationStableEndpoints(relation)
	if !ok {
		return RecallCandidate{}, false
	}
	target, ok := index.Memories[endpoints.ToMemoryID]
	if !ok {
		return RecallCandidate{}, false
	}
	targetMemoryID, stable := recallCandidateStableMemoryID(target)
	if !stable || targetMemoryID != endpoints.ToMemoryID {
		return RecallCandidate{}, false
	}
	target.MemoryID = targetMemoryID
	return target, true
}

func graphExpandedProvenance(source, target RecallCandidate, relation GraphRelation) []EvidenceItem {
	provenance := sliceutil.Clone(target.Provenance)
	provenance = append(provenance, graphPathEvidence(source.Provenance)...)
	endpoints, _ := graphRelationStableEndpoints(relation)
	provenance = append(provenance, EvidenceItem{
		Kind:   "graph",
		ID:     relation.EvidenceID,
		Source: endpoints.FromMemoryID,
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
	endpoints, _ := graphRelationStableEndpoints(relation)
	note := graphRelationNote(relation)
	for _, item := range candidate.Provenance {
		if item.Kind != "graph" {
			continue
		}
		if relation.EvidenceID != "" && item.ID == relation.EvidenceID {
			return true
		}
		if item.Source == endpoints.FromMemoryID && item.Note == note {
			return true
		}
	}
	return false
}

func graphRelationNote(relation GraphRelation) string {
	endpoints, _ := graphRelationStableEndpoints(relation)
	return endpoints.FromMemoryID + " -> " + relation.Relation + " -> " + endpoints.ToMemoryID
}

func graphRelationIsAccepted(relation GraphRelation) bool {
	state := textutil.LowerTrimmed(relation.State)
	return state == "" || state == GraphRelationAccepted
}

func graphRelationMatchesQuery(query string, terms []string) bool {
	return textutil.ContainsAllSubstringsFold(query, terms)
}
