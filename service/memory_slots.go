package goncho

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/dbscan"
	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
)

var ErrMemorySlotNotFound = errors.New("goncho: memory slot not found")
var ErrMemorySlotConflict = errors.New("goncho: memory slot conflict")

var memorySlotDDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_memory_slots (
		workspace_id TEXT NOT NULL,
		profile_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL,
		scope TEXT NOT NULL,
		name TEXT NOT NULL,
		kind TEXT NOT NULL DEFAULT 'fact',
		value TEXT NOT NULL,
		revision INTEGER NOT NULL DEFAULT 1,
		deleted INTEGER NOT NULL DEFAULT 0 CHECK(deleted IN (0, 1)),
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		PRIMARY KEY(workspace_id, profile_id, peer_id, scope, name)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_memory_slots_scope ON goncho_memory_slots(workspace_id, profile_id, peer_id, scope, deleted, updated_at DESC)`,
}

type MemorySlotParams struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	Scope       string `json:"scope,omitempty"`
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	Value       string `json:"value"`
}

type MemorySlotQuery struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	Scope       string `json:"scope,omitempty"`
	Name        string `json:"name,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type MemorySlot struct {
	WorkspaceID string `json:"workspace_id"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	Scope       string `json:"scope"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Value       string `json:"value"`
	Revision    int    `json:"revision"`
	Deleted     bool   `json:"deleted,omitempty"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type MemorySlotList struct {
	WorkspaceID string       `json:"workspace_id"`
	ProfileID   string       `json:"profile_id,omitempty"`
	Peer        string       `json:"peer"`
	Scope       string       `json:"scope"`
	Slots       []MemorySlot `json:"slots"`
}

func (s *Service) CreateMemorySlot(ctx context.Context, params MemorySlotParams) (MemorySlot, error) {
	norm, err := s.normalizeMemorySlotParams(params)
	if err != nil {
		return MemorySlot{}, err
	}
	existing, found, err := getMemorySlotRow(ctx, s.db, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.Scope, norm.Name, true)
	if err != nil {
		return MemorySlot{}, err
	}
	if found && !existing.Deleted {
		return MemorySlot{}, fmt.Errorf("%w: %s", ErrMemorySlotConflict, norm.Name)
	}
	now := time.Now().Unix()
	slot := norm
	slot.Revision = 1
	slot.CreatedAt = now
	slot.UpdatedAt = now
	if found && existing.CreatedAt > 0 {
		slot.CreatedAt = existing.CreatedAt
	}
	if err := upsertMemorySlotRow(ctx, s.db, slot); err != nil {
		return MemorySlot{}, err
	}
	if err := s.auditMemorySlot(ctx, "create", slot); err != nil {
		return MemorySlot{}, err
	}
	return slot, nil
}

func (s *Service) AppendMemorySlot(ctx context.Context, params MemorySlotParams) (MemorySlot, error) {
	norm, err := s.normalizeMemorySlotParams(params)
	if err != nil {
		return MemorySlot{}, err
	}
	existing, found, err := getMemorySlotRow(ctx, s.db, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.Scope, norm.Name, false)
	if err != nil {
		return MemorySlot{}, err
	}
	if !found {
		return MemorySlot{}, fmt.Errorf("%w: %s", ErrMemorySlotNotFound, norm.Name)
	}
	appendValue := strings.TrimSpace(norm.Value)
	if appendValue == "" {
		return MemorySlot{}, fmt.Errorf("goncho: memory slot value is required")
	}
	if strings.TrimSpace(existing.Value) == "" {
		existing.Value = appendValue
	} else {
		existing.Value = existing.Value + "\n" + appendValue
	}
	if norm.Kind != "" {
		existing.Kind = norm.Kind
	}
	existing.Revision++
	existing.UpdatedAt = time.Now().Unix()
	if err := upsertMemorySlotRow(ctx, s.db, existing); err != nil {
		return MemorySlot{}, err
	}
	if err := s.auditMemorySlot(ctx, "append", existing); err != nil {
		return MemorySlot{}, err
	}
	return existing, nil
}

func (s *Service) ReplaceMemorySlot(ctx context.Context, params MemorySlotParams) (MemorySlot, error) {
	norm, err := s.normalizeMemorySlotParams(params)
	if err != nil {
		return MemorySlot{}, err
	}
	existing, found, err := getMemorySlotRow(ctx, s.db, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.Scope, norm.Name, false)
	if err != nil {
		return MemorySlot{}, err
	}
	if !found {
		return MemorySlot{}, fmt.Errorf("%w: %s", ErrMemorySlotNotFound, norm.Name)
	}
	existing.Value = norm.Value
	if norm.Kind != "" {
		existing.Kind = norm.Kind
	}
	existing.Revision++
	existing.UpdatedAt = time.Now().Unix()
	if err := upsertMemorySlotRow(ctx, s.db, existing); err != nil {
		return MemorySlot{}, err
	}
	if err := s.auditMemorySlot(ctx, "replace", existing); err != nil {
		return MemorySlot{}, err
	}
	return existing, nil
}

func (s *Service) DeleteMemorySlot(ctx context.Context, query MemorySlotQuery) (MemorySlot, error) {
	norm, err := s.normalizeMemorySlotQuery(query)
	if err != nil {
		return MemorySlot{}, err
	}
	if norm.Name == "" {
		return MemorySlot{}, fmt.Errorf("goncho: memory slot name is required")
	}
	existing, found, err := getMemorySlotRow(ctx, s.db, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.Scope, norm.Name, false)
	if err != nil {
		return MemorySlot{}, err
	}
	if !found {
		return MemorySlot{}, fmt.Errorf("%w: %s", ErrMemorySlotNotFound, norm.Name)
	}
	existing.Deleted = true
	existing.Revision++
	existing.UpdatedAt = time.Now().Unix()
	if err := upsertMemorySlotRow(ctx, s.db, existing); err != nil {
		return MemorySlot{}, err
	}
	if err := s.auditMemorySlot(ctx, "delete", existing); err != nil {
		return MemorySlot{}, err
	}
	return existing, nil
}

func (s *Service) GetMemorySlot(ctx context.Context, query MemorySlotQuery) (MemorySlot, error) {
	norm, err := s.normalizeMemorySlotQuery(query)
	if err != nil {
		return MemorySlot{}, err
	}
	if norm.Name == "" {
		return MemorySlot{}, fmt.Errorf("goncho: memory slot name is required")
	}
	slot, found, err := getMemorySlotRow(ctx, s.db, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.Scope, norm.Name, false)
	if err != nil {
		return MemorySlot{}, err
	}
	if !found {
		return MemorySlot{}, fmt.Errorf("%w: %s", ErrMemorySlotNotFound, norm.Name)
	}
	return slot, nil
}

func (s *Service) ListMemorySlots(ctx context.Context, query MemorySlotQuery) (MemorySlotList, error) {
	norm, err := s.normalizeMemorySlotQuery(query)
	if err != nil {
		return MemorySlotList{}, err
	}
	limit := limitutil.Default(query.Limit, 100)
	rows, err := s.db.QueryContext(ctx, `
		SELECT workspace_id, profile_id, peer_id, scope, name, kind, value, revision, deleted, created_at, updated_at
		FROM goncho_memory_slots
		WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND scope = ? AND deleted = 0
		ORDER BY name ASC
		LIMIT ?
	`, norm.WorkspaceID, norm.ProfileID, norm.Peer, norm.Scope, limit)
	if err != nil {
		return MemorySlotList{}, fmt.Errorf("goncho: list memory slots: %w", err)
	}
	defer rows.Close()
	out := MemorySlotList{WorkspaceID: norm.WorkspaceID, ProfileID: norm.ProfileID, Peer: norm.Peer, Scope: norm.Scope, Slots: []MemorySlot{}}
	for rows.Next() {
		slot, err := scanMemorySlot(rows)
		if err != nil {
			return MemorySlotList{}, err
		}
		out.Slots = append(out.Slots, slot)
	}
	if err := rows.Err(); err != nil {
		return MemorySlotList{}, fmt.Errorf("goncho: iterate memory slots: %w", err)
	}
	return out, nil
}

func (s *Service) normalizeMemorySlotParams(params MemorySlotParams) (MemorySlot, error) {
	workspaceID := firstNonBlank(params.WorkspaceID, s.workspaceID)
	profileID := strings.TrimSpace(params.ProfileID)
	peer := strings.TrimSpace(params.Peer)
	name := strings.TrimSpace(params.Name)
	kind := strings.TrimSpace(params.Kind)
	if kind == "" {
		kind = "fact"
	}
	value := strings.TrimSpace(params.Value)
	if workspaceID == "" || peer == "" || name == "" || value == "" {
		return MemorySlot{}, fmt.Errorf("goncho: memory slot workspace_id, peer, name, and value are required")
	}
	return MemorySlot{WorkspaceID: workspaceID, ProfileID: profileID, Peer: peer, Scope: normalizeMemoryScope(params.Scope, profileID), Name: name, Kind: kind, Value: value}, nil
}

func (s *Service) normalizeMemorySlotQuery(query MemorySlotQuery) (MemorySlot, error) {
	workspaceID := firstNonBlank(query.WorkspaceID, s.workspaceID)
	profileID := strings.TrimSpace(query.ProfileID)
	peer := strings.TrimSpace(query.Peer)
	if workspaceID == "" || peer == "" {
		return MemorySlot{}, fmt.Errorf("goncho: memory slot workspace_id and peer are required")
	}
	return MemorySlot{WorkspaceID: workspaceID, ProfileID: profileID, Peer: peer, Scope: normalizeMemoryScope(query.Scope, profileID), Name: strings.TrimSpace(query.Name)}, nil
}

func getMemorySlotRow(ctx context.Context, db *sql.DB, workspaceID, profileID, peer, scope, name string, includeDeleted bool) (MemorySlot, bool, error) {
	query := `
		SELECT workspace_id, profile_id, peer_id, scope, name, kind, value, revision, deleted, created_at, updated_at
		FROM goncho_memory_slots
		WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND scope = ? AND name = ?`
	if !includeDeleted {
		query += ` AND deleted = 0`
	}
	var slot MemorySlot
	err := db.QueryRowContext(ctx, query, workspaceID, profileID, peer, scope, name).Scan(&slot.WorkspaceID, &slot.ProfileID, &slot.Peer, &slot.Scope, &slot.Name, &slot.Kind, &slot.Value, &slot.Revision, dbscan.Bool(&slot.Deleted), &slot.CreatedAt, &slot.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return MemorySlot{}, false, nil
	}
	if err != nil {
		return MemorySlot{}, false, fmt.Errorf("goncho: get memory slot: %w", err)
	}
	return slot, true, nil
}

func upsertMemorySlotRow(ctx context.Context, db *sql.DB, slot MemorySlot) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO goncho_memory_slots(workspace_id, profile_id, peer_id, scope, name, kind, value, revision, deleted, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, profile_id, peer_id, scope, name)
		DO UPDATE SET kind = excluded.kind, value = excluded.value, revision = excluded.revision, deleted = excluded.deleted, updated_at = excluded.updated_at
	`, slot.WorkspaceID, slot.ProfileID, slot.Peer, slot.Scope, slot.Name, slot.Kind, slot.Value, slot.Revision, dbscan.BoolInt(slot.Deleted), slot.CreatedAt, slot.UpdatedAt)
	if err != nil {
		return fmt.Errorf("goncho: upsert memory slot: %w", err)
	}
	return nil
}

func scanMemorySlot(scanner interface{ Scan(...any) error }) (MemorySlot, error) {
	var slot MemorySlot
	if err := scanner.Scan(&slot.WorkspaceID, &slot.ProfileID, &slot.Peer, &slot.Scope, &slot.Name, &slot.Kind, &slot.Value, &slot.Revision, dbscan.Bool(&slot.Deleted), &slot.CreatedAt, &slot.UpdatedAt); err != nil {
		return MemorySlot{}, fmt.Errorf("goncho: scan memory slot: %w", err)
	}
	return slot, nil
}

func (s *Service) auditMemorySlot(ctx context.Context, action string, slot MemorySlot) error {
	_, err := s.Observe(ctx, ObservationParams{
		Kind:       ObservationKindCustom,
		ProfileID:  slot.ProfileID,
		PeerID:     slot.Peer,
		Input:      slot.Name,
		Output:     action,
		ObservedAt: time.Now().UTC(),
		Reason:     "memory_slot_" + action,
		Metadata: map[string]string{
			"custom_kind": "memory_slot",
			"action":      action,
			"slot_name":   slot.Name,
			"slot_scope":  slot.Scope,
			"profile_id":  slot.ProfileID,
			"kind":        slot.Kind,
			"revision":    fmt.Sprintf("%d", slot.Revision),
		},
	})
	return err
}
