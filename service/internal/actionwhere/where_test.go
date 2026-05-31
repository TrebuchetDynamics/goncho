package actionwhere

import "testing"

func TestScopedActionOmitsBlankActionID(t *testing.T) {
	where, args := ScopedAction("ws", "profile", "peer", "")
	if want := `workspace_id = ? AND profile_id = ? AND peer_id = ?`; where != want {
		t.Fatalf("where = %q, want %q", where, want)
	}
	if len(args) != 3 || args[0] != "ws" || args[1] != "profile" || args[2] != "peer" {
		t.Fatalf("args = %#v", args)
	}
}

func TestScopedActionIncludesActionID(t *testing.T) {
	where, args := ScopedAction("ws", "profile", "peer", "action")
	if want := `workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ?`; where != want {
		t.Fatalf("where = %q, want %q", where, want)
	}
	if len(args) != 4 || args[3] != "action" {
		t.Fatalf("args = %#v", args)
	}
}

func TestScopedActionSignalIncludesRequiredActionAndOptionalSignal(t *testing.T) {
	where, args := ScopedActionSignal("ws", "profile", "peer", "action", 12)
	if want := `workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ? AND signal_id = ?`; where != want {
		t.Fatalf("where = %q, want %q", where, want)
	}
	if len(args) != 5 || args[4] != int64(12) {
		t.Fatalf("args = %#v", args)
	}
}
