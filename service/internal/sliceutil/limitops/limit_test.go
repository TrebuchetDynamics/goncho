package limitops

import "testing"

func TestLimit(t *testing.T) {
	values := []int{1, 2, 3}
	if got := Limit(values, 2); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("Limit = %#v, want first two values", got)
	}
	if got := Limit(values, 0); len(got) != 3 {
		t.Fatalf("Limit with zero = %#v, want unchanged", got)
	}
	if got := Limit(values, -1); len(got) != 3 {
		t.Fatalf("Limit with negative = %#v, want unchanged", got)
	}
}

func TestLimitClone(t *testing.T) {
	values := []int{1, 2, 3}
	got := LimitClone(values, 2)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("LimitClone = %#v, want first two values", got)
	}
	got[0] = 99
	if values[0] != 1 {
		t.Fatalf("LimitClone aliased source slice; source = %#v", values)
	}
	if got := LimitClone([]int(nil), 2); got != nil {
		t.Fatalf("LimitClone nil = %#v, want nil", got)
	}
}
