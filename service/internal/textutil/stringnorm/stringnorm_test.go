package stringnorm

import (
	"reflect"
	"strings"
	"testing"
)

func TestApplyPreservesNilNormalizerContract(t *testing.T) {
	if got := Apply(nil, " Value "); got != " Value " {
		t.Fatalf("Apply(nil) = %q, want unchanged", got)
	}
	if got := Apply(strings.TrimSpace, " Value "); got != "Value" {
		t.Fatalf("Apply(trim) = %q, want Value", got)
	}
}

func TestUniquePreservesOrderOrSortsNormalizedValues(t *testing.T) {
	got := Unique([]string{"./src", "src", " ./pkg "}, func(value string) string {
		value = strings.TrimSpace(value)
		return strings.TrimPrefix(value, "./")
	}, false)
	want := []string{"src", "pkg"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unique() = %#v, want %#v", got, want)
	}

	got = Unique([]string{" Beta ", "alpha", "BETA"}, func(value string) string {
		return strings.ToLower(strings.TrimSpace(value))
	}, true)
	want = []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unique(sorted) = %#v, want %#v", got, want)
	}
}

func TestSetAndSortedSetValuesDropEmptyNormalizedValues(t *testing.T) {
	got := Set([]string{" Alpha ", "alpha", "", " BETA"}, func(value string) string {
		return strings.ToLower(strings.TrimSpace(value))
	})
	want := map[string]struct{}{"alpha": {}, "beta": {}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Set() = %#v, want %#v", got, want)
	}
	if got := Set([]string{" ", ""}, strings.TrimSpace); got != nil {
		t.Fatalf("Set(empty) = %#v, want nil", got)
	}

	values := map[string]struct{}{" beta ": {}, "alpha": {}, "": {}}
	if got := SortedSetValues(values, strings.TrimSpace); !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Fatalf("SortedSetValues() = %#v", got)
	}
}
