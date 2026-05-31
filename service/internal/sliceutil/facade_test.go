package sliceutil

import (
	"reflect"
	"testing"
)

func TestFacadeDelegatesCopyOps(t *testing.T) {
	input := []string{"first", "second", "third"}
	cloned := Clone(input)
	cloned[0] = "changed"
	if input[0] != "first" {
		t.Fatalf("Clone aliased input: input = %#v", input)
	}

	got := ReverseClone(input)
	want := []string{"third", "second", "first"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReverseClone = %v, want %v", got, want)
	}
}

func TestFacadeDelegatesSearchOps(t *testing.T) {
	if !Contains([]string{"operator", "system"}, "system") {
		t.Fatal("Contains did not find existing string")
	}
	if Contains([]string{"operator", "system"}, "developer") {
		t.Fatal("Contains found absent string")
	}

	type warning struct{ code string }
	warnings := []warning{{code: "token_budget"}, {code: "semantic_unavailable"}}
	if !ContainsFunc(warnings, func(w warning) bool { return w.code == "semantic_unavailable" }) {
		t.Fatal("ContainsFunc did not find matching struct")
	}

	if got := First([]string{"a", "b"}); got != "a" {
		t.Fatalf("First(strings) = %q, want a", got)
	}
}

func TestFacadeDelegatesTransformOps(t *testing.T) {
	mapped := Map([]string{"alpha", "go"}, func(value string) int { return len(value) })
	if want := []int{5, 2}; !reflect.DeepEqual(mapped, want) {
		t.Fatalf("Map = %v, want %v", mapped, want)
	}

	filtered := FilterMap([]int{1, 2, 3, 4}, func(value int) (string, bool) {
		if value%2 != 0 {
			return "", false
		}
		return string(rune('a' + value)), true
	})
	if want := []string{"c", "e"}; !reflect.DeepEqual(filtered, want) {
		t.Fatalf("FilterMap = %v, want %v", filtered, want)
	}

	indexed := IndexBy([]string{"alpha", "", "beta", "alpha"}, func(value string) (string, bool) {
		return value, value != ""
	})
	if want := map[string]int{"alpha": 0, "beta": 2}; !reflect.DeepEqual(indexed, want) {
		t.Fatalf("IndexBy = %#v, want %#v", indexed, want)
	}
}

func TestFacadeDelegatesLimitOps(t *testing.T) {
	values := []int{1, 2, 3}
	if got := Limit(values, 2); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("Limit = %#v, want first two values", got)
	}

	limited := LimitClone(values, 2)
	if !reflect.DeepEqual(limited, []int{1, 2}) {
		t.Fatalf("LimitClone = %#v, want first two values", limited)
	}
	limited[0] = 99
	if values[0] != 1 {
		t.Fatalf("LimitClone aliased source slice; source = %#v", values)
	}
}
