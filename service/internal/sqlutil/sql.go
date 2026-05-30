package sqlutil

import (
	"context"
	"database/sql"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

// LifecycleSQL is the minimal interface satisfied by both *sql.DB and *sql.Tx.
type LifecycleSQL interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// NullIfBlank returns nil for blank/whitespace values or the original string
// for non-blank values. Use when a blank string should be stored as SQL NULL.
func NullIfBlank(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// AppendInClause appends "column IN (?,?,...)" to a strings.Builder and adds
// the values to the args slice. It does not handle the leading AND/OR.
func AppendInClause(b *strings.Builder, column string, values []string, args *[]any) {
	b.WriteString(column)
	b.WriteString(` IN (`)
	for i, value := range values {
		if i > 0 {
			b.WriteString(`,`)
		}
		b.WriteString(`?`)
		*args = append(*args, value)
	}
	b.WriteString(`)`)
}

// ExecDeleteCount executes a DELETE query and returns the number of rows affected.
func ExecDeleteCount(ctx context.Context, db LifecycleSQL, query string, args ...any) (int64, error) {
	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return count, nil
}

// SessionKeyMatchesSources checks whether a sessionKey's source prefix matches
// one of the given sources. If sources is empty or contains a wildcard, returns true.
func SessionKeyMatchesSources(sessionKey string, sources []string) bool {
	if len(sources) == 0 || HasWildcard(sources) {
		return true
	}
	source, _, ok := strings.Cut(strings.TrimSpace(sessionKey), ":")
	if !ok {
		return false
	}
	return ContainsFold(sources, strings.ToLower(strings.TrimSpace(source)))
}

// OriginSourceFromChatKey extracts the source prefix from a "source:chatID" chat key.
func OriginSourceFromChatKey(chatKey string) string {
	chatKey = strings.TrimSpace(chatKey)
	idx := strings.Index(chatKey, ":")
	if idx <= 0 {
		return ""
	}
	return chatKey[:idx]
}

// HasWildcard reports whether values contains "*".
func HasWildcard(values []string) bool {
	return sliceutil.ContainsFunc(values, func(value string) bool {
		return strings.TrimSpace(value) == "*"
	})
}

// ContainsFold reports whether slice contains value (case-insensitive comparison).
func ContainsFold(slice []string, value string) bool {
	return sliceutil.ContainsFunc(slice, func(item string) bool {
		return strings.EqualFold(strings.TrimSpace(item), value)
	})
}

// IsSQLiteNoSuchTableError reports whether err is SQLite's missing-table error.
func IsSQLiteNoSuchTableError(err error) bool {
	return errorContainsFold(err, []string{"no such table"})
}

// IsSQLiteDuplicateColumnError reports whether err is SQLite's duplicate-column migration error.
func IsSQLiteDuplicateColumnError(err error) bool {
	return errorContainsFold(err, []string{"duplicate column name"})
}

// IsSQLiteTransientLockError reports whether err is a retryable SQLite lock/busy error.
func IsSQLiteTransientLockError(err error) bool {
	return errorContainsFold(err, []string{
		"database is locked",
		"database table is locked",
		"database is busy",
		"database table is busy",
		"sqlite_busy",
		"sqlite_locked",
	})
}

func errorContainsFold(err error, markers []string) bool {
	return err != nil && textutil.ContainsAnySubstringFold(err.Error(), markers)
}
