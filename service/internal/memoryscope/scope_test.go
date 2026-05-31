package memoryscope

import "testing"

func TestNormalizeKeepsKnownScopes(t *testing.T) {
	for _, scope := range []string{Profile, Workspace, Shared, Session, Global} {
		if got := Normalize("  "+scope+"  ", "profile-a"); got != scope {
			t.Fatalf("Normalize(%q) = %q, want %q", scope, got, scope)
		}
	}
}

func TestNormalizeDefaultsByProfilePresence(t *testing.T) {
	if got := Normalize("", "profile-a"); got != Profile {
		t.Fatalf("Normalize empty with profile = %q, want %q", got, Profile)
	}
	if got := Normalize("", " "); got != Workspace {
		t.Fatalf("Normalize empty without profile = %q, want %q", got, Workspace)
	}
}

func TestNormalizeLowercasesKnownScope(t *testing.T) {
	if got := Normalize(" SHARED ", ""); got != Shared {
		t.Fatalf("Normalize uppercase shared = %q, want %q", got, Shared)
	}
}
