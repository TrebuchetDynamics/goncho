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

func TestStringSetsNormalizeAndDropEmpty(t *testing.T) {
	got := LowerTrimmedSet([]string{" Alpha ", "alpha", "", " BETA"})
	want := map[string]struct{}{"alpha": {}, "beta": {}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LowerTrimmedSet() = %#v, want %#v", got, want)
	}
	if got := TrimmedSet([]string{" ", ""}); got != nil {
		t.Fatalf("TrimmedSet(empty) = %#v, want nil", got)
	}
}

func TestSortedSetValuesNormalizesAndSorts(t *testing.T) {
	got := SortedSetValues(map[string]struct{}{" beta ": {}, "alpha": {}, "": {}}, strings.TrimSpace)
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SortedSetValues() = %#v, want %#v", got, want)
	}
}
