package textutil

import "testing"

func TestTruncateUTF8Bytes(t *testing.T) {
	tests := []struct {
		name  string
		value string
		limit int
		want  string
	}{
		{name: "no limit", value: "abcdef", limit: 0, want: "abcdef"},
		{name: "under limit", value: "abc", limit: 5, want: "abc"},
		{name: "ascii", value: "abcdef", limit: 3, want: "abc"},
		{name: "does not split rune", value: "ab🙂cd", limit: 5, want: "ab"},
		{name: "keeps whole rune", value: "ab🙂cd", limit: 6, want: "ab🙂"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TruncateUTF8Bytes(tt.value, tt.limit); got != tt.want {
				t.Fatalf("TruncateUTF8Bytes(%q, %d) = %q, want %q", tt.value, tt.limit, got, tt.want)
			}
		})
	}
}
