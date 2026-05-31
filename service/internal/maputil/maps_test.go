package maputil

import "testing"

func TestClonePreservesNilAndCopiesGenericMaps(t *testing.T) {
	if got := Clone[string, int](nil); got != nil {
		t.Fatalf("nil input = %#v, want nil", got)
	}
	in := map[string]int{"a": 1}
	got := Clone(in)
	if got["a"] != 1 {
		t.Fatalf("cloned value = %d, want 1", got["a"])
	}
	got["a"] = 2
	if in["a"] != 1 {
		t.Fatalf("clone aliased input; input value = %d", in["a"])
	}
}

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
