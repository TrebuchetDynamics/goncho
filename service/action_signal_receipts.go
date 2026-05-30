package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/scopeauth"
)

type ActionSignalReceiptDecision string

const (
	ActionSignalReceiptDecisionAllowed ActionSignalReceiptDecision = "allowed"
	ActionSignalReceiptDecisionDenied  ActionSignalReceiptDecision = "denied"
)

var actionSignalReceiptDDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_action_signal_receipts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id TEXT NOT NULL,
		profile_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL,
		action_id TEXT NOT NULL,
		signal_id INTEGER NOT NULL,
		actor TEXT NOT NULL,
		read_at INTEGER NOT NULL,
		UNIQUE(workspace_id, profile_id, peer_id, action_id, signal_id, actor)
	)`,
	`CREATE TABLE IF NOT EXISTS goncho_action_signal_receipt_audit (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id TEXT NOT NULL,
		profile_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL,
		action_id TEXT NOT NULL,
		signal_id INTEGER NOT NULL,
		actor TEXT NOT NULL,
		actor_workspace_id TEXT NOT NULL DEFAULT '',
		actor_profile_id TEXT NOT NULL DEFAULT '',
		decision TEXT NOT NULL,
		reason TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_action_signal_receipts_signal ON goncho_action_signal_receipts(workspace_id, profile_id, peer_id, action_id, signal_id, read_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_action_signal_receipt_audit_scope ON goncho_action_signal_receipt_audit(workspace_id, profile_id, peer_id, action_id, created_at DESC)`,
}

type ActionSignalReceiptParams struct {
	WorkspaceID      string `json:"workspace_id,omitempty"`
	ProfileID        string `json:"profile_id,omitempty"`
	Peer             string `json:"peer"`
	ActionID         string `json:"action_id"`
	SignalID         int64  `json:"signal_id"`
	Actor            string `json:"actor"`
	ActorWorkspaceID string `json:"actor_workspace_id,omitempty"`
	ActorProfileID   string `json:"actor_profile_id,omitempty"`
}

type ActionSignalReceiptQuery struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	ActionID    string `json:"action_id"`
	SignalID    int64  `json:"signal_id,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type ActionSignalReceiptAuditQuery struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	ActionID    string `json:"action_id,omitempty"`
	SignalID    int64  `json:"signal_id,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type ActionSignalReceipt struct {
	ID          int64  `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	ActionID    string `json:"action_id"`
	SignalID    int64  `json:"signal_id"`
	Actor       string `json:"actor"`
	ReadAt      int64  `json:"read_at"`
}

type ActionSignalReceiptResult struct {
	Authorized bool                        `json:"authorized"`
	Decision   ActionSignalReceiptDecision `json:"decision"`
	Receipt    ActionSignalReceipt         `json:"receipt,omitempty"`
	Reason     string                      `json:"reason,omitempty"`
	AuditID    int64                       `json:"audit_id,omitempty"`
}

type ActionSignalReceiptList struct {
	Receipts []ActionSignalReceipt `json:"receipts"`
	Count    int                   `json:"count"`
}

type ActionSignalReceiptAuditEvent struct {
	ID               int64                       `json:"id"`
	WorkspaceID      string                      `json:"workspace_id"`
	ProfileID        string                      `json:"profile_id,omitempty"`
	Peer             string                      `json:"peer"`
	ActionID         string                      `json:"action_id"`
	SignalID         int64                       `json:"signal_id"`
	Actor            string                      `json:"actor"`
	ActorWorkspaceID string                      `json:"actor_workspace_id,omitempty"`
	ActorProfileID   string                      `json:"actor_profile_id,omitempty"`
	Decision         ActionSignalReceiptDecision `json:"decision"`
	Reason           string                      `json:"reason,omitempty"`
	CreatedAt        int64                       `json:"created_at"`
}

type ActionSignalReceiptAuditResult struct {
	Events []ActionSignalReceiptAuditEvent `json:"events"`
	Count  int                             `json:"count"`
}

func (s *Service) RecordActionSignalReceipt(ctx context.Context, params ActionSignalReceiptParams) (ActionSignalReceiptResult, error) {
	norm, err := s.normalizeActionSignalReceiptParams(params)
	if err != nil {
		return ActionSignalReceiptResult{}, err
	}
	actorScope := scopeauth.NormalizeActorScope(params.ActorWorkspaceID, params.ActorProfileID, norm.WorkspaceID)
	if !scopeauth.SameScope(actorScope, norm.WorkspaceID, norm.ProfileID) {
		reason := scopeauth.DeniedReadReason(actorScope, "signal", norm.WorkspaceID, norm.ProfileID)
		auditID, err := s.insertActionSignalReceiptAudit(ctx, norm, actorScope.WorkspaceID, actorScope.ProfileID, ActionSignalReceiptDecisionDenied, reason)
		if err != nil {
			return ActionSignalReceiptResult{}, err
		}
		return ActionSignalReceiptResult{Decision: ActionSignalReceiptDecisionDenied, Reason: reason, AuditID: auditID}, nil
	}
	if err := ensureActionSignalExists(ctx, s.db, norm); err != nil {
		return ActionSignalReceiptResult{}, err
	}
	now := time.Now().UnixNano()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO goncho_action_signal_receipts(workspace_id, profile_id, peer_id, action_id, signal_id, actor, read_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, profile_id, peer_id, action_id, signal_id, actor)
		DO UPDATE SET read_at = excluded.read_at
	`, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.ActionID, norm.SignalID, norm.Actor, now)
	if err != nil {
		return ActionSignalReceiptResult{}, fmt.Errorf("goncho: record action signal receipt: %w", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		id = lookupActionSignalReceiptID(ctx, s.db, norm)
	}
	receipt := ActionSignalReceipt{ID: id, WorkspaceID: norm.WorkspaceID, ProfileID: norm.ProfileID, Peer: norm.Peer, ActionID: norm.ActionID, SignalID: norm.SignalID, Actor: norm.Actor, ReadAt: now}
	auditID, err := s.insertActionSignalReceiptAudit(ctx, norm, actorScope.WorkspaceID, actorScope.ProfileID, ActionSignalReceiptDecisionAllowed, "read receipt recorded")
	if err != nil {
		return ActionSignalReceiptResult{}, err
	}
	return ActionSignalReceiptResult{Authorized: true, Decision: ActionSignalReceiptDecisionAllowed, Receipt: receipt, AuditID: auditID}, nil
}

func (s *Service) ListActionSignalReceipts(ctx context.Context, query ActionSignalReceiptQuery) (ActionSignalReceiptList, error) {
	norm, err := s.normalizeActionSignalReceiptQuery(query)
	if err != nil {
		return ActionSignalReceiptList{}, err
	}
	limit := limitutil.DefaultClamped(query.Limit, 100, 100)
	receipts, err := listActionSignalReceiptsFiltered(ctx, s.db, norm, limit)
	if err != nil {
		return ActionSignalReceiptList{}, err
	}
	return ActionSignalReceiptList{Receipts: receipts, Count: len(receipts)}, nil
}

func (s *Service) ListActionSignalReceiptAudit(ctx context.Context, query ActionSignalReceiptAuditQuery) (ActionSignalReceiptAuditResult, error) {
	norm, err := s.normalizeActionSignalReceiptAuditQuery(query)
	if err != nil {
		return ActionSignalReceiptAuditResult{}, err
	}
	limit := limitutil.DefaultClamped(query.Limit, 100, 100)
	args := []any{norm.WorkspaceID, norm.ProfileID, norm.Peer}
	where := `workspace_id = ? AND profile_id = ? AND peer_id = ?`
	if norm.ActionID != "" {
		where += ` AND action_id = ?`
		args = append(args, norm.ActionID)
	}
	if norm.SignalID != 0 {
		where += ` AND signal_id = ?`
		args = append(args, norm.SignalID)
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, profile_id, peer_id, action_id, signal_id, actor, actor_workspace_id, actor_profile_id, decision, reason, created_at FROM goncho_action_signal_receipt_audit WHERE `+where+` ORDER BY created_at DESC, id DESC LIMIT ?`, args...)
	if err != nil {
		return ActionSignalReceiptAuditResult{}, fmt.Errorf("goncho: list action signal receipt audit: %w", err)
	}
	defer rows.Close()
	out := []ActionSignalReceiptAuditEvent{}
	for rows.Next() {
		var event ActionSignalReceiptAuditEvent
		var decision string
		if err := rows.Scan(&event.ID, &event.WorkspaceID, &event.ProfileID, &event.Peer, &event.ActionID, &event.SignalID, &event.Actor, &event.ActorWorkspaceID, &event.ActorProfileID, &decision, &event.Reason, &event.CreatedAt); err != nil {
			return ActionSignalReceiptAuditResult{}, fmt.Errorf("goncho: scan action signal receipt audit: %w", err)
		}
		event.Decision = ActionSignalReceiptDecision(decision)
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return ActionSignalReceiptAuditResult{}, fmt.Errorf("goncho: iterate action signal receipt audit: %w", err)
	}
	return ActionSignalReceiptAuditResult{Events: out, Count: len(out)}, nil
}

func (s *Service) normalizeActionSignalReceiptParams(params ActionSignalReceiptParams) (ActionSignalReceipt, error) {
	workspaceID := firstNonBlank(params.WorkspaceID, s.workspaceID)
	peer := strings.TrimSpace(params.Peer)
	actionID := strings.TrimSpace(params.ActionID)
	actor := strings.TrimSpace(params.Actor)
	if workspaceID == "" || peer == "" || actionID == "" || params.SignalID == 0 || actor == "" {
		return ActionSignalReceipt{}, fmt.Errorf("goncho: action signal receipt workspace_id, peer, action_id, signal_id, and actor are required")
	}
	return ActionSignalReceipt{WorkspaceID: workspaceID, ProfileID: strings.TrimSpace(params.ProfileID), Peer: peer, ActionID: actionID, SignalID: params.SignalID, Actor: actor}, nil
}

func (s *Service) normalizeActionSignalReceiptQuery(query ActionSignalReceiptQuery) (ActionSignalReceipt, error) {
	workspaceID := firstNonBlank(query.WorkspaceID, s.workspaceID)
	peer := strings.TrimSpace(query.Peer)
	actionID := strings.TrimSpace(query.ActionID)
	if workspaceID == "" || peer == "" || actionID == "" {
		return ActionSignalReceipt{}, fmt.Errorf("goncho: action signal receipt query workspace_id, peer, and action_id are required")
	}
	return ActionSignalReceipt{WorkspaceID: workspaceID, ProfileID: strings.TrimSpace(query.ProfileID), Peer: peer, ActionID: actionID, SignalID: query.SignalID}, nil
}

func (s *Service) normalizeActionSignalReceiptAuditQuery(query ActionSignalReceiptAuditQuery) (ActionSignalReceipt, error) {
	workspaceID := firstNonBlank(query.WorkspaceID, s.workspaceID)
	peer := strings.TrimSpace(query.Peer)
	if workspaceID == "" || peer == "" {
		return ActionSignalReceipt{}, fmt.Errorf("goncho: action signal receipt audit workspace_id and peer are required")
	}
	return ActionSignalReceipt{WorkspaceID: workspaceID, ProfileID: strings.TrimSpace(query.ProfileID), Peer: peer, ActionID: strings.TrimSpace(query.ActionID), SignalID: query.SignalID}, nil
}

func (s *Service) insertActionSignalReceiptAudit(ctx context.Context, receipt ActionSignalReceipt, actorWorkspaceID, actorProfileID string, decision ActionSignalReceiptDecision, reason string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO goncho_action_signal_receipt_audit(workspace_id, profile_id, peer_id, action_id, signal_id, actor, actor_workspace_id, actor_profile_id, decision, reason, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, receipt.WorkspaceID, receipt.ProfileID, receipt.Peer, receipt.ActionID, receipt.SignalID, receipt.Actor, actorWorkspaceID, actorProfileID, string(decision), reason, time.Now().UnixNano())
	if err != nil {
		return 0, fmt.Errorf("goncho: insert action signal receipt audit: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("goncho: action signal receipt audit id: %w", err)
	}
	return id, nil
}

func ensureActionSignalExists(ctx context.Context, db *sql.DB, receipt ActionSignalReceipt) error {
	var exists int
	err := db.QueryRowContext(ctx, `SELECT 1 FROM goncho_action_signals WHERE id = ? AND workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ?`, receipt.SignalID, receipt.WorkspaceID, receipt.ProfileID, receipt.Peer, receipt.ActionID).Scan(&exists)
	if err == sql.ErrNoRows {
		return sql.ErrNoRows
	}
	if err != nil {
		return fmt.Errorf("goncho: check action signal receipt target: %w", err)
	}
	return nil
}

func lookupActionSignalReceiptID(ctx context.Context, db *sql.DB, receipt ActionSignalReceipt) int64 {
	var id int64
	_ = db.QueryRowContext(ctx, `SELECT id FROM goncho_action_signal_receipts WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ? AND signal_id = ? AND actor = ?`, receipt.WorkspaceID, receipt.ProfileID, receipt.Peer, receipt.ActionID, receipt.SignalID, receipt.Actor).Scan(&id)
	return id
}

func listActionSignalReceiptsBySignal(ctx context.Context, db *sql.DB, workspaceID, profileID, peer, actionID string, signalID int64) ([]ActionSignalReceipt, error) {
	return listActionSignalReceiptsFiltered(ctx, db, ActionSignalReceipt{WorkspaceID: workspaceID, ProfileID: profileID, Peer: peer, ActionID: actionID, SignalID: signalID}, 100)
}

func listActionSignalReceiptsFiltered(ctx context.Context, db *sql.DB, query ActionSignalReceipt, limit int) ([]ActionSignalReceipt, error) {
	args := []any{query.WorkspaceID, query.ProfileID, query.Peer, query.ActionID}
	where := `workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ?`
	if query.SignalID != 0 {
		where += ` AND signal_id = ?`
		args = append(args, query.SignalID)
	}
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, `SELECT id, workspace_id, profile_id, peer_id, action_id, signal_id, actor, read_at FROM goncho_action_signal_receipts WHERE `+where+` ORDER BY read_at ASC, id ASC LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("goncho: list action signal receipts: %w", err)
	}
	defer rows.Close()
	out := []ActionSignalReceipt{}
	for rows.Next() {
		var receipt ActionSignalReceipt
		if err := rows.Scan(&receipt.ID, &receipt.WorkspaceID, &receipt.ProfileID, &receipt.Peer, &receipt.ActionID, &receipt.SignalID, &receipt.Actor, &receipt.ReadAt); err != nil {
			return nil, fmt.Errorf("goncho: scan action signal receipt: %w", err)
		}
		out = append(out, receipt)
	}
	return out, rows.Err()
}
