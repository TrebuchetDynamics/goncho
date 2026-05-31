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

func TestTemporalMarkersRequireTokenBoundaries(t *testing.T) {
	query := "Maybe we should review honeymoon notes."
	markers := Markers(query)
	if len(markers) != 0 {
		t.Fatalf("Markers(%q) = %#v, want no month markers from substrings", query, markers)
	}
}

func TestTemporalMarkersMatchShortMonthAsToken(t *testing.T) {
	query := "What did I finish in May?"
	markers := Markers(query)
	if !stringSliceContains(markers, "may") {
		t.Fatalf("Markers(%q) = %#v, want may", query, markers)
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
