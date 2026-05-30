package goncho

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
)

type ImageEmbeddingStatus string

const ImageEmbeddingDeferred ImageEmbeddingStatus = "deferred"

var imageMemoryDDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_image_memories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id TEXT NOT NULL,
		profile_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL,
		session_key TEXT NOT NULL DEFAULT '',
		image_ref TEXT NOT NULL,
		checksum TEXT NOT NULL,
		alt_text TEXT NOT NULL DEFAULT '',
		embedding_status TEXT NOT NULL DEFAULT 'deferred',
		metadata_json TEXT NOT NULL DEFAULT '{}',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		UNIQUE(workspace_id, profile_id, peer_id, checksum)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_image_memories_lookup ON goncho_image_memories(workspace_id, profile_id, peer_id, updated_at DESC)`,
}

type ImageMemoryParams struct {
	WorkspaceID string            `json:"workspace_id,omitempty"`
	ProfileID   string            `json:"profile_id,omitempty"`
	Peer        string            `json:"peer"`
	SessionKey  string            `json:"session_key,omitempty"`
	ImageRef    string            `json:"image_ref"`
	Checksum    string            `json:"checksum"`
	AltText     string            `json:"alt_text,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ImageMemoryQuery struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	Query       string `json:"query,omitempty"`
	SessionKey  string `json:"session_key,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type ImageMemory struct {
	ID              int64                `json:"id"`
	WorkspaceID     string               `json:"workspace_id"`
	ProfileID       string               `json:"profile_id,omitempty"`
	Peer            string               `json:"peer"`
	SessionKey      string               `json:"session_key,omitempty"`
	ImageRef        string               `json:"image_ref"`
	Checksum        string               `json:"checksum"`
	AltText         string               `json:"alt_text,omitempty"`
	EmbeddingStatus ImageEmbeddingStatus `json:"embedding_status"`
	Metadata        map[string]string    `json:"metadata,omitempty"`
	CreatedAt       int64                `json:"created_at"`
	UpdatedAt       int64                `json:"updated_at"`
	Replayed        bool                 `json:"replayed,omitempty"`
}

type ImageMemoryList struct {
	WorkspaceID string        `json:"workspace_id"`
	ProfileID   string        `json:"profile_id,omitempty"`
	Peer        string        `json:"peer"`
	Images      []ImageMemory `json:"images"`
}

func (s *Service) StoreImageMemory(ctx context.Context, params ImageMemoryParams) (ImageMemory, error) {
	workspaceID := firstNonBlank(params.WorkspaceID, s.workspaceID)
	profileID := strings.TrimSpace(params.ProfileID)
	peer := strings.TrimSpace(params.Peer)
	imageRef := strings.TrimSpace(params.ImageRef)
	checksum := strings.TrimSpace(params.Checksum)
	if workspaceID == "" || peer == "" || imageRef == "" || checksum == "" {
		return ImageMemory{}, fmt.Errorf("goncho: image memory workspace_id, peer, image_ref, and checksum are required")
	}
	existing, found, err := getImageMemoryByChecksum(ctx, s.db, workspaceID, profileID, peer, checksum)
	if err != nil {
		return ImageMemory{}, err
	}
	if found {
		existing.Replayed = true
		return existing, nil
	}
	metadataJSON, err := marshalImageMetadata(params.Metadata)
	if err != nil {
		return ImageMemory{}, err
	}
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO goncho_image_memories(workspace_id, profile_id, peer_id, session_key, image_ref, checksum, alt_text, embedding_status, metadata_json, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, workspaceID, profileID, peer, strings.TrimSpace(params.SessionKey), imageRef, checksum, strings.TrimSpace(params.AltText), string(ImageEmbeddingDeferred), metadataJSON, now, now)
	if err != nil {
		return ImageMemory{}, fmt.Errorf("goncho: store image memory: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ImageMemory{}, fmt.Errorf("goncho: image memory id: %w", err)
	}
	return ImageMemory{ID: id, WorkspaceID: workspaceID, ProfileID: profileID, Peer: peer, SessionKey: strings.TrimSpace(params.SessionKey), ImageRef: imageRef, Checksum: checksum, AltText: strings.TrimSpace(params.AltText), EmbeddingStatus: ImageEmbeddingDeferred, Metadata: cloneStringMap(params.Metadata), CreatedAt: now, UpdatedAt: now}, nil
}

func (s *Service) SearchImageMemories(ctx context.Context, query ImageMemoryQuery) (ImageMemoryList, error) {
	workspaceID := firstNonBlank(query.WorkspaceID, s.workspaceID)
	profileID := strings.TrimSpace(query.ProfileID)
	peer := strings.TrimSpace(query.Peer)
	if workspaceID == "" || peer == "" {
		return ImageMemoryList{}, fmt.Errorf("goncho: image memory workspace_id and peer are required")
	}
	limit := limitutil.Default(query.Limit, 20)
	base := `SELECT id, workspace_id, profile_id, peer_id, session_key, image_ref, checksum, alt_text, embedding_status, metadata_json, created_at, updated_at FROM goncho_image_memories WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND embedding_status != 'archived'`
	args := []any{workspaceID, profileID, peer}
	if sessionKey := strings.TrimSpace(query.SessionKey); sessionKey != "" {
		base += ` AND session_key = ?`
		args = append(args, sessionKey)
	}
	if q := strings.TrimSpace(query.Query); q != "" {
		base += ` AND (checksum = ? OR image_ref LIKE ? OR alt_text LIKE ?)`
		like := "%" + q + "%"
		args = append(args, q, like, like)
	}
	base += ` ORDER BY updated_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, base, args...)
	if err != nil {
		return ImageMemoryList{}, fmt.Errorf("goncho: search image memories: %w", err)
	}
	defer rows.Close()
	out := ImageMemoryList{WorkspaceID: workspaceID, ProfileID: profileID, Peer: peer, Images: []ImageMemory{}}
	for rows.Next() {
		image, err := scanImageMemory(rows)
		if err != nil {
			return ImageMemoryList{}, err
		}
		out.Images = append(out.Images, image)
	}
	if err := rows.Err(); err != nil {
		return ImageMemoryList{}, fmt.Errorf("goncho: iterate image memories: %w", err)
	}
	return out, nil
}

func getImageMemoryByChecksum(ctx context.Context, db *sql.DB, workspaceID, profileID, peer, checksum string) (ImageMemory, bool, error) {
	row := db.QueryRowContext(ctx, `SELECT id, workspace_id, profile_id, peer_id, session_key, image_ref, checksum, alt_text, embedding_status, metadata_json, created_at, updated_at FROM goncho_image_memories WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? AND checksum = ?`, workspaceID, profileID, peer, checksum)
	image, err := scanImageMemory(row)
	if err == sql.ErrNoRows {
		return ImageMemory{}, false, nil
	}
	if err != nil {
		return ImageMemory{}, false, err
	}
	return image, true, nil
}

func scanImageMemory(scanner interface{ Scan(...any) error }) (ImageMemory, error) {
	var image ImageMemory
	var status string
	var metadataJSON string
	if err := scanner.Scan(&image.ID, &image.WorkspaceID, &image.ProfileID, &image.Peer, &image.SessionKey, &image.ImageRef, &image.Checksum, &image.AltText, &status, &metadataJSON, &image.CreatedAt, &image.UpdatedAt); err != nil {
		return ImageMemory{}, err
	}
	metadata := map[string]string{}
	if strings.TrimSpace(metadataJSON) != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			return ImageMemory{}, fmt.Errorf("goncho: decode image metadata: %w", err)
		}
	}
	image.EmbeddingStatus = ImageEmbeddingStatus(status)
	image.Metadata = metadata
	return image, nil
}

func marshalImageMetadata(metadata map[string]string) (string, error) {
	if metadata == nil {
		metadata = map[string]string{}
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("goncho: marshal image metadata: %w", err)
	}
	return string(raw), nil
}
