package shared

import "testing"

func TestJSONRawIsEmptyOrNull(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want bool
	}{
		{name: "empty", raw: nil, want: true},
		{name: "whitespace", raw: []byte(" \n\t "), want: true},
		{name: "null", raw: []byte(" \n null \t"), want: true},
		{name: "object", raw: []byte(" {} "), want: false},
		{name: "string null", raw: []byte(`"null"`), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JSONRawIsEmptyOrNull(tt.raw); got != tt.want {
				t.Fatalf("JSONRawIsEmptyOrNull(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestTrimJSONRaw(t *testing.T) {
	if got := string(TrimJSONRaw([]byte(" \n {\"ok\":true} \t"))); got != `{"ok":true}` {
		t.Fatalf("TrimJSONRaw() = %q", got)
	}
}
