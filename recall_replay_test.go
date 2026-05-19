package goncho

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRecallReplayBuildsDeterministicTimelineFromTrace(t *testing.T) {
	trace := RecallTrace{
		TraceID:         "trace-replay",
		PipelineVersion: "test-pipeline",
		Query:           RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "auth rate limit", ScopeID: "team", Limit: 2, MaxTokens: 64},
		ScoringConfig:   RecallScoringConfig{Version: "replay-v1", Weights: map[string]float64{"keyword": 0.6, "semantic": 0.4}, RRFK: 60, MMRLambda: 0.7, TokenBudget: 64},
		Candidates: []ScoredRecallCandidate{
			{
				Candidate: RecallCandidate{MemoryID: "mem-auth", SourceType: "conclusion", Content: "JWT auth uses jose middleware.", SessionID: "sess-auth", ScopeID: "team"},
				Score: RecallScore{
					KeywordScore:  0.8,
					SemanticScore: 0.9,
					FinalScore:    0.82,
					WhySelected:   []string{"final_score=0.820000", "scoring_config=replay-v1"},
				},
			},
			{
				Candidate: RecallCandidate{MemoryID: "mem-rate", SourceType: "turn", Content: "Rate limiting uses token bucket middleware.", SessionID: "sess-rate", ScopeID: "team"},
				Score: RecallScore{
					KeywordScore:     0.7,
					SemanticScore:    0.86,
					DiversityPenalty: 0.3,
					FinalScore:       0.42,
					WhySelected:      []string{"final_score=0.720000", "scoring_config=replay-v1"},
				},
			},
		},
		Selected: []ScoredRecallCandidate{{
			Candidate: RecallCandidate{MemoryID: "mem-auth", SourceType: "conclusion", Content: "JWT auth uses jose middleware.", SessionID: "sess-auth", ScopeID: "team"},
			Score:     RecallScore{KeywordScore: 0.8, SemanticScore: 0.9, FinalScore: 0.82, WhySelected: []string{"final_score=0.820000", "scoring_config=replay-v1"}},
		}},
		Rejected: []RejectedRecallCandidate{{
			Candidate:   RecallCandidate{MemoryID: "mem-rate", SourceType: "turn", Content: "Rate limiting uses token bucket middleware.", SessionID: "sess-rate", ScopeID: "team"},
			Score:       RecallScore{KeywordScore: 0.7, SemanticScore: 0.86, DiversityPenalty: 0.3, FinalScore: 0.42},
			Reason:      RecallRejectNotSelected,
			WhyRejected: []string{"limit=2"},
		}},
		Warnings: []RecallWarning{{
			Code:     RecallWarningTokenBudgetTruncated,
			Stage:    RecallStageSelect,
			Severity: RecallWarningDegraded,
			Message:  "token budget truncated selected recall context",
		}},
	}

	replay := BuildRecallReplay(trace)
	if replay.Service != "goncho" || replay.TraceID != "trace-replay" || replay.PipelineVersion != "test-pipeline" || replay.ScoringConfigVersion != "replay-v1" {
		t.Fatalf("replay header = %+v", replay)
	}
	if replay.ProjectionInvariant != "no_projection_without_recall_trace" {
		t.Fatalf("ProjectionInvariant = %q", replay.ProjectionInvariant)
	}
	if replay.ReplayContract != "deterministic_replay_from_recall_trace" {
		t.Fatalf("ReplayContract = %q", replay.ReplayContract)
	}
	if replay.EventCount != len(replay.Events) || replay.EventCount != 7 {
		t.Fatalf("EventCount = %d len(events)=%d", replay.EventCount, len(replay.Events))
	}
	for i, event := range replay.Events {
		if event.Index != i+1 {
			t.Fatalf("event[%d].Index = %d, want %d", i, event.Index, i+1)
		}
	}
	assertRecallReplayEvent(t, replay.Events[0], "query", "recall_query", "")
	assertRecallReplayEvent(t, replay.Events[1], "score", "candidate_scored", "mem-auth")
	assertRecallReplayEvent(t, replay.Events[2], "score", "candidate_scored", "mem-rate")
	assertRecallReplayEvent(t, replay.Events[3], "warn", "warning", "")
	assertRecallReplayEvent(t, replay.Events[4], "select", "selected", "mem-auth")
	assertRecallReplayEvent(t, replay.Events[5], "select", "rejected", "mem-rate")
	assertRecallReplayEvent(t, replay.Events[6], "project", "projection_ready", "")
	if replay.Events[3].WarningCode != RecallWarningTokenBudgetTruncated || replay.Events[3].Severity != RecallWarningDegraded {
		t.Fatalf("warning event = %+v", replay.Events[3])
	}
	if replay.Events[5].Reason != RecallRejectNotSelected {
		t.Fatalf("rejected event = %+v", replay.Events[5])
	}
	if replay.Events[6].Details[0] != "trace_only=true" {
		t.Fatalf("projection event details = %+v", replay.Events[6].Details)
	}

	raw1, err := json.MarshalIndent(replay, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw2, err := json.MarshalIndent(BuildRecallReplay(trace), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw1) != string(raw2) {
		t.Fatalf("replay JSON is not deterministic:\n%s\n---\n%s", raw1, raw2)
	}

	text := FormatRecallReplay(replay)
	for _, want := range []string{
		"Goncho recall replay",
		"trace_id: trace-replay",
		"events: 7",
		"candidate_scored memory_id=mem-auth",
		"selected memory_id=mem-auth",
		"rejected memory_id=mem-rate reason=not_selected",
		"warning code=token_budget_truncated",
		"projection_ready trace_only=true",
		"projection_invariant: no_projection_without_recall_trace",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("formatted replay missing %q:\n%s", want, text)
		}
	}
}

func assertRecallReplayEvent(t *testing.T, event RecallReplayEvent, stage string, kind string, memoryID string) {
	t.Helper()
	if event.Stage != stage || event.Kind != kind || event.MemoryID != memoryID {
		t.Fatalf("event = %+v, want stage=%s kind=%s memory_id=%s", event, stage, kind, memoryID)
	}
}
