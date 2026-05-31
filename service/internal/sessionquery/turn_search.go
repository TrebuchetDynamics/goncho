package sessionquery

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/sqlutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

// BuildTurnSearchQuery builds the SQLite query used to search synced session turns.
func BuildTurnSearchQuery(rawQuery string, sessionIDs, chatKeys, roles []string, limit int, sessionsOnly bool) (string, []any) {
	var b strings.Builder
	args := make([]any, 0, len(sessionIDs)+len(chatKeys)+len(roles)+2)
	if sessionsOnly {
		b.WriteString(`SELECT t.session_id, t.chat_id, MAX(t.ts_unix) AS latest_turn_unix FROM turns t`)
	} else {
		b.WriteString(`SELECT t.session_id, t.chat_id, t.role, t.content, t.ts_unix FROM turns t`)
	}

	query := sqlutil.SanitizeFTS5Pattern(rawQuery)
	if query != "" {
		b.WriteString(` JOIN turns_fts fts ON fts.rowid = t.id WHERE turns_fts MATCH ?`)
		args = append(args, query)
	} else {
		b.WriteString(` WHERE 1=1`)
	}

	b.WriteString(` AND (`)
	sqlutil.AppendInClause(&b, "t.session_id", sessionIDs, &args)
	if len(chatKeys) > 0 {
		b.WriteString(` OR `)
		sqlutil.AppendInClause(&b, "t.chat_id", chatKeys, &args)
	}
	b.WriteString(`)`)
	b.WriteString(` AND t.memory_sync_status = 'ready'`)
	if normalizedRoles := normalizeRoles(roles); len(normalizedRoles) > 0 {
		b.WriteString(` AND `)
		sqlutil.AppendInClause(&b, "t.role", normalizedRoles, &args)
	}

	if sessionsOnly {
		b.WriteString(` GROUP BY t.session_id, t.chat_id ORDER BY latest_turn_unix DESC, t.session_id ASC LIMIT ?`)
	} else {
		b.WriteString(` ORDER BY t.ts_unix DESC, t.id DESC LIMIT ?`)
	}
	args = append(args, limit)
	return b.String(), args
}

func normalizeRoles(roles []string) []string {
	return textutil.UniqueLowerTrimmed(roles, false)
}
