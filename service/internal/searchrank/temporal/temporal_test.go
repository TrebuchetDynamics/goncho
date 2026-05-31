package temporal

import "testing"

func TestTemporalQueryAndMarkersNormalizeCase(t *testing.T) {
	query := "What happened LAST WEEK in JANUARY?"
	if !Query(query) {
		t.Fatalf("Query(%q) = false, want true for uppercase temporal marker", query)
	}
	markers := Markers(query)
	if !stringSliceContains(markers, "last week") || !stringSliceContains(markers, "january") {
		t.Fatalf("TemporalMarkers(%q) = %#v, want last week and january", query, markers)
	}
}

func TestTemporalQueryDoesNotTreatLastNameAsTemporal(t *testing.T) {
	query := "What is Maya's last name?"
	if Query(query) {
		t.Fatalf("Query(%q) = true, want false for non-temporal last-name phrase", query)
	}
	if !Query("What happened last week?") {
		t.Fatalf("Query(last week) = false, want temporal phrase retained")
	}
}

func TestTemporalMarkersRequireTokenBoundaries(t *testing.T) {
	query := "Maybe we should review honeymoon notes."
	markers := Markers(query)
	if len(markers) != 0 {
		t.Fatalf("Markers(%q) = %#v, want no month markers from substrings", query, markers)
	}
}

func TestTemporalQueryPhrasesRequireTokenBoundaries(t *testing.T) {
	for _, query := range []string{
		"What should I do whenever the deploy finishes?",
		"Where is the firstName field stored?",
	} {
		if Query(query) {
			t.Fatalf("Query(%q) = true, want false for temporal phrase substring", query)
		}
	}
	if !Query("When did the deploy finish?") {
		t.Fatalf("Query(when) = false, want temporal phrase retained")
	}
}

func TestTemporalMarkersMatchShortMonthAsToken(t *testing.T) {
	query := "What did I finish in May?"
	markers := Markers(query)
	if !stringSliceContains(markers, "may") {
		t.Fatalf("Markers(%q) = %#v, want may", query, markers)
	}
}

func TestTemporalRerankBonusRequiresMarkerBoundaries(t *testing.T) {
	features := Intent("What did I finish first in May?")
	bonus := RerankBonus(features, "user: Maybe I should finish the migration soon.", 0, 2, 1, 1)
	if bonus != 0 {
		t.Fatalf("RerankBonus with marker substring = %v, want 0", bonus)
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
