package sliceutil

import (
	"reflect"
	"testing"
)

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
