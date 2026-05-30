package goncho

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/idutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

type RecallProjector struct{}

func searchHitFromScoredRecallCandidate(item ScoredRecallCandidate) SearchHit {
	return SearchHit{
		ID:         parseRecallMemoryID(item.Candidate.MemoryID),
		Source:     item.Candidate.SourceType,
		Content:    item.Candidate.Content,
		SessionKey: item.Candidate.SessionID,
	}
}

func (p *RecallProjector) ProjectSearch(trace RecallTrace) SearchResultSet {
	results := sliceutil.Map(trace.Selected, searchHitFromScoredRecallCandidate)
	if results == nil {
		results = []SearchHit{}
	}
	return SearchResultSet{
		WorkspaceID: trace.Query.WorkspaceID,
		Peer:        trace.Query.Peer,
		Query:       trace.Query.Query,
		Results:     results,
	}
}

func (p *RecallProjector) ProjectContext(trace RecallTrace) ContextResult {
	search := p.ProjectSearch(trace)
	conclusions := conclusionsFromSearchHits(search.Results)
	var representation strings.Builder
	for _, item := range trace.Selected {
		hit := searchHitFromScoredRecallCandidate(item)
		if strings.TrimSpace(hit.Content) == "" {
			continue
		}
		if representation.Len() > 0 {
			representation.WriteByte('\n')
		}
		representation.WriteString("- ")
		representation.WriteString(hit.Content)
		for _, note := range graphRelationPathNotes(item.Candidate.Provenance) {
			representation.WriteString("\n  relation path: ")
			representation.WriteString(note)
		}
	}
	return ContextResult{
		WorkspaceID:    trace.Query.WorkspaceID,
		Peer:           trace.Query.Peer,
		SessionKey:     trace.Query.SessionKey,
		Representation: representation.String(),
		Conclusions:    conclusions,
		SearchResults:  search.Results,
	}
}

func graphRelationPathNotes(provenance []EvidenceItem) []string {
	notes := make([]string, 0)
	seen := make(map[string]bool)
	for _, item := range provenance {
		note := strings.TrimSpace(item.Note)
		if item.Kind != "graph" || note == "" || seen[note] {
			continue
		}
		notes = append(notes, note)
		seen[note] = true
	}
	return notes
}

func parseRecallMemoryID(id string) int64 {
	n, err := idutil.ParseDecimal(id)
	if err != nil {
		return 0
	}
	return n
}
