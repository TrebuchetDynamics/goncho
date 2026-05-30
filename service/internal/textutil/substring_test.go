package textutil

import "testing"

func TestContainsAnySubstring(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		markers []string
		want    bool
	}{
		{name: "matches one marker", value: "remember the latest plan", markers: []string{"current", "latest"}, want: true},
		{name: "empty markers do not match", value: "remember the latest plan", markers: nil, want: false},
		{name: "no marker", value: "remember the plan", markers: []string{"current", "latest"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsAnySubstring(tt.value, tt.markers); got != tt.want {
				t.Fatalf("ContainsAnySubstring(%q, %v) = %v, want %v", tt.value, tt.markers, got, tt.want)
			}
		})
	}
}

func TestContainsAnySubstringFold(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		markers []string
		want    bool
	}{
		{name: "case folded marker", value: "Authorization: Bearer abc123", markers: []string{"bearer "}, want: true},
		{name: "case folded value and marker", value: "Remember the API TOKEN", markers: []string{"api token"}, want: true},
		{name: "empty markers do not match", value: "Authorization: Bearer abc123", markers: nil, want: false},
		{name: "no marker", value: "ordinary preference", markers: []string{"secret"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsAnySubstringFold(tt.value, tt.markers); got != tt.want {
				t.Fatalf("ContainsAnySubstringFold(%q, %v) = %v, want %v", tt.value, tt.markers, got, tt.want)
			}
		})
	}
}

func TestContainsAllSubstringsFold(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		markers []string
		want    bool
	}{
		{name: "all case folded markers", value: "Graph edge activates Auth Owner recall", markers: []string{"graph", " AUTH owner "}, want: true},
		{name: "blank markers ignored", value: "Graph edge", markers: []string{"", " edge "}, want: true},
		{name: "empty markers match vacuously", value: "Graph edge", markers: nil, want: true},
		{name: "missing marker", value: "Graph edge", markers: []string{"graph", "owner"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsAllSubstringsFold(tt.value, tt.markers); got != tt.want {
				t.Fatalf("ContainsAllSubstringsFold(%q, %v) = %v, want %v", tt.value, tt.markers, got, tt.want)
			}
		})
	}
}
