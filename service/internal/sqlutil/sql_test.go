package sqlutil

import (
	"errors"
	"testing"
)

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
