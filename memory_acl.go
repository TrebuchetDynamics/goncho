package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type ACLQuery struct {
	AgentID     string
	IsParent    bool
	ReadTiers   []MemoryTier
	WriteTier   MemoryTier
	WorkspaceID string
}

func (q ACLQuery) ReadScopeSQL() (string, []any) {
	if len(q.ReadTiers) == 0 {
		return "1 = 0", nil
	}
	var parts []string
	var args []any
	parts = append(parts, `m.memory_id IN (SELECT acl.memory_id FROM memory_acl acl WHERE acl.agent_id = ? AND acl.permission = 'read')`)
	args = append(args, q.AgentID)

	args = append(args, q.WorkspaceID)
	placeholders := make([]string, len(q.ReadTiers))
	for i := range q.ReadTiers {
		placeholders[i] = "?"
		args = append(args, string(q.ReadTiers[i]))
	}
	parts = append(parts, fmt.Sprintf(`(m.workspace_id = ? AND m.tier IN (%s))`, strings.Join(placeholders, ",")))

	parts = append(parts, `(m.agent_id = ? AND m.tier = 'workspace')`)
	args = append(args, q.AgentID)
	return "(" + strings.Join(parts, " OR ") + ")", args
}

func (q ACLQuery) CanWrite(tier MemoryTier) bool {
	if q.IsParent {
		return true
	}
	return tier == TierWorkspace && q.WriteTier == TierWorkspace
}

func (q ACLQuery) CanRead(tier MemoryTier) bool {
	for _, t := range q.ReadTiers {
		if t == tier {
			return true
		}
	}
	return false
}

func GrantReadACL(ctx context.Context, db *sql.DB, memoryID, agentID, grantedBy string) error {
	_, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO memory_acl(memory_id, agent_id, permission, granted_by, granted_at) VALUES(?, ?, 'read', ?, unixepoch())`, memoryID, agentID, grantedBy)
	return err
}

func RevokeACL(ctx context.Context, db *sql.DB, memoryID, agentID, permission string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM memory_acl WHERE memory_id = ? AND agent_id = ? AND permission = ?`, memoryID, agentID, permission)
	return err
}

func AgentCanAccessMemory(ctx context.Context, db *sql.DB, memoryID, agentID, workspaceID string, readableTiers []MemoryTier) (bool, error) {
	tierPlaceholders := make([]string, len(readableTiers))
	tierArgs := make([]any, len(readableTiers))
	for i, t := range readableTiers {
		tierPlaceholders[i] = "?"
		tierArgs[i] = string(t)
	}
	allArgs := append([]any{memoryID, agentID}, tierArgs...)
	allArgs = append(allArgs, workspaceID, agentID, agentID)
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM goncho_memory_items m WHERE m.memory_id = ?1 AND m.active = 1 AND (EXISTS(SELECT 1 FROM memory_acl acl WHERE acl.memory_id = m.memory_id AND acl.agent_id = ?2 AND acl.permission = 'read') OR (m.workspace_id = ?%d AND m.tier IN (%s)) OR (m.agent_id = ?%d AND m.tier = 'workspace')))`, 2+len(tierArgs)+1, strings.Join(tierPlaceholders, ","), 2+len(tierArgs)+2)
	var exists bool
	err := db.QueryRowContext(ctx, query, allArgs...).Scan(&exists)
	return exists, err
}
