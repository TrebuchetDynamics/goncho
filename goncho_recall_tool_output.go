package goncho

func buildGonchoRecallToolOutput(trace RecallTrace, compact bool) map[string]any {
	replay := BuildRecallReplay(trace)
	diagnostics := BuildRecallDiagnostics(trace)
	out := map[string]any{
		"action":             "recall",
		"trace_id":           trace.TraceID,
		"pipeline_version":   trace.PipelineVersion,
		"workspace_id":       trace.Query.WorkspaceID,
		"peer":               trace.Query.Peer,
		"query":              trace.Query.Query,
		"candidate_count":    len(trace.Candidates),
		"selected_count":     len(trace.Selected),
		"rejected_count":     len(trace.Rejected),
		"warning_count":      len(trace.Warnings),
		"diagnostics":        diagnostics,
		"replay_contract":    replay.ReplayContract,
		"projection_ready":   true,
		"projection_warning": "recall trace is orientation evidence; hosts must verify live state before acting",
	}
	if !compact {
		out["selected"] = trace.Selected
		out["warnings"] = trace.Warnings
		out["trace"] = trace
		out["replay"] = replay
		out["diagnostics_text"] = FormatRecallDiagnosticsReport(diagnostics)
	}
	return out
}
