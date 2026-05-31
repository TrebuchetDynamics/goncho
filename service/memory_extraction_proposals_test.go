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

func TestExtractMemoryProposalsDoesNotDuplicateOpenReviewItems(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if _, err := svc.CreateMessages(ctx, CreateMessagesParams{SessionKey: "sess-review-repeat", Messages: []CreateMessage{
		{Peer: "peer-review-repeat", Role: "user", Content: "Remember: API token is sk-live-repeat-secret."},
	}}); err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}

	first, err := svc.ExtractMemoryProposals(ctx, ExtractMemoryProposalsParams{Peer: "peer-review-repeat", SessionKey: "sess-review-repeat", Window: 10})
	if err != nil {
		t.Fatalf("ExtractMemoryProposals first: %v", err)
	}
	second, err := svc.ExtractMemoryProposals(ctx, ExtractMemoryProposalsParams{Peer: "peer-review-repeat", SessionKey: "sess-review-repeat", Window: 10})
	if err != nil {
		t.Fatalf("ExtractMemoryProposals second: %v", err)
	}
	if len(first.Proposals) != 1 || len(second.Proposals) != 1 {
		t.Fatalf("proposal counts first=%+v second=%+v, want one sensitive proposal each", first.Proposals, second.Proposals)
	}
	if first.Proposals[0].ReviewItemID == "" || second.Proposals[0].ReviewItemID == "" {
		t.Fatalf("review item ids first=%+v second=%+v, want both linked to review", first.Proposals, second.Proposals)
	}
	if first.Proposals[0].ReviewItemID != second.Proposals[0].ReviewItemID {
		t.Fatalf("review item ids = %q then %q, want repeated extraction to reuse existing open review", first.Proposals[0].ReviewItemID, second.Proposals[0].ReviewItemID)
	}
	open, err := svc.ListReviewItems(ctx, ReviewQuery{PeerID: "peer-review-repeat", SessionKey: "sess-review-repeat", SubjectID: first.Proposals[0].ID, Status: ReviewStatusOpen})
	if err != nil {
		t.Fatalf("ListReviewItems: %v", err)
	}
	if len(open.Items) != 1 {
		t.Fatalf("open review items = %+v, want one idempotent review item for repeated extraction", open.Items)
	}
}

func TestSelectProposalWindowKeepsNewestMessagesInTimelineOrder(t *testing.T) {
	messages := []MessageRecord{
		{ID: 1, Content: "oldest"},
		{ID: 2, Content: "middle"},
		{ID: 3, Content: "newest"},
	}

	selection := selectProposalWindow(messages, 2)

	if selection.Window.Requested != 2 || selection.Window.MessageCount != 2 || selection.Window.Total != 3 || !selection.Window.Truncated {
		t.Fatalf("window = %+v, want requested=2 message_count=2 total=3 truncated=true", selection.Window)
	}
	if got := []int64{selection.Messages[0].ID, selection.Messages[1].ID}; !slices.Equal(got, []int64{2, 3}) {
		t.Fatalf("selected message ids = %v, want newest messages in original timeline order", got)
	}
	selection.Messages[0].Content = "mutated selection"
	if messages[1].Content != "middle" {
		t.Fatalf("source messages were mutated through selection alias: %+v", messages)
	}
}

func TestSelectProposalWindowDefaultsNonPositiveRequests(t *testing.T) {
	selection := selectProposalWindow([]MessageRecord{{ID: 1}}, 0)

	if selection.Window.Requested != 20 || selection.Window.MessageCount != 1 || selection.Window.Total != 1 || selection.Window.Truncated {
		t.Fatalf("window = %+v, want default requested=20 without truncation", selection.Window)
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
		{Peer: "peer-pref", ProfileID: "profile-b", Role: "user", Content: "Preference: prefers detailed release notes."},
	}}); err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}

	got, err := svc.ExtractMemoryProposals(ctx, ExtractMemoryProposalsParams{Peer: "peer-pref", ProfileID: "profile-a", SessionKey: "sess-pref", Window: 5})
	if err != nil {
		t.Fatalf("ExtractMemoryProposals: %v", err)
	}
	if len(got.Proposals) != 1 {
		t.Fatalf("proposals = %+v, want only profile-a preference proposal", got.Proposals)
	}
	proposal := got.Proposals[0]
	if proposal.Kind != MemoryProposalPreference || proposal.Scope != MemoryScopeProfile || proposal.ProfileID != "profile-a" || !strings.Contains(proposal.ExpiryHint, "stable preference") {
		t.Fatalf("preference proposal = %+v, want profile-scoped stable preference", proposal)
	}
	if strings.Contains(proposal.Content, "detailed") {
		t.Fatalf("preference proposal = %+v, want no profile-b content attributed to profile-a", proposal)
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
