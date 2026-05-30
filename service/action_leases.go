package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/scopekey"
)

type ActionLeaseDecision string

const (
	ActionLeaseDecisionAcquired    ActionLeaseDecision = "acquired"
	ActionLeaseDecisionHeldByOther ActionLeaseDecision = "held_by_other"
	ActionLeaseDecisionRenewed     ActionLeaseDecision = "renewed"
	ActionLeaseDecisionExpired     ActionLeaseDecision = "expired"
)

var actionLeaseDDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_action_leases (
		workspace_id TEXT NOT NULL,
		profile_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL,
		action_id TEXT NOT NULL,
		owner TEXT NOT NULL,
		acquired_at INTEGER NOT NULL,
		renewed_at INTEGER NOT NULL,
		expires_at INTEGER NOT NULL,
		PRIMARY KEY(workspace_id, profile_id, peer_id, action_id)
	)`,
	`CREATE TABLE IF NOT EXISTS goncho_action_lease_audit (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id TEXT NOT NULL,
		profile_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL,
		action_id TEXT NOT NULL,
		actor TEXT NOT NULL,
		decision TEXT NOT NULL,
		reason TEXT NOT NULL DEFAULT '',
		expires_at INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_action_leases_expiry ON goncho_action_leases(expires_at)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_action_lease_audit_scope ON goncho_action_lease_audit(workspace_id, profile_id, peer_id, action_id, created_at DESC)`,
}

type ActionLeaseParams struct {
	WorkspaceID string        `json:"workspace_id,omitempty"`
	ProfileID   string        `json:"profile_id,omitempty"`
	Peer        string        `json:"peer"`
	ActionID    string        `json:"action_id"`
	Owner       string        `json:"owner"`
	TTL         time.Duration `json:"ttl"`
}

type ActionLeaseExpireParams struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	ActionID    string `json:"action_id,omitempty"`
}

type ActionLeaseAuditQuery struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	ActionID    string `json:"action_id,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type ActionLease struct {
	WorkspaceID string `json:"workspace_id"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	ActionID    string `json:"action_id"`
	Owner       string `json:"owner"`
	AcquiredAt  int64  `json:"acquired_at"`
	RenewedAt   int64  `json:"renewed_at"`
	ExpiresAt   int64  `json:"expires_at"`
}

type ActionLeaseResult struct {
	Acquired bool                `json:"acquired"`
	Decision ActionLeaseDecision `json:"decision"`
	Lease    ActionLease         `json:"lease"`
	Reason   string              `json:"reason,omitempty"`
	AuditID  int64               `json:"audit_id,omitempty"`
}

type ActionLeaseAuditEvent struct {
	ID          int64               `json:"id"`
	WorkspaceID string              `json:"workspace_id"`
	ProfileID   string              `json:"profile_id,omitempty"`
	Peer        string              `json:"peer"`
	ActionID    string              `json:"action_id"`
	Actor       string              `json:"actor"`
	Decision    ActionLeaseDecision `json:"decision"`
	Reason      string              `json:"reason,omitempty"`
	ExpiresAt   int64               `json:"expires_at,omitempty"`
	CreatedAt   int64               `json:"created_at"`
}

type ActionLeaseAuditResult struct {
	Events []ActionLeaseAuditEvent `json:"events"`
	Count  int                     `json:"count"`
}

type ActionLeaseExpireResult struct {
	ExpiredCount int                     `json:"expired_count"`
	Events       []ActionLeaseAuditEvent `json:"events"`
}

func (s *Service) AcquireActionLease(ctx context.Context, params ActionLeaseParams) (ActionLeaseResult, error) {
	norm, err := s.normalizeActionLeaseParams(params)
	if err != nil {
		return ActionLeaseResult{}, err
	}
	now := time.Now().UnixNano()
	expiresAt := now + params.TTL.Nanoseconds()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ActionLeaseResult{}, fmt.Errorf("goncho: begin action lease acquire: %w", err)
	}
	defer tx.Rollback()
	if err := ensureActionExistsTx(ctx, tx, norm); err != nil {
		return ActionLeaseResult{}, err
	}
	existing, found, err := getActionLeaseTx(ctx, tx, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.ActionID)
	if err != nil {
		return ActionLeaseResult{}, err
	}
	if found && existing.ExpiresAt > now && existing.Owner != norm.Owner {
		reason := fmt.Sprintf("action lease held by %s until %d", existing.Owner, existing.ExpiresAt)
		auditID, err := insertActionLeaseAuditTx(ctx, tx, existing, norm.Owner, ActionLeaseDecisionHeldByOther, reason, now)
		if err != nil {
			return ActionLeaseResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return ActionLeaseResult{}, fmt.Errorf("goncho: commit action lease denial: %w", err)
		}
		return ActionLeaseResult{Decision: ActionLeaseDecisionHeldByOther, Lease: existing, Reason: reason, AuditID: auditID}, nil
	}
	lease := ActionLease{WorkspaceID: norm.WorkspaceID, ProfileID: norm.ProfileID, Peer: norm.Peer, ActionID: norm.ActionID, Owner: norm.Owner, AcquiredAt: now, RenewedAt: now, ExpiresAt: expiresAt}
	if found && existing.Owner == norm.Owner && existing.AcquiredAt > 0 {
		lease.AcquiredAt = existing.AcquiredAt
	}
	if err := upsertActionLeaseTx(ctx, tx, lease); err != nil {
		return ActionLeaseResult{}, err
	}
	auditID, err := insertActionLeaseAuditTx(ctx, tx, lease, norm.Owner, ActionLeaseDecisionAcquired, "lease acquired", now)
	if err != nil {
		return ActionLeaseResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ActionLeaseResult{}, fmt.Errorf("goncho: commit action lease acquire: %w", err)
	}
	return ActionLeaseResult{Acquired: true, Decision: ActionLeaseDecisionAcquired, Lease: lease, AuditID: auditID}, nil
}

func (s *Service) RenewActionLease(ctx context.Context, params ActionLeaseParams) (ActionLeaseResult, error) {
	norm, err := s.normalizeActionLeaseParams(params)
	if err != nil {
		return ActionLeaseResult{}, err
	}
	now := time.Now().UnixNano()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ActionLeaseResult{}, fmt.Errorf("goncho: begin action lease renew: %w", err)
	}
	defer tx.Rollback()
	if err := ensureActionExistsTx(ctx, tx, norm); err != nil {
		return ActionLeaseResult{}, err
	}
	existing, found, err := getActionLeaseTx(ctx, tx, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.ActionID)
	if err != nil {
		return ActionLeaseResult{}, err
	}
	if !found || existing.ExpiresAt <= now {
		reason := "action lease is missing or expired"
		base := ActionLease{WorkspaceID: norm.WorkspaceID, ProfileID: norm.ProfileID, Peer: norm.Peer, ActionID: norm.ActionID, Owner: norm.Owner}
		auditID, err := insertActionLeaseAuditTx(ctx, tx, base, norm.Owner, ActionLeaseDecisionExpired, reason, now)
		if err != nil {
			return ActionLeaseResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return ActionLeaseResult{}, fmt.Errorf("goncho: commit action lease renew miss: %w", err)
		}
		return ActionLeaseResult{Decision: ActionLeaseDecisionExpired, Lease: base, Reason: reason, AuditID: auditID}, nil
	}
	if existing.Owner != norm.Owner {
		reason := fmt.Sprintf("action lease held by %s until %d", existing.Owner, existing.ExpiresAt)
		auditID, err := insertActionLeaseAuditTx(ctx, tx, existing, norm.Owner, ActionLeaseDecisionHeldByOther, reason, now)
		if err != nil {
			return ActionLeaseResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return ActionLeaseResult{}, fmt.Errorf("goncho: commit action lease renew denial: %w", err)
		}
		return ActionLeaseResult{Decision: ActionLeaseDecisionHeldByOther, Lease: existing, Reason: reason, AuditID: auditID}, nil
	}
	existing.RenewedAt = now
	existing.ExpiresAt = now + params.TTL.Nanoseconds()
	if err := upsertActionLeaseTx(ctx, tx, existing); err != nil {
		return ActionLeaseResult{}, err
	}
	auditID, err := insertActionLeaseAuditTx(ctx, tx, existing, norm.Owner, ActionLeaseDecisionRenewed, "lease renewed", now)
	if err != nil {
		return ActionLeaseResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ActionLeaseResult{}, fmt.Errorf("goncho: commit action lease renew: %w", err)
	}
	return ActionLeaseResult{Acquired: true, Decision: ActionLeaseDecisionRenewed, Lease: existing, AuditID: auditID}, nil
}

func (s *Service) ExpireActionLeases(ctx context.Context, params ActionLeaseExpireParams) (ActionLeaseExpireResult, error) {
	norm, err := s.normalizeActionLeaseExpireParams(params)
	if err != nil {
		return ActionLeaseExpireResult{}, err
	}
	now := time.Now().UnixNano()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ActionLeaseExpireResult{}, fmt.Errorf("goncho: begin action lease expiry: %w", err)
	}
	defer tx.Rollback()
	leases, err := listExpiredActionLeasesTx(ctx, tx, norm, now)
	if err != nil {
		return ActionLeaseExpireResult{}, err
	}
	events := make([]ActionLeaseAuditEvent, 0, len(leases))
	for _, lease := range leases {
		if _, err := tx.ExecContext(ctx, `DELETE FROM goncho_action_leases WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ?`, lease.WorkspaceID, lease.ProfileID, lease.Peer, lease.ActionID); err != nil {
			return ActionLeaseExpireResult{}, fmt.Errorf("goncho: delete expired action lease: %w", err)
		}
		auditID, err := insertActionLeaseAuditTx(ctx, tx, lease, lease.Owner, ActionLeaseDecisionExpired, "lease expired", now)
		if err != nil {
			return ActionLeaseExpireResult{}, err
		}
		events = append(events, ActionLeaseAuditEvent{ID: auditID, WorkspaceID: lease.WorkspaceID, ProfileID: lease.ProfileID, Peer: lease.Peer, ActionID: lease.ActionID, Actor: lease.Owner, Decision: ActionLeaseDecisionExpired, Reason: "lease expired", ExpiresAt: lease.ExpiresAt, CreatedAt: now})
	}
	if err := tx.Commit(); err != nil {
		return ActionLeaseExpireResult{}, fmt.Errorf("goncho: commit action lease expiry: %w", err)
	}
	return ActionLeaseExpireResult{ExpiredCount: len(events), Events: events}, nil
}

func (s *Service) ListActionLeaseAudit(ctx context.Context, query ActionLeaseAuditQuery) (ActionLeaseAuditResult, error) {
	norm, err := s.normalizeActionLeaseAuditQuery(query)
	if err != nil {
		return ActionLeaseAuditResult{}, err
	}
	limit := limitutil.DefaultClamped(query.Limit, 100, 100)
	args := []any{norm.WorkspaceID, norm.ProfileID, norm.Peer}
	where := `workspace_id = ? AND profile_id = ? AND peer_id = ?`
	if norm.ActionID != "" {
		where += ` AND action_id = ?`
		args = append(args, norm.ActionID)
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, profile_id, peer_id, action_id, actor, decision, reason, expires_at, created_at FROM goncho_action_lease_audit WHERE `+where+` ORDER BY created_at DESC, id DESC LIMIT ?`, args...)
	if err != nil {
		return ActionLeaseAuditResult{}, fmt.Errorf("goncho: list action lease audit: %w", err)
	}
	defer rows.Close()
	out := []ActionLeaseAuditEvent{}
	for rows.Next() {
		var event ActionLeaseAuditEvent
		var decision string
		if err := rows.Scan(&event.ID, &event.WorkspaceID, &event.ProfileID, &event.Peer, &event.ActionID, &event.Actor, &decision, &event.Reason, &event.ExpiresAt, &event.CreatedAt); err != nil {
			return ActionLeaseAuditResult{}, fmt.Errorf("goncho: scan action lease audit: %w", err)
		}
		event.Decision = ActionLeaseDecision(decision)
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return ActionLeaseAuditResult{}, fmt.Errorf("goncho: iterate action lease audit: %w", err)
	}
	return ActionLeaseAuditResult{Events: out, Count: len(out)}, nil
}

func (s *Service) normalizeActionLeaseParams(params ActionLeaseParams) (ActionLease, error) {
	scope := scopekey.Normalize(s.workspaceID, params.WorkspaceID, params.ProfileID, params.Peer)
	actionID := strings.TrimSpace(params.ActionID)
	owner := strings.TrimSpace(params.Owner)
	if !scope.Complete() || actionID == "" || owner == "" {
		return ActionLease{}, fmt.Errorf("goncho: action lease workspace_id, peer, action_id, and owner are required")
	}
	if params.TTL <= 0 {
		return ActionLease{}, fmt.Errorf("goncho: action lease ttl must be positive")
	}
	return ActionLease{WorkspaceID: scope.WorkspaceID, ProfileID: scope.ProfileID, Peer: scope.Peer, ActionID: actionID, Owner: owner}, nil
}

func (s *Service) normalizeActionLeaseExpireParams(params ActionLeaseExpireParams) (ActionLease, error) {
	scope := scopekey.Normalize(s.workspaceID, params.WorkspaceID, params.ProfileID, params.Peer)
	if !scope.Complete() {
		return ActionLease{}, fmt.Errorf("goncho: action lease expiry workspace_id and peer are required")
	}
	return ActionLease{WorkspaceID: scope.WorkspaceID, ProfileID: scope.ProfileID, Peer: scope.Peer, ActionID: strings.TrimSpace(params.ActionID)}, nil
}

func (s *Service) normalizeActionLeaseAuditQuery(query ActionLeaseAuditQuery) (ActionLease, error) {
	scope := scopekey.Normalize(s.workspaceID, query.WorkspaceID, query.ProfileID, query.Peer)
	if !scope.Complete() {
		return ActionLease{}, fmt.Errorf("goncho: action lease audit workspace_id and peer are required")
	}
	return ActionLease{WorkspaceID: scope.WorkspaceID, ProfileID: scope.ProfileID, Peer: scope.Peer, ActionID: strings.TrimSpace(query.ActionID)}, nil
}

func ensureActionExistsTx(ctx context.Context, tx *sql.Tx, lease ActionLease) error {
	var exists int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM goncho_actions WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ?`, lease.WorkspaceID, lease.ProfileID, lease.Peer, lease.ActionID).Scan(&exists)
	if err == sql.ErrNoRows {
		return sql.ErrNoRows
	}
	if err != nil {
		return fmt.Errorf("goncho: check action for lease: %w", err)
	}
	return nil
}

func getActionLeaseTx(ctx context.Context, tx *sql.Tx, workspaceID, profileID, peer, actionID string) (ActionLease, bool, error) {
	var lease ActionLease
	err := tx.QueryRowContext(ctx, `SELECT workspace_id, profile_id, peer_id, action_id, owner, acquired_at, renewed_at, expires_at FROM goncho_action_leases WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ?`, workspaceID, profileID, peer, actionID).Scan(&lease.WorkspaceID, &lease.ProfileID, &lease.Peer, &lease.ActionID, &lease.Owner, &lease.AcquiredAt, &lease.RenewedAt, &lease.ExpiresAt)
	if err == sql.ErrNoRows {
		return ActionLease{}, false, nil
	}
	if err != nil {
		return ActionLease{}, false, fmt.Errorf("goncho: get action lease: %w", err)
	}
	return lease, true, nil
}

func upsertActionLeaseTx(ctx context.Context, tx *sql.Tx, lease ActionLease) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO goncho_action_leases(workspace_id, profile_id, peer_id, action_id, owner, acquired_at, renewed_at, expires_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, profile_id, peer_id, action_id)
		DO UPDATE SET owner = excluded.owner, acquired_at = excluded.acquired_at, renewed_at = excluded.renewed_at, expires_at = excluded.expires_at
	`, lease.WorkspaceID, lease.ProfileID, lease.Peer, lease.ActionID, lease.Owner, lease.AcquiredAt, lease.RenewedAt, lease.ExpiresAt)
	if err != nil {
		return fmt.Errorf("goncho: upsert action lease: %w", err)
	}
	return nil
}

func insertActionLeaseAuditTx(ctx context.Context, tx *sql.Tx, lease ActionLease, actor string, decision ActionLeaseDecision, reason string, createdAt int64) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO goncho_action_lease_audit(workspace_id, profile_id, peer_id, action_id, actor, decision, reason, expires_at, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, lease.WorkspaceID, lease.ProfileID, lease.Peer, lease.ActionID, actor, string(decision), reason, lease.ExpiresAt, createdAt)
	if err != nil {
		return 0, fmt.Errorf("goncho: insert action lease audit: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("goncho: action lease audit id: %w", err)
	}
	return id, nil
}

func listExpiredActionLeasesTx(ctx context.Context, tx *sql.Tx, query ActionLease, now int64) ([]ActionLease, error) {
	args := []any{query.WorkspaceID, query.ProfileID, query.Peer, now}
	where := `workspace_id = ? AND profile_id = ? AND peer_id = ? AND expires_at <= ?`
	if query.ActionID != "" {
		where += ` AND action_id = ?`
		args = append(args, query.ActionID)
	}
	rows, err := tx.QueryContext(ctx, `SELECT workspace_id, profile_id, peer_id, action_id, owner, acquired_at, renewed_at, expires_at FROM goncho_action_leases WHERE `+where+` ORDER BY expires_at ASC, action_id ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("goncho: list expired action leases: %w", err)
	}
	defer rows.Close()
	out := []ActionLease{}
	for rows.Next() {
		var lease ActionLease
		if err := rows.Scan(&lease.WorkspaceID, &lease.ProfileID, &lease.Peer, &lease.ActionID, &lease.Owner, &lease.AcquiredAt, &lease.RenewedAt, &lease.ExpiresAt); err != nil {
			return nil, fmt.Errorf("goncho: scan expired action lease: %w", err)
		}
		out = append(out, lease)
	}
	return out, rows.Err()
}
