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
