package goncho

import "testing"

func TestBuildGonchoRecallToolOutputShapesCompactAndFullPayloads(t *testing.T) {
	trace := RecallTrace{
		TraceID:         "trace-output-1",
		PipelineVersion: "tool-output-v1",
		Query: RecallQuery{
			WorkspaceID: "workspace-output",
			Peer:        "peer-output",
			Query:       "output diagnostics",
		},
		Candidates: []ScoredRecallCandidate{{
			Candidate: RecallCandidate{MemoryID: "mem-output-1", SourceType: "conclusion", Content: "Recall output should keep diagnostics stable."},
			Score:     RecallScore{FinalScore: 0.9},
		}},
		Selected: []ScoredRecallCandidate{{
			Candidate: RecallCandidate{MemoryID: "mem-output-1", SourceType: "conclusion", Content: "Recall output should keep diagnostics stable."},
			Score:     RecallScore{FinalScore: 0.9},
		}},
		Warnings: []RecallWarning{{Code: RecallWarningGraphDisabled, Stage: RecallStageGenerate, Severity: RecallWarningInfo}},
	}

	full := buildGonchoRecallToolOutput(trace, false)
	if full["trace_id"] != "trace-output-1" || full["replay_contract"] != "deterministic_replay_from_recall_trace" {
		t.Fatalf("full output = %+v, want trace identity and replay contract", full)
	}
	for _, included := range []string{"selected", "warnings", "trace", "replay", "diagnostics", "diagnostics_text"} {
		if _, ok := full[included]; !ok {
			t.Fatalf("full output missing %q: %+v", included, full)
		}
	}

	compact := buildGonchoRecallToolOutput(trace, true)
	if compact["trace_id"] != "trace-output-1" || compact["warning_count"] != 1 || compact["selected_count"] != 1 {
		t.Fatalf("compact output = %+v, want stable counts and trace identity", compact)
	}
	if _, ok := compact["diagnostics"]; !ok {
		t.Fatalf("compact output missing diagnostics: %+v", compact)
	}
	for _, omitted := range []string{"selected", "warnings", "trace", "replay", "diagnostics_text"} {
		if _, ok := compact[omitted]; ok {
			t.Fatalf("compact output included %q: %+v", omitted, compact)
		}
	}
}
