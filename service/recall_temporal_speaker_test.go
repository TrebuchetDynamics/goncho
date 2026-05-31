package goncho

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

func TestRecallCurrentTruthIntentTokenization(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "leading now", query: "Now, who owns component A-17?", want: true},
		{name: "trailing now", query: "Who owns component A-17 now?", want: true},
		{name: "latest", query: "What is the latest owner for component A-17?", want: true},
		{name: "not substring", query: "What knowledge exists for component A-17?", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := recallQueryAsksCurrentTruth(tt.query); got != tt.want {
				t.Fatalf("recallQueryAsksCurrentTruth(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestRecallTemporalRoutingDoesNotWarnOnNegatedSupersededMarker(t *testing.T) {
	candidate := ScoredRecallCandidate{Candidate: RecallCandidate{
		MemoryID:   "mem-current",
		Provenance: []EvidenceItem{{Kind: "temporal", Note: "current_fact valid_now not_superseded"}},
	}}

	if got := recallTemporalAdjustment(candidate, "who owns component A-17 now?"); got != recallTemporalCurrentBonus {
		t.Fatalf("recallTemporalAdjustment() = %v, want current bonus for not_superseded current fact", got)
	}
	if recallHasSupersededEvidence([]ScoredRecallCandidate{candidate}) {
		t.Fatalf("recallHasSupersededEvidence() = true, want false for negated superseded marker")
	}
}

func TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-owner-old",
			Content:    "Mira owns component A-17.",
			ScopeID:    "team",
			CreatedAt:  now.Add(-48 * time.Hour),
			Importance: 0.95,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 1.00, Note: "matched component owner"},
				{Kind: "temporal", Score: 0.10, Note: "superseded_by=mem-owner-current"},
			},
		},
		{
			MemoryID:   "mem-owner-current",
			Content:    "Nadia now owns component A-17.",
			ScopeID:    "team",
			CreatedAt:  now.Add(-2 * time.Hour),
			Importance: 0.70,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.86, Note: "matched component owner"},
				{Kind: "temporal", Score: 1.00, Note: "current_fact"},
			},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "temporal-routing-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:       "temporal-routing-test-v1",
			Weights:       map[string]float64{"keyword": 0.65, "recency": 0.10, "importance": 0.15, "scope": 0.10},
			RRFK:          60,
			MMRLambda:     0.70,
			DiversityKeys: []string{"memory_id"},
			TokenBudget:   120,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "Who owns component A-17 now?",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-owner-current"}) {
		t.Fatalf("selected IDs = %v, want current owner", selectedRecallIDs(trace))
	}
	if !recallTraceHasWarning(trace, RecallWarningSupersededEvidenceObserved) {
		t.Fatalf("warnings = %+v, want superseded-evidence warning", trace.Warnings)
	}
	if !candidateIDSeen(trace.Candidates, "mem-owner-old") {
		t.Fatalf("candidates = %+v, want superseded evidence preserved", trace.Candidates)
	}
}

func TestRecallSpeakerRoutingDoesNotTreatSpeakerAsSubstring(t *testing.T) {
	candidate := ScoredRecallCandidate{Candidate: RecallCandidate{
		MemoryID:   "mem-ann",
		Content:    "Ann described the deployment plan.",
		AgentID:    "ann",
		Provenance: []EvidenceItem{{Kind: "speaker", Source: "ann", Score: 1, Note: "speaker=ann"}},
	}}

	if got := recallSpeakerAdjustment(candidate, "What is the planning status?"); got != 0 {
		t.Fatalf("recallSpeakerAdjustment() = %v, want no substring speaker bonus", got)
	}
}

func TestRecallSpeakerTargetMatchesFullIdentityPrefixOnly(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		speaker string
		want    bool
	}{
		{name: "exact", target: "juan perez", speaker: "juan perez", want: true},
		{name: "first name full identity", target: "juan", speaker: "juan perez", want: true},
		{name: "not substring", target: "ann", speaker: "annette", want: false},
		{name: "not suffix", target: "code", speaker: "claude-code", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := recallSpeakerTargetMatchesSpeaker(tt.target, tt.speaker); got != tt.want {
				t.Fatalf("recallSpeakerTargetMatchesSpeaker(%q, %q) = %v, want %v", tt.target, tt.speaker, got, tt.want)
			}
		})
	}
}

func TestRecallSpeakerRoutingMatchesMultiTokenSpeakerTarget(t *testing.T) {
	candidate := ScoredRecallCandidate{Candidate: RecallCandidate{
		MemoryID:   "mem-claude-code",
		Content:    "Claude Code summarized the migration risk.",
		AgentID:    "claude-code",
		Provenance: []EvidenceItem{{Kind: "speaker", Source: "claude-code", Score: 1, Note: "speaker=claude-code"}},
	}}

	if got := recallSpeakerAdjustment(candidate, "What did Claude Code say about migration risk?"); got != recallSpeakerMatchBonus {
		t.Fatalf("recallSpeakerAdjustment() = %v, want multi-token speaker bonus %v", got, recallSpeakerMatchBonus)
	}
}

func TestRecallSpeakerRoutingParsesSpeakerFieldWithoutTrailingMetadata(t *testing.T) {
	tests := []struct {
		name string
		note string
	}{
		{name: "leading speaker", note: "speaker=Mira source=turn-17"},
		{name: "speaker after provenance metadata", note: "source=turn-17 speaker=Mira"},
		{name: "parenthesized speaker field", note: "source=turn-17 (speaker=Mira)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := ScoredRecallCandidate{Candidate: RecallCandidate{
				MemoryID: "mem-mira-metadata",
				Content:  "Mira summarized the migration risk.",
				Provenance: []EvidenceItem{{
					Kind:   "speaker",
					Source: "turn-17",
					Score:  1,
					Note:   tt.note,
				}},
			}}

			if got := recallCandidateSpeaker(candidate.Candidate); got != "mira" {
				t.Fatalf("recallCandidateSpeaker() = %q, want speaker identity without trailing note metadata", got)
			}
			if got := recallSpeakerAdjustment(candidate, "What did Mira say about migration risk?"); got != recallSpeakerMatchBonus {
				t.Fatalf("recallSpeakerAdjustment() = %v, want speaker bonus %v", got, recallSpeakerMatchBonus)
			}
		})
	}
}

func TestRecallSpeakerRoutingPrefersExplicitSpeakerNoteOverOpaqueSource(t *testing.T) {
	candidate := ScoredRecallCandidate{Candidate: RecallCandidate{
		MemoryID: "mem-mira",
		Content:  "Mira summarized the migration risk.",
		Provenance: []EvidenceItem{{
			Kind:   "speaker",
			Source: "turn-17",
			Score:  1,
			Note:   "speaker=Mira",
		}},
	}}

	if got := recallCandidateSpeaker(candidate.Candidate); got != "mira" {
		t.Fatalf("recallCandidateSpeaker() = %q, want explicit speaker note mira", got)
	}
	if got := recallSpeakerAdjustment(candidate, "What did Mira say about migration risk?"); got != recallSpeakerMatchBonus {
		t.Fatalf("recallSpeakerAdjustment() = %v, want speaker bonus %v", got, recallSpeakerMatchBonus)
	}
}

func TestRecallSpeakerRoutingKeepsWhoSaidWhatInBranch(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-juan-theme",
			Content:    "Juan said he prefers dark theme for dense dashboards.",
			AgentID:    "juan",
			ScopeID:    "team",
			CreatedAt:  now.Add(-30 * time.Minute),
			Importance: 0.95,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 1.00}, {Kind: "speaker", Source: "juan", Score: 1.00, Note: "speaker=juan"}},
		},
		{
			MemoryID:   "mem-mira-theme",
			Content:    "Mira said Juan prefers light theme during demos.",
			AgentID:    "mira",
			ScopeID:    "team",
			CreatedAt:  now.Add(-2 * time.Hour),
			Importance: 0.70,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.92}, {Kind: "speaker", Source: "mira", Score: 1.00, Note: "speaker=mira"}},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "speaker-routing-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:       "speaker-routing-test-v1",
			Weights:       map[string]float64{"keyword": 0.75, "recency": 0.10, "importance": 0.10, "scope": 0.05},
			RRFK:          60,
			MMRLambda:     0.70,
			DiversityKeys: []string{"memory_id"},
			TokenBudget:   120,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "What did Mira say Juan preferred for demos?",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-mira-theme"}) {
		t.Fatalf("selected IDs = %v, want Mira speaker branch", selectedRecallIDs(trace))
	}
}

func candidateIDSeen(items []ScoredRecallCandidate, memoryID string) bool {
	return sliceutil.ContainsFunc(items, func(item ScoredRecallCandidate) bool {
		return item.Candidate.MemoryID == memoryID
	})
}
