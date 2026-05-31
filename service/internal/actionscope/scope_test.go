package actionscope

import "testing"

func TestNormalizeUsesDefaultWorkspaceAndTrimsActionID(t *testing.T) {
	got := Normalize(" default-workspace ", "", " profile ", " peer ", " action-1 ")
	if got.WorkspaceID != "default-workspace" || got.ProfileID != "profile" || got.Peer != "peer" || got.ActionID != "action-1" {
		t.Fatalf("Normalize() = %+v, want trimmed default workspace/profile/peer/action", got)
	}
	if !got.Complete() {
		t.Fatalf("Normalize().Complete() = false, want true")
	}
}

func TestScopeCompleteRequiresWorkspaceAndPeer(t *testing.T) {
	if (Scope{WorkspaceID: "w"}).Complete() {
		t.Fatalf("Complete() with missing peer = true, want false")
	}
	if (Scope{Peer: "p"}).Complete() {
		t.Fatalf("Complete() with missing workspace = true, want false")
	}
}
