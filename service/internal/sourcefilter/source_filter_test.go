package sourcefilter

import "testing"

func TestAllowsEmptyAndWildcardSources(t *testing.T) {
	for _, sources := range [][]string{nil, []string{}, []string{" * "}, []string{" "}} {
		if !Allows(sources, "memory", false) {
			t.Fatalf("Allows(%#v) should permit every source", sources)
		}
	}
}

func TestAllowsEmptySourceOnlyWhenLegacyAllowed(t *testing.T) {
	sources := []string{"memory"}
	if !Allows(sources, " ", true) {
		t.Fatalf("Allows should permit blank source when legacy empty-source match is enabled")
	}
	if Allows(sources, " ", false) {
		t.Fatalf("Allows should reject blank source when legacy empty-source match is disabled")
	}
}

func TestAllowsMatchesSourcesCaseInsensitively(t *testing.T) {
	if !Allows([]string{" Memory "}, "memory", false) {
		t.Fatalf("Allows should compare source names case-insensitively after trimming")
	}
	if Allows([]string{"memory"}, "search", false) {
		t.Fatalf("Allows should reject a source not in the allow-list")
	}
}
