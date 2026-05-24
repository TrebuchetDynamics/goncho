package memorypolicy

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Tier identifies the governance level that owns a memory item.
type Tier string

const (
	TierGlobal    Tier = "global"
	TierProject   Tier = "project"
	TierTask      Tier = "task"
	TierWorkspace Tier = "workspace"
	TierDecision  Tier = "decision"
)

// ValidTiers is the ordered set of accepted memory governance tiers.
var ValidTiers = []Tier{
	TierGlobal, TierProject, TierTask, TierWorkspace, TierDecision,
}

// ValidTier reports whether raw names one of Goncho's memory governance tiers.
func ValidTier(raw string) bool {
	switch Tier(strings.ToLower(strings.TrimSpace(raw))) {
	case TierGlobal, TierProject, TierTask, TierWorkspace, TierDecision:
		return true
	default:
		return false
	}
}

// NormalizeTier returns a valid tier, defaulting unknown input to global.
func NormalizeTier(raw string) Tier {
	tier := Tier(strings.ToLower(strings.TrimSpace(raw)))
	if !ValidTier(string(tier)) {
		return TierGlobal
	}
	return tier
}

// Hierarchy returns tiers from broadest/global to most governed/decision.
func Hierarchy() []Tier {
	return []Tier{TierGlobal, TierProject, TierTask, TierWorkspace, TierDecision}
}

// ReadableBy returns the tier window visible to an agent at agentTier.
func ReadableBy(agentTier Tier) []Tier {
	switch agentTier {
	case TierGlobal:
		return []Tier{TierGlobal}
	case TierProject:
		return []Tier{TierGlobal, TierProject}
	case TierTask:
		return []Tier{TierGlobal, TierProject, TierTask}
	case TierWorkspace:
		return []Tier{TierGlobal, TierProject, TierTask, TierWorkspace}
	case TierDecision:
		return []Tier{TierGlobal, TierProject, TierTask, TierWorkspace, TierDecision}
	default:
		return nil
	}
}

// WritableBy returns the tiers an agent may write directly.
func WritableBy(isParent bool) []Tier {
	if isParent {
		return Hierarchy()
	}
	return []Tier{TierWorkspace}
}

// DefaultTierForSource assigns source kinds to conservative memory tiers.
func DefaultTierForSource(sourceKind string) Tier {
	switch strings.ToLower(strings.TrimSpace(sourceKind)) {
	case "manual", "import":
		return TierProject
	case "tool", "runtime":
		return TierTask
	case "derived":
		return TierDecision
	case "reviewed_proposal":
		return TierProject
	default:
		return TierGlobal
	}
}

// ValidateTierOrErr rejects unknown tier names with a public error message.
func ValidateTierOrErr(raw string) error {
	if !ValidTier(raw) {
		return fmt.Errorf("invalid memory tier %q: must be one of global, project, task, workspace, decision", raw)
	}
	return nil
}

// ACLQuery captures the caller's memory tier window and explicit ACL identity.
type ACLQuery struct {
	AgentID     string
	IsParent    bool
	ReadTiers   []Tier
	WriteTier   Tier
	WorkspaceID string
}

// ReadScopeSQL returns the SQL clause and args for readable memory rows.
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

// CanWrite reports whether the query identity can write tier.
func (q ACLQuery) CanWrite(tier Tier) bool {
	if q.IsParent {
		return true
	}
	return tier == TierWorkspace && q.WriteTier == TierWorkspace
}

// CanRead reports whether the query identity can read tier through its tier window.
func (q ACLQuery) CanRead(tier Tier) bool {
	for _, readable := range q.ReadTiers {
		if readable == tier {
			return true
		}
	}
	return false
}

// GrantReadACL grants one agent explicit read access to one memory item.
func GrantReadACL(ctx context.Context, db *sql.DB, memoryID, agentID, grantedBy string) error {
	_, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO memory_acl(memory_id, agent_id, permission, granted_by, granted_at) VALUES(?, ?, 'read', ?, unixepoch())`, memoryID, agentID, grantedBy)
	return err
}

// RevokeACL removes one explicit ACL grant.
func RevokeACL(ctx context.Context, db *sql.DB, memoryID, agentID, permission string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM memory_acl WHERE memory_id = ? AND agent_id = ? AND permission = ?`, memoryID, agentID, permission)
	return err
}

// AgentCanAccessMemory checks explicit ACL grants, workspace tier visibility,
// and the agent's own workspace scratch memory allowance.
func AgentCanAccessMemory(ctx context.Context, db *sql.DB, memoryID, agentID, workspaceID string, readableTiers []Tier) (bool, error) {
	tierPlaceholders := make([]string, len(readableTiers))
	tierArgs := make([]any, len(readableTiers))
	for i, tier := range readableTiers {
		tierPlaceholders[i] = "?"
		tierArgs[i] = string(tier)
	}
	allArgs := append([]any{memoryID, agentID}, tierArgs...)
	allArgs = append(allArgs, workspaceID, agentID, agentID)
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM goncho_memory_items m WHERE m.memory_id = ?1 AND m.active = 1 AND (EXISTS(SELECT 1 FROM memory_acl acl WHERE acl.memory_id = m.memory_id AND acl.agent_id = ?2 AND acl.permission = 'read') OR (m.workspace_id = ?%d AND m.tier IN (%s)) OR (m.agent_id = ?%d AND m.tier = 'workspace')))`, 2+len(tierArgs)+1, strings.Join(tierPlaceholders, ","), 2+len(tierArgs)+2)
	var exists bool
	err := db.QueryRowContext(ctx, query, allArgs...).Scan(&exists)
	return exists, err
}
