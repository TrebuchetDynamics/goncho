package goncho

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/idutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/scopeauth"
)

type TeamFeedDecision string

const (
	TeamFeedDecisionAllowed TeamFeedDecision = "allowed"
	TeamFeedDecisionDenied  TeamFeedDecision = "denied"
)

var teamFeedDDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_team_feed_audit (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id TEXT NOT NULL,
		profile_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL,
		actor TEXT NOT NULL,
		actor_workspace_id TEXT NOT NULL DEFAULT '',
		actor_profile_id TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL DEFAULT '',
		decision TEXT NOT NULL,
		reason TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_team_feed_audit_scope ON goncho_team_feed_audit(workspace_id, profile_id, peer_id, created_at DESC)`,
}

type TeamFeedQuery struct {
	WorkspaceID      string `json:"workspace_id,omitempty"`
	ProfileID        string `json:"profile_id,omitempty"`
	Peer             string `json:"peer"`
	Actor            string `json:"actor"`
	ActorWorkspaceID string `json:"actor_workspace_id,omitempty"`
	ActorProfileID   string `json:"actor_profile_id,omitempty"`
	Role             string `json:"role,omitempty"`
	Limit            int    `json:"limit,omitempty"`
	Cursor           string `json:"cursor,omitempty"`
}

type TeamFeedAuditQuery struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	Limit       int    `json:"limit,omitempty"`
}

type TeamFeedEntry struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	WorkspaceID string `json:"workspace_id"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	ActionID    string `json:"action_id"`
	SignalID    int64  `json:"signal_id"`
	Signal      string `json:"signal"`
	Message     string `json:"message,omitempty"`
	Actor       string `json:"actor,omitempty"`
	CreatedAt   int64  `json:"created_at"`
}

type TeamFeedResult struct {
	Authorized bool             `json:"authorized"`
	Decision   TeamFeedDecision `json:"decision"`
	Reason     string           `json:"reason,omitempty"`
	Entries    []TeamFeedEntry  `json:"entries"`
	NextCursor string           `json:"next_cursor,omitempty"`
	AuditID    int64            `json:"audit_id,omitempty"`
}

type TeamFeedAuditEvent struct {
	ID               int64            `json:"id"`
	WorkspaceID      string           `json:"workspace_id"`
	ProfileID        string           `json:"profile_id,omitempty"`
	Peer             string           `json:"peer"`
	Actor            string           `json:"actor"`
	ActorWorkspaceID string           `json:"actor_workspace_id,omitempty"`
	ActorProfileID   string           `json:"actor_profile_id,omitempty"`
	Role             string           `json:"role,omitempty"`
	Decision         TeamFeedDecision `json:"decision"`
	Reason           string           `json:"reason,omitempty"`
	CreatedAt        int64            `json:"created_at"`
}

type TeamFeedAuditResult struct {
	Events []TeamFeedAuditEvent `json:"events"`
	Count  int                  `json:"count"`
}

func (s *Service) TeamFeed(ctx context.Context, query TeamFeedQuery) (TeamFeedResult, error) {
	norm, err := s.normalizeTeamFeedQuery(query)
	if err != nil {
		return TeamFeedResult{}, err
	}
	actorScope := scopeauth.NormalizeActorScope(query.ActorWorkspaceID, query.ActorProfileID, norm.WorkspaceID)
	role := strings.TrimSpace(query.Role)
	allowed := role == "admin" || scopeauth.SameScope(actorScope, norm.WorkspaceID, norm.ProfileID)
	if !allowed {
		reason := scopeauth.DeniedReadReason(actorScope, "team feed", norm.WorkspaceID, norm.ProfileID)
		auditID, err := s.insertTeamFeedAudit(ctx, norm, actorScope.WorkspaceID, actorScope.ProfileID, role, TeamFeedDecisionDenied, reason)
		if err != nil {
			return TeamFeedResult{}, err
		}
		return TeamFeedResult{Decision: TeamFeedDecisionDenied, Reason: reason, Entries: []TeamFeedEntry{}, AuditID: auditID}, nil
	}
	limit := limitutil.DefaultClamped(query.Limit, 50, 100)
	cursor := int64(0)
	if strings.TrimSpace(query.Cursor) != "" {
		cursor, err = idutil.ParseDecimal(query.Cursor)
		if err != nil || cursor < 0 {
			return TeamFeedResult{}, fmt.Errorf("goncho: invalid team feed cursor %q", query.Cursor)
		}
	}
	entries, next, err := s.listTeamFeedEntries(ctx, norm, cursor, limit)
	if err != nil {
		return TeamFeedResult{}, err
	}
	auditID, err := s.insertTeamFeedAudit(ctx, norm, actorScope.WorkspaceID, actorScope.ProfileID, role, TeamFeedDecisionAllowed, "team feed read")
	if err != nil {
		return TeamFeedResult{}, err
	}
	return TeamFeedResult{Authorized: true, Decision: TeamFeedDecisionAllowed, Entries: entries, NextCursor: next, AuditID: auditID}, nil
}

func (s *Service) ListTeamFeedAudit(ctx context.Context, query TeamFeedAuditQuery) (TeamFeedAuditResult, error) {
	norm, err := s.normalizeTeamFeedAuditQuery(query)
	if err != nil {
		return TeamFeedAuditResult{}, err
	}
	limit := limitutil.DefaultClamped(query.Limit, 100, 100)
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, profile_id, peer_id, actor, actor_workspace_id, actor_profile_id, role, decision, reason, created_at FROM goncho_team_feed_audit WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? ORDER BY created_at DESC, id DESC LIMIT ?`, norm.WorkspaceID, norm.ProfileID, norm.Peer, limit)
	if err != nil {
		return TeamFeedAuditResult{}, fmt.Errorf("goncho: list team feed audit: %w", err)
	}
	defer rows.Close()
	out := []TeamFeedAuditEvent{}
	for rows.Next() {
		var event TeamFeedAuditEvent
		var decision string
		if err := rows.Scan(&event.ID, &event.WorkspaceID, &event.ProfileID, &event.Peer, &event.Actor, &event.ActorWorkspaceID, &event.ActorProfileID, &event.Role, &decision, &event.Reason, &event.CreatedAt); err != nil {
			return TeamFeedAuditResult{}, fmt.Errorf("goncho: scan team feed audit: %w", err)
		}
		event.Decision = TeamFeedDecision(decision)
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return TeamFeedAuditResult{}, fmt.Errorf("goncho: iterate team feed audit: %w", err)
	}
	return TeamFeedAuditResult{Events: out, Count: len(out)}, nil
}

func (s *Service) normalizeTeamFeedQuery(query TeamFeedQuery) (TeamFeedEntry, error) {
	workspaceID := firstNonBlank(query.WorkspaceID, s.workspaceID)
	peer := strings.TrimSpace(query.Peer)
	actor := strings.TrimSpace(query.Actor)
	if workspaceID == "" || peer == "" || actor == "" {
		return TeamFeedEntry{}, fmt.Errorf("goncho: team feed workspace_id, peer, and actor are required")
	}
	return TeamFeedEntry{WorkspaceID: workspaceID, ProfileID: strings.TrimSpace(query.ProfileID), Peer: peer, Actor: actor}, nil
}

func (s *Service) normalizeTeamFeedAuditQuery(query TeamFeedAuditQuery) (TeamFeedEntry, error) {
	workspaceID := firstNonBlank(query.WorkspaceID, s.workspaceID)
	peer := strings.TrimSpace(query.Peer)
	if workspaceID == "" || peer == "" {
		return TeamFeedEntry{}, fmt.Errorf("goncho: team feed audit workspace_id and peer are required")
	}
	return TeamFeedEntry{WorkspaceID: workspaceID, ProfileID: strings.TrimSpace(query.ProfileID), Peer: peer}, nil
}

func (s *Service) listTeamFeedEntries(ctx context.Context, scope TeamFeedEntry, cursor int64, limit int) ([]TeamFeedEntry, string, error) {
	args := []any{scope.WorkspaceID, scope.ProfileID, scope.Peer}
	where := `workspace_id = ? AND profile_id = ? AND peer_id = ?`
	if cursor > 0 {
		where += ` AND id < ?`
		args = append(args, cursor)
	}
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, profile_id, peer_id, action_id, signal, message, actor, created_at FROM goncho_action_signals WHERE `+where+` ORDER BY id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, "", fmt.Errorf("goncho: list team feed: %w", err)
	}
	defer rows.Close()
	entries := []TeamFeedEntry{}
	for rows.Next() {
		var entry TeamFeedEntry
		if err := rows.Scan(&entry.SignalID, &entry.WorkspaceID, &entry.ProfileID, &entry.Peer, &entry.ActionID, &entry.Signal, &entry.Message, &entry.Actor, &entry.CreatedAt); err != nil {
			return nil, "", fmt.Errorf("goncho: scan team feed: %w", err)
		}
		entry.ID = idutil.Prefixed("signal:", entry.SignalID)
		entry.Kind = "action_signal"
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("goncho: iterate team feed: %w", err)
	}
	next := ""
	if len(entries) > limit {
		next = idutil.Decimal(entries[limit-1].SignalID)
		entries = entries[:limit]
	}
	return entries, next, nil
}

func (s *Service) insertTeamFeedAudit(ctx context.Context, scope TeamFeedEntry, actorWorkspaceID, actorProfileID, role string, decision TeamFeedDecision, reason string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO goncho_team_feed_audit(workspace_id, profile_id, peer_id, actor, actor_workspace_id, actor_profile_id, role, decision, reason, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, scope.WorkspaceID, scope.ProfileID, scope.Peer, scope.Actor, actorWorkspaceID, actorProfileID, role, string(decision), reason, time.Now().UnixNano())
	if err != nil {
		return 0, fmt.Errorf("goncho: insert team feed audit: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("goncho: team feed audit id: %w", err)
	}
	return id, nil
}
