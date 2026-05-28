package goncho

import (
	"strings"
	"testing"
	"time"
)

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
