package session

import (
	"context"
	"strings"
	"sync"
	"time"
)

const (
	LineageKindPrimary     = "primary"
	LineageKindCompression = "compression"
	LineageKindFork        = "fork"
)

type Metadata struct {
	SessionID       string `json:"session_id"`
	Source          string `json:"source,omitempty"`
	ChatID          string `json:"chat_id,omitempty"`
	UserID          string `json:"user_id,omitempty"`
	Title           string `json:"title,omitempty"`
	ParentSessionID string `json:"parent_session_id,omitempty"`
	LineageKind     string `json:"lineage_kind"`
	CreatedAt       int64  `json:"created_at,omitempty"`
	UpdatedAt       int64  `json:"updated_at"`
}

type MemMap struct {
	mu        sync.Mutex
	kv        map[string]string
	meta      map[string]Metadata
	chatUsers map[string]string
}

func NewMemMap() *MemMap {
	return &MemMap{
		kv:        map[string]string{},
		meta:      map[string]Metadata{},
		chatUsers: map[string]string{},
	}
}

func (m *MemMap) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.kv[key], nil
}

func (m *MemMap) Put(ctx context.Context, key, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.TrimSpace(sessionID) == "" {
		delete(m.kv, key)
		return nil
	}
	m.kv[key] = sessionID
	return nil
}

func (m *MemMap) PutMetadata(ctx context.Context, meta Metadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	meta = normalizeMetadata(meta)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.meta[meta.SessionID] = meta
	if meta.Source != "" && meta.ChatID != "" && meta.UserID != "" {
		m.chatUsers[chatBindingKey(meta.Source, meta.ChatID)] = meta.UserID
	}
	return nil
}

func (m *MemMap) ResolveUserID(ctx context.Context, source, chatID string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	userID, ok := m.chatUsers[chatBindingKey(strings.TrimSpace(source), strings.TrimSpace(chatID))]
	return userID, ok, nil
}

func (m *MemMap) ListMetadataByUserID(ctx context.Context, userID string) ([]Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Metadata{}
	for _, meta := range m.meta {
		if meta.UserID == userID {
			out = append(out, meta)
		}
	}
	return out, nil
}

func (*MemMap) Close() error { return nil }

func normalizeMetadata(meta Metadata) Metadata {
	meta.SessionID = strings.TrimSpace(meta.SessionID)
	meta.Source = strings.TrimSpace(meta.Source)
	meta.ChatID = strings.TrimSpace(meta.ChatID)
	meta.UserID = strings.TrimSpace(meta.UserID)
	meta.Title = strings.TrimSpace(meta.Title)
	meta.ParentSessionID = strings.TrimSpace(meta.ParentSessionID)
	meta.LineageKind = strings.TrimSpace(meta.LineageKind)
	if meta.LineageKind == "" {
		meta.LineageKind = LineageKindPrimary
	}
	now := time.Now().Unix()
	if meta.CreatedAt == 0 {
		meta.CreatedAt = now
	}
	if meta.UpdatedAt == 0 {
		meta.UpdatedAt = meta.CreatedAt
	}
	return meta
}

func chatBindingKey(source, chatID string) string {
	return source + "\x00" + chatID
}
