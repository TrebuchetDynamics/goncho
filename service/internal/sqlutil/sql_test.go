package sqlutil

import (
	"errors"
	"testing"
)

func TestSanitizeFTS5Pattern(t *testing.T) {
	got := SanitizeFTS5Pattern(` owner:(alice) +tag "quoted" path/a.go* `)
	want := `owner  alice  +tag "quoted" path a go*`
	if got != want {
		t.Fatalf("SanitizeFTS5Pattern() = %q, want %q", got, want)
	}
}

func TestSessionKeyMatchesSourcesAllowsTurnKindOrAdapterPrefix(t *testing.T) {
	if !SessionKeyMatchesSources("discord:chan-9", []string{"turn"}) {
		t.Fatalf("source kind turn should allow any adapter-backed turn session")
	}
	if !SessionKeyMatchesSources("discord:chan-9", []string{"discord"}) {
		t.Fatalf("adapter source discord should allow matching session prefix")
	}
	if SessionKeyMatchesSources("discord:chan-9", []string{"conclusion"}) {
		t.Fatalf("conclusion source should not allow turn fallback")
	}
}

func TestSQLiteErrorClassifiers(t *testing.T) {
	tests := []struct {
		name string
		err  error
		fn   func(error) bool
		want bool
	}{
		{name: "nil", err: nil, fn: IsSQLiteNoSuchTableError, want: false},
		{name: "no such table", err: errors.New("SQLITE_ERROR: no such table: goncho_conclusions"), fn: IsSQLiteNoSuchTableError, want: true},
		{name: "duplicate column", err: errors.New("duplicate column name: retention_expires_at"), fn: IsSQLiteDuplicateColumnError, want: true},
		{name: "transient lock", err: errors.New("database table is LOCKED"), fn: IsSQLiteTransientLockError, want: true},
		{name: "unrelated", err: errors.New("constraint failed"), fn: IsSQLiteTransientLockError, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(tt.err); got != tt.want {
				t.Fatalf("classifier returned %v, want %v", got, tt.want)
			}
		})
	}
}
