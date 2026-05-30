package scopeauth

import "testing"

func TestNormalizeActorScopeDefaultsWorkspaceAndTrimsProfile(t *testing.T) {
	got := NormalizeActorScope("", " profile-a ", "workspace-a")
	if got.WorkspaceID != "workspace-a" || got.ProfileID != "profile-a" {
		t.Fatalf("NormalizeActorScope() = %+v", got)
	}
}

func TestSameScope(t *testing.T) {
	actor := ActorScope{WorkspaceID: "workspace-a", ProfileID: "profile-a"}
	if !SameScope(actor, "workspace-a", "profile-a") {
		t.Fatal("SameScope returned false for identical scope")
	}
	if SameScope(actor, "workspace-a", "profile-b") {
		t.Fatal("SameScope returned true for different profile")
	}
}

func TestDeniedReadReason(t *testing.T) {
	actor := ActorScope{WorkspaceID: "workspace-b", ProfileID: "profile-b"}
	want := `actor scope workspace="workspace-b" profile="profile-b" cannot read signal scope workspace="workspace-a" profile="profile-a"`
	if got := DeniedReadReason(actor, "signal", "workspace-a", "profile-a"); got != want {
		t.Fatalf("DeniedReadReason() = %q, want %q", got, want)
	}
}
