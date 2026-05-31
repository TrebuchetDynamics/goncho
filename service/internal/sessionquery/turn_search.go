package sessionquery

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/sqlutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

// TurnSearchScope is the explicit identifier set allowed to match synced session turns.
// Empty scopes are valid inputs and intentionally compile to a no-row predicate.
type TurnSearchScope struct {
	SessionIDs []string
	ChatKeys   []string
}

func newTurnSearchScope(sessionIDs, chatKeys []string) TurnSearchScope {
	return TurnSearchScope{
		SessionIDs: textutil.UniqueTrimmed(sessionIDs, false),
		ChatKeys:   textutil.UniqueTrimmed(chatKeys, false),
	}
}

func (scope TurnSearchScope) appendPredicate(b *strings.Builder, args *[]any) {
	if len(scope.SessionIDs) == 0 && len(scope.ChatKeys) == 0 {
		b.WriteString(`0=1`)
		return
	}
	b.WriteString(`(`)
	if len(scope.SessionIDs) > 0 {
		sqlutil.AppendInClause(b, "t.session_id", scope.SessionIDs, args)
	}
	if len(scope.ChatKeys) > 0 {
		if len(scope.SessionIDs) > 0 {
			b.WriteString(` OR `)
		}
		sqlutil.AppendInClause(b, "t.chat_id", scope.ChatKeys, args)
	}
	b.WriteString(`)`)
}

type turnQueryPlan struct {
	RawQuery   string
	FTSPattern string
}

// planTurnQuery keeps raw non-blank queries distinct from their FTS pattern.
// FTS sanitation can erase punctuation-only input; callers must still keep that
// input constrained instead of silently treating it as an unfiltered browse.
func planTurnQuery(rawQuery string) turnQueryPlan {
	trimmed := strings.TrimSpace(rawQuery)
	if trimmed == "" {
		return turnQueryPlan{}
	}
	return turnQueryPlan{
		RawQuery:   trimmed,
		FTSPattern: sqlutil.SanitizeFTS5Pattern(trimmed),
	}
}

// BuildTurnSearchQuery builds the SQLite query used to search synced session turns.
func BuildTurnSearchQuery(rawQuery string, sessionIDs, chatKeys, roles []string, limit int, sessionsOnly bool) (string, []any) {
	var b strings.Builder
	scope := newTurnSearchScope(sessionIDs, chatKeys)
	args := make([]any, 0, len(scope.SessionIDs)+len(scope.ChatKeys)+len(roles)+2)
	if sessionsOnly {
		b.WriteString(`SELECT t.session_id, t.chat_id, MAX(t.ts_unix) AS latest_turn_unix FROM turns t`)
	} else {
		b.WriteString(`SELECT t.session_id, t.chat_id, t.role, t.content, t.ts_unix FROM turns t`)
	}

	query := planTurnQuery(rawQuery)
	if query.FTSPattern != "" {
		b.WriteString(` JOIN turns_fts fts ON fts.rowid = t.id WHERE turns_fts MATCH ?`)
		args = append(args, query.FTSPattern)
	} else if query.RawQuery != "" {
		b.WriteString(` WHERE t.content LIKE ?`)
		args = append(args, "%"+query.RawQuery+"%")
	} else {
		b.WriteString(` WHERE 1=1`)
	}

	b.WriteString(` AND `)
	scope.appendPredicate(&b, &args)
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
