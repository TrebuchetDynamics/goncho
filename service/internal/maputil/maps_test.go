package maputil

import "testing"

func TestCloneStringStringNilIfEmpty(t *testing.T) {
	if got := CloneStringStringNilIfEmpty(nil); got != nil {
		t.Fatalf("nil input = %#v, want nil", got)
	}
	if got := CloneStringStringNilIfEmpty(map[string]string{}); got != nil {
		t.Fatalf("empty input = %#v, want nil", got)
	}
	in := map[string]string{"k": "v"}
	got := CloneStringStringNilIfEmpty(in)
	if got["k"] != "v" {
		t.Fatalf("cloned value = %q, want v", got["k"])
	}
	got["k"] = "changed"
	if in["k"] != "v" {
		t.Fatalf("clone aliased input; input value = %q", in["k"])
	}
}
