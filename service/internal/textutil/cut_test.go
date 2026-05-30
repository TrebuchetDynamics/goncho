package textutil

import "testing"

func TestCutAnyPrefixFold(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		prefixes []string
		wantTail string
		wantOK   bool
	}{
		{name: "matches first prefix case-insensitively", value: "Where Is Vault", prefixes: []string{"where is ", "where are "}, wantTail: "Vault", wantOK: true},
		{name: "preserves original tail spacing", value: "HOW MANY open tickets", prefixes: []string{"how many "}, wantTail: "open tickets", wantOK: true},
		{name: "no match", value: "why vault", prefixes: []string{"where is "}, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTail, gotOK := CutAnyPrefixFold(tt.value, tt.prefixes)
			if gotTail != tt.wantTail || gotOK != tt.wantOK {
				t.Fatalf("CutAnyPrefixFold(%q, %v) = (%q, %v), want (%q, %v)", tt.value, tt.prefixes, gotTail, gotOK, tt.wantTail, tt.wantOK)
			}
		})
	}
}

func TestCutAroundAnySubstringFold(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		markers    []string
		wantBefore string
		wantMarker string
		wantAfter  string
		wantOK     bool
	}{
		{name: "splits around folded marker", value: "Vault Is Located At Shelf 3", markers: []string{" is located at "}, wantBefore: "Vault", wantMarker: " is located at ", wantAfter: "Shelf 3", wantOK: true},
		{name: "uses first marker in policy order", value: "A is in B and lives in C", markers: []string{" lives in ", " is in "}, wantBefore: "A is in B and", wantMarker: " lives in ", wantAfter: "C", wantOK: true},
		{name: "no marker", value: "Vault near shelf", markers: []string{" is in "}, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBefore, gotAfter, gotOK := CutAroundAnySubstringFold(tt.value, tt.markers)
			if gotBefore != tt.wantBefore || gotAfter != tt.wantAfter || gotOK != tt.wantOK {
				t.Fatalf("CutAroundAnySubstringFold(%q, %v) = (%q, %q, %v), want (%q, %q, %v)", tt.value, tt.markers, gotBefore, gotAfter, gotOK, tt.wantBefore, tt.wantAfter, tt.wantOK)
			}
			gotBefore, gotMarker, gotAfter, gotOK := CutAroundAnySubstringFoldMatch(tt.value, tt.markers)
			if gotBefore != tt.wantBefore || gotMarker != tt.wantMarker || gotAfter != tt.wantAfter || gotOK != tt.wantOK {
				t.Fatalf("CutAroundAnySubstringFoldMatch(%q, %v) = (%q, %q, %q, %v), want (%q, %q, %q, %v)", tt.value, tt.markers, gotBefore, gotMarker, gotAfter, gotOK, tt.wantBefore, tt.wantMarker, tt.wantAfter, tt.wantOK)
			}
		})
	}
}
