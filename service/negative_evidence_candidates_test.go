package goncho

import (
	"strings"
	"testing"
	"time"
)

func TestNegativeEvidenceObservationQueryCapsCallerLimitAtScanGuard(t *testing.T) {
	q := negativeEvidenceObservationQuery(ObservationQuery{Limit: negativeEvidenceObservationScanLimit + 1})
	if q.Limit != negativeEvidenceObservationScanLimit {
		t.Fatalf("limit = %d, want scan guard %d", q.Limit, negativeEvidenceObservationScanLimit)
	}
}

func TestNegativeEvidenceCandidatesTrimScopeAndToolBeforeBucketing(t *testing.T) {
	failed := false
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: " gormes "}),
		MinFailures: 2,
		Observations: []Observation{
			{ID: "fail-1", Kind: ObservationKindToolError, ProfileID: " mineru ", PeerID: " peer ", SessionKey: " sess ", Success: &failed, Metadata: map[string]string{"tool_name": " Bash "}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: "fail-2", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", PeerID: "peer", SessionKey: "sess", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(20, 0).UTC()},
		},
	})
	if len(candidates) != 1 {
		t.Fatalf("candidates = %+v, want one normalized scope/tool bucket", candidates)
	}
	candidate := candidates[0]
	if candidate.WorkspaceID != "gormes" || candidate.ProfileID != "mineru" || candidate.PeerID != "peer" || candidate.SessionKey != "sess" || candidate.ToolName != "bash" || candidate.FailureCount != 2 {
		t.Fatalf("candidate = %+v, want trimmed scope and lowercase tool with two failures", candidate)
	}
}

func TestNegativeEvidenceCandidatesDoNotCollapseDelimiterBearingDimensions(t *testing.T) {
	failed := false
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: "gormes"}),
		MinFailures: 2,
		Observations: []Observation{
			{ID: "left-1", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "a\x00b", PeerID: "c", SessionKey: "sess", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: "left-2", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "a\x00b", PeerID: "c", SessionKey: "sess", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(11, 0).UTC()},
			{ID: "right-1", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "a", PeerID: "b\x00c", SessionKey: "sess", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(12, 0).UTC()},
			{ID: "right-2", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "a", PeerID: "b\x00c", SessionKey: "sess", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(13, 0).UTC()},
		},
	})
	if len(candidates) != 2 {
		t.Fatalf("candidates = %+v, want delimiter-bearing profile/peer dimensions kept distinct", candidates)
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate.FailureCount != 2 {
			t.Fatalf("candidate = %+v, want each dimension bucket to keep its own two failures", candidate)
		}
		seen[candidate.ProfileID+"|"+candidate.PeerID] = true
	}
	if !seen["a\x00b|c"] || !seen["a|b\x00c"] {
		t.Fatalf("candidates = %+v, want both delimiter-bearing dimensions", candidates)
	}
}

func TestNegativeEvidenceCandidatesBuildFailureSignalFromCustomKindFallback(t *testing.T) {
	failed := false
	observedAt := time.Unix(10, 0).UTC()
	signal, ok := negativeEvidenceFailureSignalFrom(
		ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: " gormes "}),
		Observation{
			ID:         " custom-fail ",
			Kind:       ObservationKindCustom,
			ProfileID:  " mineru ",
			PeerID:     " peer ",
			SessionKey: " sess ",
			Success:    &failed,
			Metadata:   map[string]string{"custom_kind": " Retrieval Planner "},
			ObservedAt: observedAt,
		},
	)
	if !ok {
		t.Fatalf("failure signal not built for replayable custom failure")
	}
	if signal.EvidenceID != "custom-fail" || signal.Scope.WorkspaceID != "gormes" || signal.Scope.ProfileID != "mineru" || signal.Scope.PeerID != "peer" || signal.Scope.SessionKey != "sess" || signal.Scope.ToolName != "retrieval planner" || !signal.ObservedAt.Equal(observedAt) {
		t.Fatalf("signal = %+v, want replay id, normalized scope, custom_kind fallback, and observed time", signal)
	}
}

func TestNegativeEvidenceCandidatesTreatReplayIDsPerCandidateScope(t *testing.T) {
	failed := false
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: "gormes"}),
		MinFailures: 2,
		Observations: []Observation{
			{ID: "shared-fail-id", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: "mineru-second", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(20, 0).UTC()},
			{ID: "shared-fail-id", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "yunobo", SessionKey: "sess-b", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(30, 0).UTC()},
			{ID: "yunobo-second", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "yunobo", SessionKey: "sess-b", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(40, 0).UTC()},
		},
	})
	if len(candidates) != 2 {
		t.Fatalf("candidates = %+v, want replay IDs deduped only within each scoped candidate bucket", candidates)
	}
	for _, candidate := range candidates {
		if candidate.FailureCount != 2 {
			t.Fatalf("candidate = %+v, want each scoped bucket to retain two unique failures", candidate)
		}
	}
}

func TestNegativeEvidenceCandidatesDoNotPromoteReplayedObservationID(t *testing.T) {
	failed := false
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: "gormes"}),
		MinFailures: 2,
		Observations: []Observation{
			{ID: "replayed-fail", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: "replayed-fail", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(20, 0).UTC()},
		},
	})
	if len(candidates) != 0 {
		t.Fatalf("candidates = %+v, want duplicate observation id not to become repeated-failure evidence", candidates)
	}
}

func TestNegativeEvidenceCandidatesRequireReplayableObservationIDs(t *testing.T) {
	failed := false
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: "gormes"}),
		MinFailures: 2,
		Observations: []Observation{
			{Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: " ", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(20, 0).UTC()},
			{ID: "replayable-fail", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(30, 0).UTC()},
		},
	})
	if len(candidates) != 0 {
		t.Fatalf("candidates = %+v, want unreplayable failures without observation ids not to create negative-memory candidates", candidates)
	}
}

func TestNegativeEvidenceCandidatesIgnoreExplicitlySuccessfulToolErrors(t *testing.T) {
	succeeded := true
	failed := false
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: "gormes"}),
		MinFailures: 2,
		Observations: []Observation{
			{ID: "contradictory-success", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &succeeded, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: "actual-failure", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(20, 0).UTC()},
		},
	})
	if len(candidates) != 0 {
		t.Fatalf("candidates = %+v, want explicit success=true not to promote contradictory tool_error into repeated-failure evidence", candidates)
	}
}

func TestNegativeEvidenceCandidatesDoNotPromoteFailuresResolvedByLaterSuccess(t *testing.T) {
	failed := false
	succeeded := true
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: "gormes"}),
		MinFailures: 2,
		Observations: []Observation{
			{ID: "fail-before-success-1", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", PeerID: "peer", SessionKey: "sess", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: "fail-before-success-2", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", PeerID: "peer", SessionKey: "sess", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(20, 0).UTC()},
			{ID: "later-success", Kind: ObservationKindToolResult, WorkspaceID: "gormes", ProfileID: "mineru", PeerID: "peer", SessionKey: "sess", Success: &succeeded, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(30, 0).UTC()},
		},
	})
	if len(candidates) != 0 {
		t.Fatalf("candidates = %+v, want later scoped success to suppress resolved repeated-failure candidate", candidates)
	}
}

func TestNegativeEvidenceCandidatesStillPromoteFailuresAfterLastSuccess(t *testing.T) {
	failed := false
	succeeded := true
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: "gormes"}),
		MinFailures: 2,
		Observations: []Observation{
			{ID: "early-fail", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", PeerID: "peer", SessionKey: "sess", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: "middle-success", Kind: ObservationKindToolResult, WorkspaceID: "gormes", ProfileID: "mineru", PeerID: "peer", SessionKey: "sess", Success: &succeeded, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(20, 0).UTC()},
			{ID: "late-fail-1", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", PeerID: "peer", SessionKey: "sess", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(30, 0).UTC()},
			{ID: "late-fail-2", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", PeerID: "peer", SessionKey: "sess", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(40, 0).UTC()},
		},
	})
	if len(candidates) != 1 || candidates[0].FailureCount != 2 {
		t.Fatalf("candidates = %+v, want only repeated failures after the latest scoped success", candidates)
	}
	if got := strings.Join(candidates[0].EvidenceIDs, ","); got != "late-fail-1,late-fail-2" {
		t.Fatalf("evidence ids = %q, want only post-success failures", got)
	}
}

func TestNegativeEvidenceCandidatesStillTreatImplicitToolErrorsAsFailures(t *testing.T) {
	failed := false
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: "gormes"}),
		MinFailures: 2,
		Observations: []Observation{
			{ID: "implicit-error", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: "explicit-failure", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(20, 0).UTC()},
		},
	})
	if len(candidates) != 1 || candidates[0].FailureCount != 2 {
		t.Fatalf("candidates = %+v, want implicit tool_error compatibility preserved", candidates)
	}
}

func TestNegativeEvidenceCandidatesNormalizeToolNameCase(t *testing.T) {
	failed := false
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: "gormes"}),
		MinFailures: 2,
		Observations: []Observation{
			{ID: "mixed-case-tool", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "Bash"}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: "lower-case-tool", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(20, 0).UTC()},
		},
	})
	if len(candidates) != 1 {
		t.Fatalf("candidates = %+v, want tool_name case variants grouped as one repeated-failure candidate", candidates)
	}
	if candidates[0].ToolName != "bash" || candidates[0].FailureCount != 2 {
		t.Fatalf("candidate = %+v, want normalized bash candidate with two failures", candidates[0])
	}
}

func TestNegativeEvidenceCandidatesNormalizeToolNameWhitespace(t *testing.T) {
	failed := false
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: "gormes"}),
		MinFailures: 2,
		Observations: []Observation{
			{ID: "spaced-tool", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "Retrieval   Planner"}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: "newline-tool", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": " retrieval\nplanner "}, ObservedAt: time.Unix(20, 0).UTC()},
		},
	})
	if len(candidates) != 1 {
		t.Fatalf("candidates = %+v, want whitespace-equivalent tool names grouped as one repeated-failure candidate", candidates)
	}
	if candidates[0].ToolName != "retrieval planner" || candidates[0].FailureCount != 2 {
		t.Fatalf("candidate = %+v, want normalized retrieval planner candidate with two failures", candidates[0])
	}
}

func TestNegativeEvidenceCandidatesOrderEvidenceByFailureTimeline(t *testing.T) {
	failed := false
	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{
		Projection:  ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: "gormes"}),
		MinFailures: 2,
		Observations: []Observation{
			{ID: "z-first", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()},
			{ID: "a-second", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(20, 0).UTC()},
		},
	})
	if len(candidates) != 1 {
		t.Fatalf("candidates = %+v, want one repeated failure", candidates)
	}
	if got := strings.Join(candidates[0].EvidenceIDs, ","); got != "z-first,a-second" {
		t.Fatalf("evidence ids = %q, want chronological failure timeline", got)
	}
}

func TestNegativeEvidenceCandidatesMineRepeatedFailuresWithoutRawContent(t *testing.T) {
	failed := false
	projection := ProjectSessionEvidence(SessionEvidenceInput{
		WorkspaceID:    "gormes",
		SessionIndexes: []SessionEvidenceIndex{{Scope: SessionEvidenceScopeProfile, ProfileID: "mineru", SessionCount: 1, LineageCount: 1}},
	})
	observations := []Observation{
		{ID: "obs-1", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Input: "secret failing command", Output: "private stack trace", Metadata: map[string]string{"tool_name": "bash", "hook_event": "tool_failure"}, ObservedAt: time.Unix(10, 0).UTC()},
		{ID: "obs-2", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "mineru", SessionKey: "sess-a", Success: &failed, Input: "secret retry command", Output: "private retry stack", Metadata: map[string]string{"tool_name": "bash", "hook_event": "tool_failure"}, ObservedAt: time.Unix(20, 0).UTC()},
		{ID: "obs-3", Kind: ObservationKindToolError, WorkspaceID: "gormes", ProfileID: "yunobo", SessionKey: "sess-b", Success: &failed, Metadata: map[string]string{"tool_name": "curl", "hook_event": "tool_failure"}, ObservedAt: time.Unix(30, 0).UTC()},
	}

	candidates := GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{Projection: projection, Observations: observations, MinFailures: 2})
	if len(candidates) != 1 {
		t.Fatalf("candidates = %+v, want one repeated mineru/bash failure", candidates)
	}
	candidate := candidates[0]
	if candidate.Kind != NegativeEvidenceRepeatedToolFailure || candidate.ProfileID != "mineru" || candidate.SessionKey != "sess-a" || candidate.ToolName != "bash" || candidate.FailureCount != 2 {
		t.Fatalf("candidate = %+v", candidate)
	}
	if got := strings.Join(candidate.EvidenceIDs, ","); got != "obs-1,obs-2" {
		t.Fatalf("evidence ids = %q", got)
	}
	serialized := candidate.String()
	for _, leaked := range []string{"secret failing command", "private stack trace", "secret retry command"} {
		if strings.Contains(serialized, leaked) {
			t.Fatalf("candidate leaked raw observation content %q in %s", leaked, serialized)
		}
	}
	if !strings.Contains(candidate.Recommendation, "negative memory") || !strings.Contains(candidate.Recommendation, "verify live state") {
		t.Fatalf("recommendation = %q, want negative-memory/live-state guidance", candidate.Recommendation)
	}
}
