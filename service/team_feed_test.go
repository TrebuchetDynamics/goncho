package goncho

import (
	"context"
	"testing"
)

func TestTeamFeedListsActionSignalsWithPaginationForAuthorizedActor(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.UpsertAction(ctx, ActionParams{ProfileID: "profile-team", Peer: "peer-team", ActionID: "alpha", Title: "Alpha"}); err != nil {
		t.Fatalf("UpsertAction alpha: %v", err)
	}
	if _, err := svc.UpsertAction(ctx, ActionParams{ProfileID: "profile-team", Peer: "peer-team", ActionID: "beta", Title: "Beta"}); err != nil {
		t.Fatalf("UpsertAction beta: %v", err)
	}
	if _, err := svc.SignalAction(ctx, ActionSignalParams{ProfileID: "profile-team", Peer: "peer-team", ActionID: "alpha", Signal: "ready", Message: "alpha ready", Actor: "agent:a"}); err != nil {
		t.Fatalf("SignalAction alpha: %v", err)
	}
	if _, err := svc.SignalAction(ctx, ActionSignalParams{ProfileID: "profile-team", Peer: "peer-team", ActionID: "beta", Signal: "blocked", Message: "beta blocked", Actor: "agent:b"}); err != nil {
		t.Fatalf("SignalAction beta: %v", err)
	}

	first, err := svc.TeamFeed(ctx, TeamFeedQuery{ProfileID: "profile-team", Peer: "peer-team", Actor: "agent:reader", ActorProfileID: "profile-team", Limit: 1})
	if err != nil {
		t.Fatalf("TeamFeed first: %v", err)
	}
	if !first.Authorized || first.Decision != TeamFeedDecisionAllowed || len(first.Entries) != 1 || first.NextCursor == "" {
		t.Fatalf("first feed = %+v, want one entry with cursor", first)
	}
	second, err := svc.TeamFeed(ctx, TeamFeedQuery{ProfileID: "profile-team", Peer: "peer-team", Actor: "agent:reader", ActorProfileID: "profile-team", Limit: 1, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("TeamFeed second: %v", err)
	}
	if !second.Authorized || len(second.Entries) != 1 || second.NextCursor != "" || second.Entries[0].ID == first.Entries[0].ID {
		t.Fatalf("second feed = %+v first=%+v, want next distinct final page", second, first)
	}
}

func TestTeamFeedDeniesCrossProfileAndAuditsDecision(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.UpsertAction(ctx, ActionParams{ProfileID: "profile-a", Peer: "peer-team", ActionID: "private", Title: "Private"}); err != nil {
		t.Fatalf("UpsertAction: %v", err)
	}
	if _, err := svc.SignalAction(ctx, ActionSignalParams{ProfileID: "profile-a", Peer: "peer-team", ActionID: "private", Signal: "ready", Actor: "agent:a"}); err != nil {
		t.Fatalf("SignalAction: %v", err)
	}

	denied, err := svc.TeamFeed(ctx, TeamFeedQuery{ProfileID: "profile-a", Peer: "peer-team", Actor: "agent:b", ActorProfileID: "profile-b", Limit: 10})
	if err != nil {
		t.Fatalf("TeamFeed denied: %v", err)
	}
	if denied.Authorized || denied.Decision != TeamFeedDecisionDenied || len(denied.Entries) != 0 || denied.Reason == "" {
		t.Fatalf("denied feed = %+v, want observable denial and no entries", denied)
	}
	audit, err := svc.ListTeamFeedAudit(ctx, TeamFeedAuditQuery{ProfileID: "profile-a", Peer: "peer-team"})
	if err != nil {
		t.Fatalf("ListTeamFeedAudit: %v", err)
	}
	if audit.Count != 1 || audit.Events[0].Decision != TeamFeedDecisionDenied || audit.Events[0].Actor != "agent:b" {
		t.Fatalf("audit = %+v, want denied actor evidence", audit)
	}
}
