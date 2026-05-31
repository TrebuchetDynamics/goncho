package scopekey

import "testing"

func TestWorkspaceTrimsDefaultsAndHandlesQueryWildcard(t *testing.T) {
	if got := Workspace(" default ", " ", false); got != "default" {
		t.Fatalf("Workspace blank = %q, want default", got)
	}
	if got := Workspace("default", " requested ", false); got != "requested" {
		t.Fatalf("Workspace requested = %q, want requested", got)
	}
	if got := Workspace("default", " * ", true); got != "" {
		t.Fatalf("Workspace wildcard = %q, want empty all-workspaces marker", got)
	}
	if got := Workspace("default", " * ", false); got != "*" {
		t.Fatalf("Workspace non-query wildcard = %q, want literal wildcard", got)
	}
}

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
