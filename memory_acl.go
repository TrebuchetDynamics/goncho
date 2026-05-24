package goncho

import (
	"context"
	"database/sql"

	"github.com/TrebuchetDynamics/goncho/internal/memorypolicy"
)

type ACLQuery = memorypolicy.ACLQuery

func GrantReadACL(ctx context.Context, db *sql.DB, memoryID, agentID, grantedBy string) error {
	return memorypolicy.GrantReadACL(ctx, db, memoryID, agentID, grantedBy)
}

func RevokeACL(ctx context.Context, db *sql.DB, memoryID, agentID, permission string) error {
	return memorypolicy.RevokeACL(ctx, db, memoryID, agentID, permission)
}

func AgentCanAccessMemory(ctx context.Context, db *sql.DB, memoryID, agentID, workspaceID string, readableTiers []MemoryTier) (bool, error) {
	return memorypolicy.AgentCanAccessMemory(ctx, db, memoryID, agentID, workspaceID, readableTiers)
}
