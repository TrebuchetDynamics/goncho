package sourcefilter

import "testing"

func TestAllowsEmptyAndWildcardSources(t *testing.T) {
	for _, sources := range [][]string{nil, []string{}, []string{" * "}, []string{" "}} {
		if !Allows(sources, "memory", false) {
			t.Fatalf("Allows(%#v) should permit every source", sources)
		}
	}
}

func TestDecideExposesSourceFilterMatchReason(t *testing.T) {
	wildcard := Decide([]string{" * "}, "memory", false)
	if !wildcard.Allowed || !wildcard.Wildcard || wildcard.MatchedSource != "*" {
		t.Fatalf("wildcard decision = %+v, want explicit wildcard match", wildcard)
	}

	matched := Decide([]string{" Memory "}, " memory ", false)
	if !matched.Allowed || matched.Wildcard || matched.MatchedSource != "memory" {
		t.Fatalf("filtered decision = %+v, want explicit source match", matched)
	}

	emptyDenied := Decide([]string{"memory"}, " ", false)
	if emptyDenied.Allowed || emptyDenied.EmptySourceMatched {
		t.Fatalf("empty-source denied decision = %+v, want denied without legacy match", emptyDenied)
	}

	emptyAllowed := Decide([]string{"memory"}, " ", true)
	if !emptyAllowed.Allowed || !emptyAllowed.EmptySourceMatched || emptyAllowed.Wildcard {
		t.Fatalf("empty-source allowed decision = %+v, want explicit legacy empty-source match", emptyAllowed)
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

func TestAllowsKindOrOriginKeepsSourceKindDistinctFromAdapterSource(t *testing.T) {
	if !AllowsKindOrOrigin([]string{"turn"}, "turn", "discord", false) {
		t.Fatalf("AllowsKindOrOrigin should allow a storage/source-kind match")
	}
	if !AllowsKindOrOrigin([]string{"discord"}, "turn", "discord", false) {
		t.Fatalf("AllowsKindOrOrigin should allow an adapter-origin match")
	}
	if AllowsKindOrOrigin([]string{"conclusion"}, "turn", "discord", false) {
		t.Fatalf("AllowsKindOrOrigin should reject unrelated source kind and adapter origin")
	}
}
