package textutil

import (
	"reflect"
	"strings"
	"testing"
)

func TestUniqueTrimmedPreservesFirstSeenOrder(t *testing.T) {
	got := UniqueTrimmed([]string{" alpha ", "", "beta", "alpha", " beta "}, false)
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UniqueTrimmed() = %#v, want %#v", got, want)
	}
}

func TestUniqueLowerTrimmedCanSort(t *testing.T) {
	got := UniqueLowerTrimmed([]string{" Beta ", "alpha", "BETA"}, true)
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UniqueLowerTrimmed() = %#v, want %#v", got, want)
	}
}

func TestNormalizeUniqueUsesCustomNormalizer(t *testing.T) {
	got := NormalizeUnique([]string{"./src", "src", " ./pkg "}, func(value string) string {
		value = strings.TrimSpace(value)
		return strings.TrimPrefix(value, "./")
	}, false)
	want := []string{"src", "pkg"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeUnique() = %#v, want %#v", got, want)
	}
}
