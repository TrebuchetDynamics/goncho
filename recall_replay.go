package goncho

import (
	"fmt"
	"strings"
)

const (
	RecallReplayStageQuery    = "query"
	RecallReplayStageWarn     = "warn"
	RecallReplayKindQuery     = "recall_query"
	RecallReplayKindCandidate = "candidate_scored"
	RecallReplayKindWarning   = "warning"
	RecallReplayKindSelected  = "selected"
	RecallReplayKindRejected  = "rejected"
	RecallReplayKindProject   = "projection_ready"
)

type RecallReplay struct {
	Service              string              `json:"service"`
	TraceID              string              `json:"trace_id"`
	PipelineVersion      string              `json:"pipeline_version"`
	ScoringConfigVersion string              `json:"scoring_config_version"`
	Query                RecallQuery         `json:"query"`
	EventCount           int                 `json:"event_count"`
	Events               []RecallReplayEvent `json:"events"`
	ProjectionInvariant  string              `json:"projection_invariant"`
	ReplayContract       string              `json:"replay_contract"`
}

type RecallReplayEvent struct {
	Index       int      `json:"index"`
	Stage       string   `json:"stage"`
	Kind        string   `json:"kind"`
	MemoryID    string   `json:"memory_id,omitempty"`
	SourceType  string   `json:"source_type,omitempty"`
	SessionID   string   `json:"session_id,omitempty"`
	AgentID     string   `json:"agent_id,omitempty"`
	ScopeID     string   `json:"scope_id,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	FinalScore  float64  `json:"final_score,omitempty"`
	WarningCode string   `json:"warning_code,omitempty"`
	Severity    string   `json:"severity,omitempty"`
	Details     []string `json:"details,omitempty"`
}

func BuildRecallReplay(trace RecallTrace) RecallReplay {
	replay := RecallReplay{
		Service:              "goncho",
		TraceID:              trace.TraceID,
		PipelineVersion:      trace.PipelineVersion,
		ScoringConfigVersion: trace.ScoringConfig.Version,
		Query:                trace.Query,
		ProjectionInvariant:  "no_projection_without_recall_trace",
		ReplayContract:       "deterministic_replay_from_recall_trace",
	}
	addEvent := func(event RecallReplayEvent) {
		event.Index = len(replay.Events) + 1
		replay.Events = append(replay.Events, event)
	}

	addEvent(RecallReplayEvent{
		Stage:   RecallReplayStageQuery,
		Kind:    RecallReplayKindQuery,
		Details: recallReplayQueryDetails(trace.Query),
	})
	for _, item := range trace.Candidates {
		addEvent(recallReplayCandidateEvent(RecallStageScore, RecallReplayKindCandidate, item))
	}
	for _, warning := range trace.Warnings {
		addEvent(RecallReplayEvent{
			Stage:       RecallReplayStageWarn,
			Kind:        RecallReplayKindWarning,
			WarningCode: warning.Code,
			Severity:    warning.Severity,
			Details:     recallReplayWarningDetails(warning),
		})
	}
	for i, item := range trace.Selected {
		event := recallReplayCandidateEvent(RecallStageSelect, RecallReplayKindSelected, item)
		event.Details = append([]string{fmt.Sprintf("rank=%d", i+1)}, event.Details...)
		addEvent(event)
	}
	for _, item := range trace.Rejected {
		event := recallReplayCandidateEvent(RecallStageSelect, RecallReplayKindRejected, ScoredRecallCandidate{
			Candidate: item.Candidate,
			Score:     item.Score,
		})
		event.Reason = item.Reason
		if len(item.WhyRejected) > 0 {
			event.Details = append(event.Details, "why="+strings.Join(item.WhyRejected, "; "))
		}
		addEvent(event)
	}
	addEvent(RecallReplayEvent{
		Stage: RecallStageProject,
		Kind:  RecallReplayKindProject,
		Details: []string{
			"trace_only=true",
			fmt.Sprintf("selected=%d", len(trace.Selected)),
			fmt.Sprintf("rejected=%d", len(trace.Rejected)),
			fmt.Sprintf("warnings=%d", len(trace.Warnings)),
		},
	})
	replay.EventCount = len(replay.Events)
	if replay.Events == nil {
		replay.Events = []RecallReplayEvent{}
	}
	return replay
}

func FormatRecallReplay(replay RecallReplay) string {
	var b strings.Builder
	fmt.Fprintln(&b, "Goncho recall replay")
	fmt.Fprintf(&b, "trace_id: %s\n", replay.TraceID)
	fmt.Fprintf(&b, "pipeline_version: %s\n", replay.PipelineVersion)
	fmt.Fprintf(&b, "scoring_config: %s\n", replay.ScoringConfigVersion)
	fmt.Fprintf(&b, "query: %s\n", replay.Query.Query)
	fmt.Fprintf(&b, "workspace: %s\n", replay.Query.WorkspaceID)
	fmt.Fprintf(&b, "peer: %s\n", replay.Query.Peer)
	if strings.TrimSpace(replay.Query.ScopeID) != "" {
		fmt.Fprintf(&b, "scope: %s\n", replay.Query.ScopeID)
	}
	fmt.Fprintf(&b, "events: %d\n", replay.EventCount)
	fmt.Fprintf(&b, "projection_invariant: %s\n", replay.ProjectionInvariant)
	fmt.Fprintf(&b, "replay_contract: %s\n", replay.ReplayContract)

	warnings := 0
	fmt.Fprintln(&b, "\nevents")
	if len(replay.Events) == 0 {
		fmt.Fprintln(&b, "  none")
	}
	for _, event := range replay.Events {
		fmt.Fprintf(&b, "  %d. %s", event.Index, event.Kind)
		if event.MemoryID != "" {
			fmt.Fprintf(&b, " memory_id=%s", event.MemoryID)
		}
		if event.Reason != "" {
			fmt.Fprintf(&b, " reason=%s", event.Reason)
		}
		if event.WarningCode != "" {
			warnings++
			fmt.Fprintf(&b, " code=%s", event.WarningCode)
		}
		if event.FinalScore != 0 {
			fmt.Fprintf(&b, " final=%.6f", event.FinalScore)
		}
		if event.SourceType != "" {
			fmt.Fprintf(&b, " source=%s", event.SourceType)
		}
		if event.SessionID != "" {
			fmt.Fprintf(&b, " session=%s", event.SessionID)
		}
		if event.ScopeID != "" {
			fmt.Fprintf(&b, " scope=%s", event.ScopeID)
		}
		if event.Severity != "" {
			fmt.Fprintf(&b, " severity=%s", event.Severity)
		}
		if len(event.Details) > 0 {
			fmt.Fprintf(&b, " %s", strings.Join(event.Details, " "))
		}
		fmt.Fprintln(&b)
	}
	if warnings == 0 {
		fmt.Fprintln(&b, "\nwarnings: none")
	}
	return b.String()
}

func recallReplayCandidateEvent(stage string, kind string, item ScoredRecallCandidate) RecallReplayEvent {
	details := []string{formatRecallDiagnosticScores(item.Score)}
	if len(item.Score.WhySelected) > 0 {
		details = append(details, "why="+strings.Join(item.Score.WhySelected, "; "))
	}
	if preview := previewRecallContent(item.Candidate.Content); preview != "" {
		details = append(details, "content="+preview)
	}
	return RecallReplayEvent{
		Stage:      stage,
		Kind:       kind,
		MemoryID:   item.Candidate.MemoryID,
		SourceType: item.Candidate.SourceType,
		SessionID:  item.Candidate.SessionID,
		AgentID:    item.Candidate.AgentID,
		ScopeID:    item.Candidate.ScopeID,
		FinalScore: item.Score.FinalScore,
		Details:    details,
	}
}

func recallReplayQueryDetails(query RecallQuery) []string {
	details := []string{
		"workspace=" + query.WorkspaceID,
		"peer=" + query.Peer,
		fmt.Sprintf("query=%q", query.Query),
	}
	if query.SessionKey != "" {
		details = append(details, "session="+query.SessionKey)
	}
	if query.ScopeID != "" {
		details = append(details, "scope="+query.ScopeID)
	}
	if query.Limit > 0 {
		details = append(details, fmt.Sprintf("limit=%d", query.Limit))
	}
	if query.MaxTokens > 0 {
		details = append(details, fmt.Sprintf("max_tokens=%d", query.MaxTokens))
	}
	return details
}

func recallReplayWarningDetails(warning RecallWarning) []string {
	details := []string{"stage=" + warning.Stage}
	if warning.Message != "" {
		details = append(details, fmt.Sprintf("message=%q", warning.Message))
	}
	return details
}
