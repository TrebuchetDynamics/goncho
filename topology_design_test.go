package goncho

import (
	"context"
	"errors"
	"slices"
	"testing"

	memory "github.com/TrebuchetDynamics/goncho/memory"
)

func TestTopologyDefaultWorkspaceIsGormes(t *testing.T) {
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	svc := NewService(store.DB(), Config{}, nil)
	got, err := svc.Profile(context.Background(), "user-juan")
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkspaceID != DefaultWorkspaceID {
		t.Fatalf("workspace = %q, want %q", got.WorkspaceID, DefaultWorkspaceID)
	}
}

func TestTopologyRejectsWorkspacePerUser(t *testing.T) {
	_, err := ResolveWorkspace(WorkspaceRequest{
		Strategy: WorkspaceStrategyPerUser,
		UserID:   "user-juan",
	})
	if !errors.Is(err, ErrWorkspacePerUser) {
		t.Fatalf("ResolveWorkspace err = %v, want ErrWorkspacePerUser", err)
	}
}

func TestTopologyAllowsExplicitHardIsolationWorkspace(t *testing.T) {
	got, err := ResolveWorkspace(WorkspaceRequest{
		Strategy:    WorkspaceStrategyHardIsolation,
		WorkspaceID: "gormes-test-fixture",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkspaceID != "gormes-test-fixture" {
		t.Fatalf("workspace = %q, want gormes-test-fixture", got.WorkspaceID)
	}
	if !got.HardIsolation {
		t.Fatal("HardIsolation = false, want true")
	}
}

func TestTopologyPeerIDPrefersCanonicalSessionUserID(t *testing.T) {
	got, err := ResolvePeerID(SessionMetadata{
		SessionID: "sess-telegram",
		Source:    "telegram",
		ChatID:    "42",
		UserID:    "user-juan",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.PeerID != "user-juan" {
		t.Fatalf("PeerID = %q, want user-juan", got.PeerID)
	}
	if got.Degraded {
		t.Fatalf("Degraded = true, evidence = %+v", got.Evidence)
	}
}

func TestTopologyUnknownExternalParticipantFallsBackWithEvidence(t *testing.T) {
	first, err := ResolvePeerID(SessionMetadata{
		SessionID: "sess-telegram",
		Source:    "telegram",
		ChatID:    "42",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ResolvePeerID(SessionMetadata{
		SessionID: "sess-later",
		Source:    " telegram ",
		ChatID:    " 42 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.PeerID != "telegram:42" || second.PeerID != first.PeerID {
		t.Fatalf("fallback peer IDs = %q and %q, want deterministic telegram:42", first.PeerID, second.PeerID)
	}
	if !first.Degraded {
		t.Fatal("Degraded = false, want true for source-prefixed fallback")
	}
	if !slices.Contains(first.Evidence, EvidenceExternalPeerFallback) {
		t.Fatalf("Evidence = %#v, want %q", first.Evidence, EvidenceExternalPeerFallback)
	}
}

func TestTopologyObservationDefaults(t *testing.T) {
	for _, role := range []PeerRole{
		PeerRoleGormesAssistant,
		PeerRoleDeterministicAssistant,
		PeerRoleTransportBot,
		PeerRoleImportHelper,
	} {
		got := DefaultObservation(ObservationRequest{Role: role})
		if got.ObserveMe {
			t.Fatalf("%s ObserveMe = true, want false", role)
		}
		if got.ObserveOthers {
			t.Fatalf("%s ObserveOthers = true, want false without opt-in", role)
		}
	}

	user := DefaultObservation(ObservationRequest{Role: PeerRoleHuman})
	if !user.ObserveMe {
		t.Fatal("human ObserveMe = false, want true")
	}
	if user.ObserveOthers {
		t.Fatal("human ObserveOthers = true, want false by default")
	}
}

func TestTopologySessionBoundaryChoices(t *testing.T) {
	for _, boundary := range []SessionBoundaryKind{
		SessionBoundaryThread,
		SessionBoundaryChannel,
		SessionBoundaryRepository,
		SessionBoundaryImportBatch,
		SessionBoundaryDelegatedChildRun,
	} {
		got, err := ResolveSessionBoundary(SessionBoundaryRequest{Kind: boundary, Key: "alpha"})
		if err != nil {
			t.Fatalf("ResolveSessionBoundary(%s): %v", boundary, err)
		}
		if got.Kind != boundary {
			t.Fatalf("Kind = %q, want %q", got.Kind, boundary)
		}
		if got.SessionKey == "" {
			t.Fatalf("SessionKey empty for %s", boundary)
		}
	}
}

func TestTopologyCrossPeerObservationIsOptIn(t *testing.T) {
	got := DefaultObservation(ObservationRequest{Role: PeerRoleParentAgent})
	if got.ObserveOthers {
		t.Fatal("ObserveOthers = true without opt-in")
	}
	if got.ObserveMe {
		t.Fatal("parent agent observer ObserveMe = true, want false")
	}

	optedIn := DefaultObservation(ObservationRequest{
		Role:                 PeerRoleParentAgent,
		CrossPeerObservation: true,
	})
	if !optedIn.ObserveOthers {
		t.Fatal("ObserveOthers = false, want true when explicitly opted in")
	}
	if optedIn.ObserveMe {
		t.Fatal("parent agent observer ObserveMe = true after opt-in, want false")
	}
}
