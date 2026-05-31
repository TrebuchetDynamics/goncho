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
				Candidate: RecallCandidate{MemoryID: "mem-auth", SourceType: "conclusion", Content: "JWT auth uses jose middleware.", SessionID: "sess-auth", AgentID: "agent-auth", ScopeID: "team"},
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
			Candidate: RecallCandidate{MemoryID: "mem-auth", SourceType: "conclusion", Content: "JWT auth uses jose middleware.", SessionID: "sess-auth", AgentID: "agent-auth", ScopeID: "team"},
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
	if replay.ReplayFingerprint == "" {
		t.Fatal("ReplayFingerprint is empty")
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
		"agent=agent-auth",
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

func TestFormatRecallReplayIncludesZeroFinalScoresForCandidates(t *testing.T) {
	trace := RecallTrace{
		TraceID:         "trace-zero-score",
		PipelineVersion: "test-pipeline",
		Query:           RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "unmatched query"},
		Candidates: []ScoredRecallCandidate{{
			Candidate: RecallCandidate{MemoryID: "mem-zero", SourceType: "turn", Content: "Candidate with no score evidence yet."},
			Score:     RecallScore{FinalScore: 0},
		}},
	}

	text := FormatRecallReplay(BuildRecallReplay(trace))
	if !strings.Contains(text, "candidate_scored memory_id=mem-zero final=0.000000") {
		t.Fatalf("formatted replay dropped explicit zero final score:\n%s", text)
	}
}

func TestRecallReplayFingerprintTreatsNilAndEmptyTraceSlicesEqually(t *testing.T) {
	base := RecallTrace{
		TraceID:         "trace-empty-slices",
		PipelineVersion: "test-pipeline",
		Query:           RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "nothing matched"},
		ScoringConfig:   RecallScoringConfig{Version: "empty-slices-v1"},
	}
	empty := base
	empty.Candidates = []ScoredRecallCandidate{}
	empty.Selected = []ScoredRecallCandidate{}
	empty.Rejected = []RejectedRecallCandidate{}
	empty.Warnings = []RecallWarning{}

	baseReplay := BuildRecallReplay(base)
	emptyReplay := BuildRecallReplay(empty)
	if baseReplay.ReplayFingerprint != emptyReplay.ReplayFingerprint {
		t.Fatalf("ReplayFingerprint differs for nil vs empty replay slices: nil=%s empty=%s", baseReplay.ReplayFingerprint, emptyReplay.ReplayFingerprint)
	}
}

func TestBuildRecallReplaySnapshotsQuerySources(t *testing.T) {
	trace := RecallTrace{
		TraceID:         "trace-query-sources",
		PipelineVersion: "test-pipeline",
		Query:           RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "auth", Sources: []string{"memory", "vector"}},
	}

	replay := BuildRecallReplay(trace)
	trace.Query.Sources[0] = "mutated-after-build"

	if got := replay.Query.Sources; len(got) != 2 || got[0] != "memory" || got[1] != "vector" {
		t.Fatalf("replay query sources = %v, want immutable replay snapshot", got)
	}
}

func TestRecallReplayWarningEventsIncludeStableEvidence(t *testing.T) {
	trace := RecallTrace{
		TraceID:         "trace-warning-evidence",
		PipelineVersion: "test-pipeline",
		Query:           RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "auth"},
		Warnings: []RecallWarning{
			{
				Code:     RecallWarningSemanticUnavailable,
				Stage:    RecallStageGenerate,
				Severity: RecallWarningDegraded,
				Message:  "semantic generator unavailable",
				Evidence: map[string]string{"generator": "vector", "error": "timeout"},
			},
			{
				Code:     RecallWarningSemanticUnavailable,
				Stage:    RecallStageGenerate,
				Severity: RecallWarningDegraded,
				Message:  "semantic generator unavailable",
				Evidence: map[string]string{"generator": "graph", "error": "timeout"},
			},
		},
	}

	replay := BuildRecallReplay(trace)
	if replay.EventCount != 4 {
		t.Fatalf("EventCount = %d, want query, two warnings, project", replay.EventCount)
	}
	if got := replay.Events[1].Details; !slicesContainsInOrder(got, []string{"stage=generate", "message=\"semantic generator unavailable\"", "evidence.error=\"timeout\"", "evidence.generator=\"vector\""}) {
		t.Fatalf("first warning details = %+v, want stable evidence fields", got)
	}
	if got := replay.Events[2].Details; !slicesContainsInOrder(got, []string{"stage=generate", "message=\"semantic generator unavailable\"", "evidence.error=\"timeout\"", "evidence.generator=\"graph\""}) {
		t.Fatalf("second warning details = %+v, want distinct stable evidence fields", got)
	}

	text := FormatRecallReplay(replay)
	for _, want := range []string{"evidence.generator=\"vector\"", "evidence.generator=\"graph\""} {
		if !strings.Contains(text, want) {
			t.Fatalf("formatted replay missing %q:\n%s", want, text)
		}
	}
}

func slicesContainsInOrder(got []string, want []string) bool {
	if len(got) < len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func assertRecallReplayEvent(t *testing.T, event RecallReplayEvent, stage string, kind string, memoryID string) {
	t.Helper()
	if event.Stage != stage || event.Kind != kind || event.MemoryID != memoryID {
		t.Fatalf("event = %+v, want stage=%s kind=%s memory_id=%s", event, stage, kind, memoryID)
	}
}
