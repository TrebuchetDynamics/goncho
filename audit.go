package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type AuditAction string

const (
	AuditActionObserve AuditAction = "observe"
)

type AuditTargetType string

const (
	AuditTargetObservation AuditTargetType = "observation"
)

type AuditEvent struct {
	ID          string            `json:"id"`
	Action      AuditAction       `json:"action"`
	TargetType  AuditTargetType   `json:"target_type"`
	TargetID    string            `json:"target_id"`
	WorkspaceID string            `json:"workspace_id"`
	PeerID      string            `json:"peer_id"`
	SessionKey  string            `json:"session_key"`
	Reason      string            `json:"reason"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

type AuditQuery struct {
	Action      AuditAction     `json:"action,omitempty"`
	TargetType  AuditTargetType `json:"target_type,omitempty"`
	TargetID    string          `json:"target_id,omitempty"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	PeerID      string          `json:"peer_id,omitempty"`
	SessionKey  string          `json:"session_key,omitempty"`
	Since       time.Time       `json:"since,omitempty"`
	Until       time.Time       `json:"until,omitempty"`
	Limit       int             `json:"limit,omitempty"`
}

type AuditResult struct {
	Events []AuditEvent `json:"events"`
	Count  int          `json:"count"`
}

func AuditTrail(ctx context.Context, db *sql.DB, q AuditQuery) (AuditResult, error) {
	if err := ctx.Err(); err != nil {
		return AuditResult{}, err
	}
	if db == nil {
		return AuditResult{}, fmt.Errorf("%w: nil db", ErrObservationInvalid)
	}
	if q.Action != "" && q.Action != AuditActionObserve {
		return AuditResult{}, fmt.Errorf("%w: unsupported audit action %q", ErrObservationInvalid, q.Action)
	}
	if q.TargetType != "" && q.TargetType != AuditTargetObservation {
		return AuditResult{}, fmt.Errorf("%w: unsupported audit target_type %q", ErrObservationInvalid, q.TargetType)
	}
	limit := normalizeObservationLimit(q.Limit)
	args := []any{}
	var where []string
	appendExactFilter := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		where = append(where, column+" = ?")
		args = append(args, value)
	}
	appendExactFilter("action", string(q.Action))
	appendExactFilter("target_type", string(q.TargetType))
	appendExactFilter("target_id", q.TargetID)
	appendExactFilter("workspace_id", q.WorkspaceID)
	appendExactFilter("peer_id", q.PeerID)
	appendExactFilter("session_key", q.SessionKey)
	if !q.Since.IsZero() {
		where = append(where, "created_at >= ?")
		args = append(args, q.Since.UTC().UnixNano())
	}
	if !q.Until.IsZero() {
		where = append(where, "created_at <= ?")
		args = append(args, q.Until.UTC().UnixNano())
	}
	query := `SELECT id, action, target_type, target_id, workspace_id, peer_id, session_key, reason, metadata_json, created_at FROM goncho_audit_events`
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, " AND ")
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return AuditResult{}, wrapObservationSQLError("audit trail", err)
	}
	defer rows.Close()
	out := AuditResult{Events: []AuditEvent{}}
	for rows.Next() {
		event, err := scanAuditEvent(rows)
		if err != nil {
			return AuditResult{}, err
		}
		out.Events = append(out.Events, event)
	}
	if err := rows.Err(); err != nil {
		return AuditResult{}, wrapObservationSQLError("iterate audit trail", err)
	}
	out.Count = len(out.Events)
	return out, nil
}

func (s *Service) AuditTrail(ctx context.Context, q AuditQuery) (AuditResult, error) {
	if s == nil {
		return AuditResult{}, fmt.Errorf("%w: nil service", ErrObservationInvalid)
	}
	q.WorkspaceID = serviceObservationWorkspace(s.workspaceID, q.WorkspaceID)
	return AuditTrail(ctx, s.db, q)
}

func insertAuditEvent(ctx context.Context, tx *sql.Tx, event AuditEvent, metadataJSON string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO goncho_audit_events(
			id, action, target_type, target_id, workspace_id, peer_id, session_key, reason, metadata_json, created_at
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		event.ID,
		string(event.Action),
		string(event.TargetType),
		event.TargetID,
		event.WorkspaceID,
		event.PeerID,
		event.SessionKey,
		event.Reason,
		metadataJSON,
		event.CreatedAt.UTC().UnixNano(),
	)
	if err != nil {
		return wrapObservationSQLError("insert audit event", err)
	}
	return nil
}

func firstObserveAuditID(ctx context.Context, tx *sql.Tx, observationID string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM goncho_audit_events
		WHERE action = ? AND target_type = ? AND target_id = ?
		ORDER BY created_at ASC, id ASC
		LIMIT 1
	`, string(AuditActionObserve), string(AuditTargetObservation), observationID).Scan(&id)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("%w: observe audit missing for %s", ErrObservationNotFound, observationID)
	}
	if err != nil {
		return "", wrapObservationSQLError("lookup observe audit", err)
	}
	return id, nil
}

type auditScanner interface {
	Scan(...any) error
}

func scanAuditEvent(scanner auditScanner) (AuditEvent, error) {
	var event AuditEvent
	var action, targetType, metadataJSON string
	var createdAt int64
	err := scanner.Scan(
		&event.ID,
		&action,
		&targetType,
		&event.TargetID,
		&event.WorkspaceID,
		&event.PeerID,
		&event.SessionKey,
		&event.Reason,
		&metadataJSON,
		&createdAt,
	)
	if err != nil {
		return AuditEvent{}, wrapObservationSQLError("scan audit event", err)
	}
	metadata, err := decodeObservationMetadata(metadataJSON)
	if err != nil {
		return AuditEvent{}, fmt.Errorf("goncho: decode audit metadata for %s: %w", event.ID, err)
	}
	event.Action = AuditAction(action)
	event.TargetType = AuditTargetType(targetType)
	event.Metadata = metadata
	event.CreatedAt = time.Unix(0, createdAt).UTC()
	return event, nil
}
