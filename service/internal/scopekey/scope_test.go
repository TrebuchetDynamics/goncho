package scopekey

import "testing"

func TestNormalizeTrimsScopeAndFallsBackToDefaultWorkspace(t *testing.T) {
	got := Normalize(" default ", " ", " profile ", " peer ")
	if got.WorkspaceID != "default" || got.ProfileID != "profile" || got.Peer != "peer" {
		t.Fatalf("Normalize() = %#v", got)
	}
	if !got.Complete() {
		t.Fatalf("Complete() = false, want true")
	}
}

func TestNormalizeIncompleteWhenWorkspaceOrPeerBlank(t *testing.T) {
	if Normalize("", "", "profile", "peer").Complete() {
		t.Fatalf("Complete() with blank workspace = true, want false")
	}
	if Normalize("workspace", "", "profile", " ").Complete() {
		t.Fatalf("Complete() with blank peer = true, want false")
	}
}
