package stringlist

import "testing"

func TestCloneReturnsIndependentStringSlice(t *testing.T) {
	in := []string{"a", "b"}
	out := Clone(in)

	if len(out) != len(in) || out[0] != "a" || out[1] != "b" {
		t.Fatalf("Clone() = %#v, want copy of %#v", out, in)
	}
	out[0] = "changed"
	if in[0] != "a" {
		t.Fatalf("Clone() returned alias; source mutated to %#v", in)
	}
}

func TestClonePreservesNil(t *testing.T) {
	if out := Clone(nil); out != nil {
		t.Fatalf("Clone(nil) = %#v, want nil", out)
	}
}

func TestAnyUsesPredicateContract(t *testing.T) {
	if !Any([]string{"alpha", "beta"}, func(value string) bool { return value == "beta" }) {
		t.Fatalf("Any should report the first predicate match")
	}
	if Any([]string{"alpha"}, nil) {
		t.Fatalf("Any should reject nil predicate")
	}
	if Any(nil, func(value string) bool { return true }) {
		t.Fatalf("Any should reject empty input")
	}
}
