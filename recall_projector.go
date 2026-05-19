package goncho

import (
	"strconv"
	"strings"
)

type RecallProjector struct{}

func (p *RecallProjector) ProjectSearch(trace RecallTrace) SearchResultSet {
	results := make([]SearchHit, 0, len(trace.Selected))
	for _, item := range trace.Selected {
		results = append(results, SearchHit{
			ID:         parseRecallMemoryID(item.Candidate.MemoryID),
			Source:     item.Candidate.SourceType,
			Content:    item.Candidate.Content,
			SessionKey: item.Candidate.SessionID,
		})
	}
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
	conclusions := make([]string, 0, len(search.Results))
	var representation strings.Builder
	for _, hit := range search.Results {
		if hit.Source == "conclusion" {
			conclusions = append(conclusions, hit.Content)
		}
		if strings.TrimSpace(hit.Content) == "" {
			continue
		}
		if representation.Len() > 0 {
			representation.WriteByte('\n')
		}
		representation.WriteString("- ")
		representation.WriteString(hit.Content)
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

func parseRecallMemoryID(id string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
