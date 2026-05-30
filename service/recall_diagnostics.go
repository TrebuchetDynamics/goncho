package goncho

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/recalldiag"
)

type RecallDiagnosticsReport struct {
	Service              string                       `json:"service"`
	Status               string                       `json:"status"`
	TraceID              string                       `json:"trace_id"`
	PipelineVersion      string                       `json:"pipeline_version"`
	Query                RecallQuery                  `json:"query"`
	ScoringConfig        RecallScoringConfig          `json:"scoring_config"`
	CandidateCount       int                          `json:"candidate_count"`
	SelectedCount        int                          `json:"selected_count"`
	RejectedCount        int                          `json:"rejected_count"`
	WarningCount         int                          `json:"warning_count"`
	Selected             []RecallDiagnosticsCandidate `json:"selected"`
	Rejected             []RecallDiagnosticsRejection `json:"rejected"`
	Warnings             []RecallWarning              `json:"warnings"`
	ProjectionInvariant  string                       `json:"projection_invariant"`
	DegradedPathContract string                       `json:"degraded_path_contract"`
}

type RecallDiagnosticsCandidate struct {
	MemoryID       string      `json:"memory_id"`
	SourceType     string      `json:"source_type,omitempty"`
	SessionID      string      `json:"session_id,omitempty"`
	AgentID        string      `json:"agent_id,omitempty"`
	ScopeID        string      `json:"scope_id,omitempty"`
	ContentPreview string      `json:"content_preview,omitempty"`
	FinalScore     float64     `json:"final_score"`
	Scores         RecallScore `json:"scores"`
	WhySelected    []string    `json:"why_selected,omitempty"`
}

type RecallDiagnosticsRejection struct {
	MemoryID       string      `json:"memory_id"`
	SourceType     string      `json:"source_type,omitempty"`
	SessionID      string      `json:"session_id,omitempty"`
	AgentID        string      `json:"agent_id,omitempty"`
	ScopeID        string      `json:"scope_id,omitempty"`
	ContentPreview string      `json:"content_preview,omitempty"`
	Reason         string      `json:"reason"`
	FinalScore     float64     `json:"final_score"`
	Scores         RecallScore `json:"scores"`
	WhyRejected    []string    `json:"why_rejected,omitempty"`
}

func DecodeRecallTraceJSON(r io.Reader) (RecallTrace, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return RecallTrace{}, err
	}
	var trace RecallTrace
	if err := json.Unmarshal(raw, &trace); err != nil {
		return RecallTrace{}, err
	}
	if strings.TrimSpace(trace.TraceID) == "" {
		return RecallTrace{}, fmt.Errorf("recall trace missing trace_id")
	}
	if strings.TrimSpace(trace.PipelineVersion) == "" {
		return RecallTrace{}, fmt.Errorf("recall trace %s missing pipeline_version", trace.TraceID)
	}
	if strings.TrimSpace(trace.ScoringConfig.Version) == "" {
		return RecallTrace{}, fmt.Errorf("recall trace %s missing scoring_config.version", trace.TraceID)
	}
	return trace, nil
}

func BuildRecallDiagnostics(trace RecallTrace) RecallDiagnosticsReport {
	status := "ok"
	if len(trace.Warnings) > 0 {
		status = "degraded"
	}
	report := RecallDiagnosticsReport{
		Service:              "goncho",
		Status:               status,
		TraceID:              trace.TraceID,
		PipelineVersion:      trace.PipelineVersion,
		Query:                trace.Query,
		ScoringConfig:        trace.ScoringConfig,
		CandidateCount:       len(trace.Candidates),
		SelectedCount:        len(trace.Selected),
		RejectedCount:        len(trace.Rejected),
		WarningCount:         len(trace.Warnings),
		Warnings:             append([]RecallWarning(nil), trace.Warnings...),
		ProjectionInvariant:  "no_projection_without_recall_trace",
		DegradedPathContract: "all_degraded_paths_emit_code_first_warnings",
	}
	for _, item := range trace.Selected {
		report.Selected = append(report.Selected, recallDiagnosticsCandidate(item))
	}
	for _, item := range trace.Rejected {
		report.Rejected = append(report.Rejected, recallDiagnosticsRejection(item))
	}
	if report.Selected == nil {
		report.Selected = []RecallDiagnosticsCandidate{}
	}
	if report.Rejected == nil {
		report.Rejected = []RecallDiagnosticsRejection{}
	}
	if report.Warnings == nil {
		report.Warnings = []RecallWarning{}
	}
	return report
}

func FormatRecallDiagnosticsReport(report RecallDiagnosticsReport) string {
	var b strings.Builder
	fmt.Fprintln(&b, "Goncho recall diagnostics")
	fmt.Fprintf(&b, "status: %s\n", report.Status)
	fmt.Fprintf(&b, "trace_id: %s\n", report.TraceID)
	fmt.Fprintf(&b, "pipeline_version: %s\n", report.PipelineVersion)
	fmt.Fprintf(&b, "scoring_config: %s\n", report.ScoringConfig.Version)
	fmt.Fprintf(&b, "query: %s\n", report.Query.Query)
	fmt.Fprintf(&b, "workspace: %s\n", report.Query.WorkspaceID)
	fmt.Fprintf(&b, "peer: %s\n", report.Query.Peer)
	if strings.TrimSpace(report.Query.ScopeID) != "" {
		fmt.Fprintf(&b, "scope: %s\n", report.Query.ScopeID)
	}
	fmt.Fprintf(&b, "candidates: total=%d selected=%d rejected=%d warnings=%d\n", report.CandidateCount, report.SelectedCount, report.RejectedCount, report.WarningCount)
	fmt.Fprintf(&b, "projection_invariant: %s\n", report.ProjectionInvariant)
	fmt.Fprintf(&b, "degraded_path_contract: %s\n", report.DegradedPathContract)

	fmt.Fprintln(&b, "\nselected candidates")
	if len(report.Selected) == 0 {
		fmt.Fprintln(&b, "  none")
	}
	for i, item := range report.Selected {
		fmt.Fprintf(&b, "  %d. %s source=%s session=%s scope=%s final=%.6f\n", i+1, item.MemoryID, item.SourceType, item.SessionID, item.ScopeID, item.FinalScore)
		fmt.Fprintf(&b, "     %s\n", formatRecallDiagnosticScores(item.Scores))
		if len(item.WhySelected) > 0 {
			fmt.Fprintf(&b, "     why: %s\n", strings.Join(item.WhySelected, "; "))
		}
		if item.ContentPreview != "" {
			fmt.Fprintf(&b, "     content: %s\n", item.ContentPreview)
		}
	}

	fmt.Fprintln(&b, "\nrejected candidates")
	if len(report.Rejected) == 0 {
		fmt.Fprintln(&b, "  none")
	}
	for _, item := range report.Rejected {
		fmt.Fprintf(&b, "  - %s reason=%s source=%s session=%s scope=%s final=%.6f\n", item.MemoryID, item.Reason, item.SourceType, item.SessionID, item.ScopeID, item.FinalScore)
		fmt.Fprintf(&b, "    %s\n", formatRecallDiagnosticScores(item.Scores))
		if len(item.WhyRejected) > 0 {
			fmt.Fprintf(&b, "    why: %s\n", strings.Join(item.WhyRejected, "; "))
		}
	}

	if len(report.Warnings) == 0 {
		fmt.Fprintln(&b, "\nwarnings: none")
		return b.String()
	}
	fmt.Fprintln(&b, "\nwarnings")
	for _, warning := range report.Warnings {
		fmt.Fprintf(&b, "  - %s stage=%s severity=%s", warning.Code, warning.Stage, warning.Severity)
		if strings.TrimSpace(warning.Message) != "" {
			fmt.Fprintf(&b, " message=%q", warning.Message)
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}

func formatRecallDiagnosticScores(scores RecallScore) string {
	return recalldiag.FormatScores(recalldiag.ScoreBreakdown{
		KeywordScore:     scores.KeywordScore,
		SemanticScore:    scores.SemanticScore,
		GraphScore:       scores.GraphScore,
		FactScore:        scores.FactScore,
		RecencyScore:     scores.RecencyScore,
		ImportanceScore:  scores.ImportanceScore,
		ScopeScore:       scores.ScopeScore,
		RRFScore:         scores.RRFScore,
		DiversityPenalty: scores.DiversityPenalty,
	})
}

func recallDiagnosticsCandidate(item ScoredRecallCandidate) RecallDiagnosticsCandidate {
	return RecallDiagnosticsCandidate{
		MemoryID:       item.Candidate.MemoryID,
		SourceType:     item.Candidate.SourceType,
		SessionID:      item.Candidate.SessionID,
		AgentID:        item.Candidate.AgentID,
		ScopeID:        item.Candidate.ScopeID,
		ContentPreview: previewRecallContent(item.Candidate.Content),
		FinalScore:     item.Score.FinalScore,
		Scores:         item.Score,
		WhySelected:    cloneStrings(item.Score.WhySelected),
	}
}

func recallDiagnosticsRejection(item RejectedRecallCandidate) RecallDiagnosticsRejection {
	return RecallDiagnosticsRejection{
		MemoryID:       item.Candidate.MemoryID,
		SourceType:     item.Candidate.SourceType,
		SessionID:      item.Candidate.SessionID,
		AgentID:        item.Candidate.AgentID,
		ScopeID:        item.Candidate.ScopeID,
		ContentPreview: previewRecallContent(item.Candidate.Content),
		Reason:         item.Reason,
		FinalScore:     item.Score.FinalScore,
		Scores:         item.Score,
		WhyRejected:    cloneStrings(item.WhyRejected),
	}
}

func previewRecallContent(content string) string { return recalldiag.PreviewContent(content) }
