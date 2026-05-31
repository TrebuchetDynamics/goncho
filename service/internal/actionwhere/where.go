package actionwhere

// ScopedAction returns the standard SQL WHERE clause and args for action-scoped
// tables. workspace_id/profile_id/peer_id are always required; action_id is
// appended when non-empty.
func ScopedAction(workspaceID, profileID, peer, actionID string) (string, []any) {
	where := `workspace_id = ? AND profile_id = ? AND peer_id = ?`
	args := []any{workspaceID, profileID, peer}
	if actionID != "" {
		where += ` AND action_id = ?`
		args = append(args, actionID)
	}
	return where, args
}

// ScopedActionSignal returns the standard SQL WHERE clause and args for
// action-scoped signal rows. The action_id predicate is always included;
// signal_id is appended when non-zero.
func ScopedActionSignal(workspaceID, profileID, peer, actionID string, signalID int64) (string, []any) {
	where := `workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ?`
	args := []any{workspaceID, profileID, peer, actionID}
	if signalID != 0 {
		where += ` AND signal_id = ?`
		args = append(args, signalID)
	}
	return where, args
}
