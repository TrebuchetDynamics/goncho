package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

type ActionStatus string

const (
	ActionStatusPending ActionStatus = "pending"
	ActionStatusDone    ActionStatus = "done"
)

var actionGraphDDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_actions (
		workspace_id TEXT NOT NULL,
		profile_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL,
		action_id TEXT NOT NULL,
		title TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		PRIMARY KEY(workspace_id, profile_id, peer_id, action_id)
	)`,
	`CREATE TABLE IF NOT EXISTS goncho_action_dependencies (
		workspace_id TEXT NOT NULL,
		profile_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL,
		action_id TEXT NOT NULL,
		depends_on TEXT NOT NULL,
		PRIMARY KEY(workspace_id, profile_id, peer_id, action_id, depends_on)
	)`,
	`CREATE TABLE IF NOT EXISTS goncho_action_signals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id TEXT NOT NULL,
		profile_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL,
		action_id TEXT NOT NULL,
		signal TEXT NOT NULL,
		message TEXT NOT NULL DEFAULT '',
		actor TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_actions_scope ON goncho_actions(workspace_id, profile_id, peer_id, status, updated_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_action_signals_scope ON goncho_action_signals(workspace_id, profile_id, peer_id, action_id, created_at DESC)`,
}

type ActionParams struct {
	WorkspaceID string   `json:"workspace_id,omitempty"`
	ProfileID   string   `json:"profile_id,omitempty"`
	Peer        string   `json:"peer"`
	ActionID    string   `json:"action_id"`
	Title       string   `json:"title"`
	DependsOn   []string `json:"depends_on,omitempty"`
}

type ActionGraphQuery struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	ActionID    string `json:"action_id,omitempty"`
}

type ActionSignalParams struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	ActionID    string `json:"action_id"`
	Signal      string `json:"signal"`
	Message     string `json:"message,omitempty"`
	Actor       string `json:"actor,omitempty"`
}

type ActionSignal struct {
	ID        int64                 `json:"id"`
	ActionID  string                `json:"action_id"`
	Signal    string                `json:"signal"`
	Message   string                `json:"message,omitempty"`
	Actor     string                `json:"actor,omitempty"`
	CreatedAt int64                 `json:"created_at"`
	Receipts  []ActionSignalReceipt `json:"receipts,omitempty"`
}

type ActionNode struct {
	WorkspaceID string         `json:"workspace_id"`
	ProfileID   string         `json:"profile_id,omitempty"`
	Peer        string         `json:"peer"`
	ActionID    string         `json:"action_id"`
	Title       string         `json:"title"`
	Status      ActionStatus   `json:"status"`
	DependsOn   []string       `json:"depends_on,omitempty"`
	Signals     []ActionSignal `json:"signals,omitempty"`
	CreatedAt   int64          `json:"created_at"`
	UpdatedAt   int64          `json:"updated_at"`
}

type ActionGraph struct {
	WorkspaceID string       `json:"workspace_id"`
	ProfileID   string       `json:"profile_id,omitempty"`
	Peer        string       `json:"peer"`
	Nodes       []ActionNode `json:"nodes"`
	Frontier    []ActionNode `json:"frontier"`
	NextAction  *ActionNode  `json:"next_action,omitempty"`
}

func (s *Service) UpsertAction(ctx context.Context, params ActionParams) (ActionNode, error) {
	norm, err := s.normalizeActionParams(params)
	if err != nil {
		return ActionNode{}, err
	}
	now := time.Now().Unix()
	existing, found, err := getActionNode(ctx, s.db, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.ActionID)
	if err != nil {
		return ActionNode{}, err
	}
	norm.Status = ActionStatusPending
	norm.CreatedAt = now
	if found {
		norm.Status = existing.Status
		norm.CreatedAt = existing.CreatedAt
	}
	norm.UpdatedAt = now
	if err := upsertActionNode(ctx, s.db, norm); err != nil {
		return ActionNode{}, err
	}
	if err := replaceActionDependencies(ctx, s.db, norm); err != nil {
		return ActionNode{}, err
	}
	return s.actionNodeWithEdges(ctx, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.ActionID)
}

func (s *Service) CompleteAction(ctx context.Context, query ActionGraphQuery) (ActionNode, error) {
	norm, err := s.normalizeActionQuery(query)
	if err != nil {
		return ActionNode{}, err
	}
	if norm.ActionID == "" {
		return ActionNode{}, fmt.Errorf("goncho: action_id is required")
	}
	res, err := s.db.ExecContext(ctx, `UPDATE goncho_actions SET status = ?, updated_at = ? WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ?`, string(ActionStatusDone), time.Now().Unix(), norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.ActionID)
	if err != nil {
		return ActionNode{}, fmt.Errorf("goncho: complete action: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ActionNode{}, sql.ErrNoRows
	}
	return s.actionNodeWithEdges(ctx, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.ActionID)
}

func (s *Service) SignalAction(ctx context.Context, params ActionSignalParams) (ActionSignal, error) {
	norm, err := s.normalizeActionSignal(params)
	if err != nil {
		return ActionSignal{}, err
	}
	if _, _, err := getActionNode(ctx, s.db, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.ActionID); err != nil {
		return ActionSignal{}, err
	}
	now := time.Now().Unix()
	signalInput := norm.Signals[0]
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO goncho_action_signals(workspace_id, profile_id, peer_id, action_id, signal, message, actor, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
	`, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.ActionID, signalInput.Signal, signalInput.Message, signalInput.Actor, now)
	if err != nil {
		return ActionSignal{}, fmt.Errorf("goncho: signal action: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ActionSignal{}, fmt.Errorf("goncho: action signal id: %w", err)
	}
	return ActionSignal{ID: id, ActionID: norm.ActionID, Signal: signalInput.Signal, Message: signalInput.Message, Actor: signalInput.Actor, CreatedAt: now}, nil
}

func (s *Service) ReadActionGraph(ctx context.Context, query ActionGraphQuery) (ActionGraph, error) {
	norm, err := s.normalizeActionQuery(query)
	if err != nil {
		return ActionGraph{}, err
	}
	nodes, err := listActionNodes(ctx, s.db, norm.WorkspaceID, norm.ProfileID, norm.Peer)
	if err != nil {
		return ActionGraph{}, err
	}
	for i := range nodes {
		deps, err := listActionDependencies(ctx, s.db, norm.WorkspaceID, norm.ProfileID, norm.Peer, nodes[i].ActionID)
		if err != nil {
			return ActionGraph{}, err
		}
		signals, err := listActionSignals(ctx, s.db, norm.WorkspaceID, norm.ProfileID, norm.Peer, nodes[i].ActionID)
		if err != nil {
			return ActionGraph{}, err
		}
		nodes[i].DependsOn = deps
		nodes[i].Signals = signals
	}
	frontier := actionFrontier(nodes)
	var next *ActionNode
	if len(frontier) > 0 {
		item := frontier[0]
		next = &item
	}
	return ActionGraph{WorkspaceID: norm.WorkspaceID, ProfileID: norm.ProfileID, Peer: norm.Peer, Nodes: nodes, Frontier: frontier, NextAction: next}, nil
}

func (s *Service) normalizeActionParams(params ActionParams) (ActionNode, error) {
	workspaceID := firstNonBlank(params.WorkspaceID, s.workspaceID)
	peer := strings.TrimSpace(params.Peer)
	actionID := strings.TrimSpace(params.ActionID)
	title := strings.TrimSpace(params.Title)
	if workspaceID == "" || peer == "" || actionID == "" || title == "" {
		return ActionNode{}, fmt.Errorf("goncho: action workspace_id, peer, action_id, and title are required")
	}
	return ActionNode{WorkspaceID: workspaceID, ProfileID: strings.TrimSpace(params.ProfileID), Peer: peer, ActionID: actionID, Title: title, DependsOn: normalizeActionIDs(params.DependsOn)}, nil
}

func (s *Service) normalizeActionQuery(query ActionGraphQuery) (ActionNode, error) {
	workspaceID := firstNonBlank(query.WorkspaceID, s.workspaceID)
	peer := strings.TrimSpace(query.Peer)
	if workspaceID == "" || peer == "" {
		return ActionNode{}, fmt.Errorf("goncho: action workspace_id and peer are required")
	}
	return ActionNode{WorkspaceID: workspaceID, ProfileID: strings.TrimSpace(query.ProfileID), Peer: peer, ActionID: strings.TrimSpace(query.ActionID)}, nil
}

func (s *Service) normalizeActionSignal(params ActionSignalParams) (ActionNode, error) {
	workspaceID := firstNonBlank(params.WorkspaceID, s.workspaceID)
	peer := strings.TrimSpace(params.Peer)
	actionID := strings.TrimSpace(params.ActionID)
	signal := strings.TrimSpace(params.Signal)
	if workspaceID == "" || peer == "" || actionID == "" || signal == "" {
		return ActionNode{}, fmt.Errorf("goncho: action signal workspace_id, peer, action_id, and signal are required")
	}
	return ActionNode{WorkspaceID: workspaceID, ProfileID: strings.TrimSpace(params.ProfileID), Peer: peer, ActionID: actionID, Signals: []ActionSignal{{ActionID: actionID, Signal: signal, Message: strings.TrimSpace(params.Message), Actor: strings.TrimSpace(params.Actor)}}}, nil
}

func upsertActionNode(ctx context.Context, db *sql.DB, node ActionNode) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO goncho_actions(workspace_id, profile_id, peer_id, action_id, title, status, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, profile_id, peer_id, action_id)
		DO UPDATE SET title = excluded.title, status = excluded.status, updated_at = excluded.updated_at
	`, node.WorkspaceID, node.ProfileID, node.Peer, node.ActionID, node.Title, string(node.Status), node.CreatedAt, node.UpdatedAt)
	if err != nil {
		return fmt.Errorf("goncho: upsert action: %w", err)
	}
	return nil
}

func replaceActionDependencies(ctx context.Context, db *sql.DB, node ActionNode) error {
	if _, err := db.ExecContext(ctx, `DELETE FROM goncho_action_dependencies WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ?`, node.WorkspaceID, node.ProfileID, node.Peer, node.ActionID); err != nil {
		return fmt.Errorf("goncho: clear action dependencies: %w", err)
	}
	for _, dep := range node.DependsOn {
		if _, err := db.ExecContext(ctx, `INSERT INTO goncho_action_dependencies(workspace_id, profile_id, peer_id, action_id, depends_on) VALUES(?, ?, ?, ?, ?)`, node.WorkspaceID, node.ProfileID, node.Peer, node.ActionID, dep); err != nil {
			return fmt.Errorf("goncho: insert action dependency: %w", err)
		}
	}
	return nil
}

func getActionNode(ctx context.Context, db *sql.DB, workspaceID, profileID, peer, actionID string) (ActionNode, bool, error) {
	var node ActionNode
	var status string
	err := db.QueryRowContext(ctx, `SELECT workspace_id, profile_id, peer_id, action_id, title, status, created_at, updated_at FROM goncho_actions WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ?`, workspaceID, profileID, peer, actionID).Scan(&node.WorkspaceID, &node.ProfileID, &node.Peer, &node.ActionID, &node.Title, &status, &node.CreatedAt, &node.UpdatedAt)
	if err == sql.ErrNoRows {
		return ActionNode{}, false, nil
	}
	if err != nil {
		return ActionNode{}, false, fmt.Errorf("goncho: get action: %w", err)
	}
	node.Status = ActionStatus(status)
	return node, true, nil
}

func listActionNodes(ctx context.Context, db *sql.DB, workspaceID, profileID, peer string) ([]ActionNode, error) {
	rows, err := db.QueryContext(ctx, `SELECT workspace_id, profile_id, peer_id, action_id, title, status, created_at, updated_at FROM goncho_actions WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? ORDER BY created_at ASC, action_id ASC`, workspaceID, profileID, peer)
	if err != nil {
		return nil, fmt.Errorf("goncho: list actions: %w", err)
	}
	defer rows.Close()
	var out []ActionNode
	for rows.Next() {
		var node ActionNode
		var status string
		if err := rows.Scan(&node.WorkspaceID, &node.ProfileID, &node.Peer, &node.ActionID, &node.Title, &status, &node.CreatedAt, &node.UpdatedAt); err != nil {
			return nil, fmt.Errorf("goncho: scan action: %w", err)
		}
		node.Status = ActionStatus(status)
		out = append(out, node)
	}
	return out, rows.Err()
}

func (s *Service) actionNodeWithEdges(ctx context.Context, workspaceID, profileID, peer, actionID string) (ActionNode, error) {
	node, found, err := getActionNode(ctx, s.db, workspaceID, profileID, peer, actionID)
	if err != nil {
		return ActionNode{}, err
	}
	if !found {
		return ActionNode{}, sql.ErrNoRows
	}
	node.DependsOn, err = listActionDependencies(ctx, s.db, workspaceID, profileID, peer, actionID)
	if err != nil {
		return ActionNode{}, err
	}
	node.Signals, err = listActionSignals(ctx, s.db, workspaceID, profileID, peer, actionID)
	if err != nil {
		return ActionNode{}, err
	}
	return node, nil
}

func listActionDependencies(ctx context.Context, db *sql.DB, workspaceID, profileID, peer, actionID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT depends_on FROM goncho_action_dependencies WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ? ORDER BY depends_on ASC`, workspaceID, profileID, peer, actionID)
	if err != nil {
		return nil, fmt.Errorf("goncho: list action dependencies: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var dep string
		if err := rows.Scan(&dep); err != nil {
			return nil, fmt.Errorf("goncho: scan action dependency: %w", err)
		}
		out = append(out, dep)
	}
	return out, rows.Err()
}

func listActionSignals(ctx context.Context, db *sql.DB, workspaceID, profileID, peer, actionID string) ([]ActionSignal, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, action_id, signal, message, actor, created_at FROM goncho_action_signals WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND action_id = ? ORDER BY created_at ASC, id ASC`, workspaceID, profileID, peer, actionID)
	if err != nil {
		return nil, fmt.Errorf("goncho: list action signals: %w", err)
	}
	out := []ActionSignal{}
	for rows.Next() {
		var signal ActionSignal
		if err := rows.Scan(&signal.ID, &signal.ActionID, &signal.Signal, &signal.Message, &signal.Actor, &signal.CreatedAt); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("goncho: scan action signal: %w", err)
		}
		out = append(out, signal)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("goncho: close action signals: %w", err)
	}
	for i := range out {
		receipts, err := listActionSignalReceiptsBySignal(ctx, db, workspaceID, profileID, peer, actionID, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Receipts = receipts
	}
	return out, nil
}

func actionFrontier(nodes []ActionNode) []ActionNode {
	statusByID := map[string]ActionStatus{}
	for _, node := range nodes {
		statusByID[node.ActionID] = node.Status
	}
	frontier := []ActionNode{}
	for _, node := range nodes {
		if node.Status != ActionStatusPending {
			continue
		}
		ready := true
		for _, dep := range node.DependsOn {
			if statusByID[dep] != ActionStatusDone {
				ready = false
				break
			}
		}
		if ready {
			frontier = append(frontier, node)
		}
	}
	sort.Slice(frontier, func(i, j int) bool {
		if frontier[i].CreatedAt != frontier[j].CreatedAt {
			return frontier[i].CreatedAt < frontier[j].CreatedAt
		}
		return frontier[i].ActionID < frontier[j].ActionID
	})
	return frontier
}

func normalizeActionIDs(values []string) []string {
	return textutil.UniqueTrimmed(values, true)
}
