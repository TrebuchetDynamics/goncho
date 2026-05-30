package limitutil

import "testing"

func TestDefault(t *testing.T) {
	if got := Default(0, 50); got != 50 {
		t.Fatalf("Default(0, 50) = %d, want 50", got)
	}
	if got := Default(-1, 50); got != 50 {
		t.Fatalf("Default(-1, 50) = %d, want 50", got)
	}
	if got := Default(42, 50); got != 42 {
		t.Fatalf("Default(42, 50) = %d, want 42", got)
	}
}

func TestDefaultClamped(t *testing.T) {
	tests := []struct {
		name         string
		limit        int
		defaultLimit int
		maxLimit     int
		want         int
	}{
		{name: "default for zero", limit: 0, defaultLimit: 50, maxLimit: 100, want: 50},
		{name: "default for negative", limit: -1, defaultLimit: 50, maxLimit: 100, want: 50},
		{name: "keeps valid", limit: 42, defaultLimit: 50, maxLimit: 100, want: 42},
		{name: "default above max", limit: 101, defaultLimit: 50, maxLimit: 100, want: 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultClamped(tt.limit, tt.defaultLimit, tt.maxLimit); got != tt.want {
				t.Fatalf("DefaultClamped(%d, %d, %d) = %d, want %d", tt.limit, tt.defaultLimit, tt.maxLimit, got, tt.want)
			}
		})
	}
}
