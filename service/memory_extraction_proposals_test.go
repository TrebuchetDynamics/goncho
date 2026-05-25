package goncho

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestExtractMemoryProposalsClassifiesAddUpdateDeleteNoopWithEvidence(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-extract", SessionKey: "prior", Conclusion: "Payment API owner is Mira."}); err != nil {
		t.Fatalf("seed conclusion: %v", err)
	}
	created, err := svc.CreateMessages(ctx, CreateMessagesParams{SessionKey: "sess-extract", Messages: []CreateMessage{
		{Peer: "peer-extract", Role: "user", Content: "Remember: Release checklist lives in docs/release.md."},
		{Peer: "peer-extract", Role: "user", Content: "Update: Payment API owner is Nia."},
		{Peer: "peer-extract", Role: "user", Content: "Forget: The staging deploy freezes after 4pm."},
		{Peer: "peer-extract", Role: "assistant", Content: "Sounds good, I will keep that in mind."},
	}})
	if err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}

	got, err := svc.ExtractMemoryProposals(ctx, ExtractMemoryProposalsParams{Peer: "peer-extract", SessionKey: "sess-extract", Window: 10})
	if err != nil {
		t.Fatalf("ExtractMemoryProposals: %v", err)
	}
	if got.WorkspaceID != svc.workspaceID || got.Peer != "peer-extract" || got.SessionKey != "sess-extract" {
		t.Fatalf("scope = %+v, want service workspace/peer/session", got)
	}
	if got.Window.MessageCount != len(created.Messages) || got.Window.Truncated {
		t.Fatalf("window = %+v, want all messages untruncated", got.Window)
	}
	if got.ActiveMemoryWrites != 0 {
		t.Fatalf("active writes = %d, want proposals only", got.ActiveMemoryWrites)
	}
	ops := proposalOps(got.Proposals)
	for _, want := range []MemoryProposalOperation{MemoryProposalAdd, MemoryProposalUpdate, MemoryProposalDelete, MemoryProposalNoop} {
		if !slices.Contains(ops, want) {
			t.Fatalf("ops = %v, missing %s in proposals %+v", ops, want, got.Proposals)
		}
	}
	for _, proposal := range got.Proposals {
		if len(proposal.EvidenceIDs) == 0 || !strings.HasPrefix(proposal.EvidenceIDs[0], "msg:") {
			t.Fatalf("proposal %+v missing message evidence id", proposal)
		}
		if proposal.Status != MemoryProposalReady && proposal.Operation != MemoryProposalNoop {
			t.Fatalf("proposal %+v should be ready unless noop", proposal)
		}
	}
}

func TestExtractMemoryProposalsRoutesConflictAndSensitiveClaimsToReview(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-conflict", SessionKey: "prior", Conclusion: "Deployment owner is Mira."}); err != nil {
		t.Fatalf("seed conclusion: %v", err)
	}
	if _, err := svc.CreateMessages(ctx, CreateMessagesParams{SessionKey: "sess-conflict", Messages: []CreateMessage{
		{Peer: "peer-conflict", Role: "user", Content: "Remember: Deployment owner is Nia."},
		{Peer: "peer-conflict", Role: "user", Content: "Remember: API token is sk-live-secret."},
	}}); err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}

	got, err := svc.ExtractMemoryProposals(ctx, ExtractMemoryProposalsParams{Peer: "peer-conflict", SessionKey: "sess-conflict", Window: 10})
	if err != nil {
		t.Fatalf("ExtractMemoryProposals: %v", err)
	}
	var reviewRequired int
	for _, proposal := range got.Proposals {
		if proposal.Status == MemoryProposalReviewRequired {
			reviewRequired++
			if proposal.ReviewItemID == "" {
				t.Fatalf("review proposal %+v missing review item id", proposal)
			}
		}
	}
	if reviewRequired != 2 {
		t.Fatalf("review-required proposal count = %d, proposals %+v", reviewRequired, got.Proposals)
	}
	open, err := svc.ListReviewItems(ctx, ReviewQuery{PeerID: "peer-conflict", SessionKey: "sess-conflict", Status: ReviewStatusOpen})
	if err != nil {
		t.Fatalf("ListReviewItems: %v", err)
	}
	if len(open.Items) != 2 {
		t.Fatalf("open review items = %+v, want conflict and privacy-sensitive items", open.Items)
	}
	for _, item := range open.Items {
		if len(item.EvidenceIDs) == 0 || !strings.HasPrefix(item.EvidenceIDs[0], "msg:") {
			t.Fatalf("review item %+v missing message evidence", item)
		}
	}
	latest, err := svc.Search(ctx, SearchParams{Peer: "peer-conflict", Query: "Deployment owner", Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, hit := range latest.Results {
		if strings.Contains(hit.Content, "sk-live-secret") || strings.Contains(hit.Content, "Deployment owner is Nia") {
			t.Fatalf("review-required proposal leaked into active memory hit %+v", hit)
		}
	}
}

func TestExtractMemoryProposalsPreferenceScopeDoesNotLeakAcrossProfiles(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if _, err := svc.CreateMessages(ctx, CreateMessagesParams{SessionKey: "sess-pref", Messages: []CreateMessage{
		{Peer: "peer-pref", ProfileID: "profile-a", Role: "user", Content: "Preference: prefers terse release notes."},
	}}); err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}

	got, err := svc.ExtractMemoryProposals(ctx, ExtractMemoryProposalsParams{Peer: "peer-pref", ProfileID: "profile-a", SessionKey: "sess-pref", Window: 5})
	if err != nil {
		t.Fatalf("ExtractMemoryProposals: %v", err)
	}
	if len(got.Proposals) != 1 {
		t.Fatalf("proposals = %+v, want one preference proposal", got.Proposals)
	}
	proposal := got.Proposals[0]
	if proposal.Kind != MemoryProposalPreference || proposal.Scope != MemoryScopeProfile || proposal.ProfileID != "profile-a" || !strings.Contains(proposal.ExpiryHint, "stable preference") {
		t.Fatalf("preference proposal = %+v, want profile-scoped stable preference", proposal)
	}
	other, err := svc.ProfileInNamespace(ctx, MemoryNamespace{ProfileID: "profile-b", PeerID: "peer-pref"})
	if err != nil {
		t.Fatalf("ProfileInNamespace other: %v", err)
	}
	if len(other.Card) != 0 {
		t.Fatalf("profile-b card = %+v, want no leak from profile-a proposal", other.Card)
	}
}

func proposalOps(proposals []MemoryProposal) []MemoryProposalOperation {
	out := make([]MemoryProposalOperation, 0, len(proposals))
	for _, proposal := range proposals {
		out = append(out, proposal.Operation)
	}
	return out
}
