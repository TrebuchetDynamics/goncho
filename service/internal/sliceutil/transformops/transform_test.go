package transformops

import (
	"reflect"
	"testing"
)

func TestMapPreservesNilInput(t *testing.T) {
	if got := Map[string, int](nil, func(value string) int { return len(value) }); got != nil {
		t.Fatalf("Map(nil) = %#v, want nil", got)
	}
}

func TestMapAppliesFunctionInOrder(t *testing.T) {
	got := Map([]string{"alpha", "go"}, func(value string) int { return len(value) })
	want := []int{5, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Map = %v, want %v", got, want)
	}
}

func TestMapNilFunctionReturnsZeroValues(t *testing.T) {
	got := Map[string, int]([]string{"alpha", "go"}, nil)
	want := []int{0, 0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Map nil fn = %v, want %v", got, want)
	}
}

func TestFilterMapKeepsMappedAcceptedValues(t *testing.T) {
	got := FilterMap([]int{1, 2, 3, 4}, func(value int) (string, bool) {
		if value%2 != 0 {
			return "", false
		}
		return string(rune('a' + value)), true
	})
	want := []string{"c", "e"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FilterMap = %v, want %v", got, want)
	}
}

func TestFilterMapPreservesNilInput(t *testing.T) {
	if got := FilterMap[int, string](nil, func(value int) (string, bool) { return "", true }); got != nil {
		t.Fatalf("FilterMap nil = %#v, want nil", got)
	}
}

func TestFilterMapNilFunctionRejectsAllValues(t *testing.T) {
	if got := FilterMap[int, string]([]int{1, 2}, nil); len(got) != 0 {
		t.Fatalf("FilterMap nil fn = %#v, want empty non-nil slice", got)
	}
}

func TestIndexByKeepsFirstAcceptedIndex(t *testing.T) {
	values := []string{"alpha", "", "beta", "alpha"}
	got := IndexBy(values, func(value string) (string, bool) {
		return value, value != ""
	})

	want := map[string]int{"alpha": 0, "beta": 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("IndexBy() = %#v, want %#v", got, want)
	}
}
