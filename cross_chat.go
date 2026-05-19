package goncho

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrUserScopeDenied is returned when a user-scoped cross-chat search cannot
// be widened beyond the current session.
var ErrUserScopeDenied = errors.New("goncho: user scope denied")

// Cross-chat decision constants.
const (
	CrossChatDecisionAllowed    = "allowed"
	CrossChatDecisionDenied     = "denied"
	CrossChatDecisionDegraded   = "degraded"
	CrossChatFallbackSameChat   = "same-chat"
)

// SearchFilter narrows cross-session search to one canonical user and an
// optional set of transport sources.
type SearchFilter struct {
	UserID           string
	Sources          []string
	SessionIDs       []string
	Query            string
	Roles            []string
	CurrentSessionID string
	CurrentChatKey   string
}

// SearchMessage is one message hit returned by cross-session search.
type SearchMessage struct {
	ID         int64
	SessionKey string
	Role       string
	Content    string
	CreatedAt  int64
	Sequence   int
	UserID     string
	Source     string
}

// MessageSearchHit is one search result with optional lineage evidence.
type MessageSearchHit struct {
	Message SearchMessage
	Lineage SearchLineage
}

// CrossChatSessionEvidence is the operator-readable identity for one session
// considered by a user-scoped cross-chat recall request.
type CrossChatSessionEvidence struct {
	SessionID string `json:"session_id"`
	Source    string `json:"source,omitempty"`
	ChatID    string `json:"chat_id,omitempty"`
	ChatKey   string `json:"chat_key,omitempty"`
	Current   bool   `json:"current,omitempty"`
}

// CrossChatRecallEvidence explains why a user-scoped recall/search request was
// allowed, denied, or degraded before the query result set is rendered.
type CrossChatRecallEvidence struct {
	Decision                  string                     `json:"decision"`
	Scope                     string                     `json:"scope"`
	FallbackScope             string                     `json:"fallback_scope,omitempty"`
	Reason                    string                     `json:"reason,omitempty"`
	UserID                    string                     `json:"user_id,omitempty"`
	CurrentSessionID          string                     `json:"current_session_id,omitempty"`
	CurrentChatKey            string                     `json:"current_chat_key,omitempty"`
	CurrentBinding            *CrossChatSessionEvidence  `json:"current_binding,omitempty"`
	SourceAllowlist           []string                   `json:"source_allowlist,omitempty"`
	SessionsConsidered        int                        `json:"sessions_considered"`
	WidenedSessionsConsidered int                        `json:"widened_sessions_considered"`
	Sessions                  []CrossChatSessionEvidence `json:"sessions,omitempty"`
}

// SessionMetadata is the minimal session metadata needed for user-scoped
// cross-chat search. Extracted from gormes-agent/internal/session to make
// goncho standalone.
type SessionMetadata struct {
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

const (
	LineageKindCompression = "compression"
	LineageKindFork        = "fork"
)

// MemMap is an in-memory session directory for testing.
type MemMap struct {
	mu   sync.RWMutex
	data map[string]SessionMetadata
}

// NewMemMap creates an empty in-memory session directory.
func NewMemMap() *MemMap {
	return &MemMap{data: make(map[string]SessionMetadata)}
}

func (m *MemMap) PutMetadata(ctx context.Context, meta SessionMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[meta.SessionID] = meta
	return nil
}

func (m *MemMap) GetMetadata(ctx context.Context, sessionID string) (SessionMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	meta, ok := m.data[sessionID]
	if !ok {
		return SessionMetadata{}, fmt.Errorf("session not found: %s", sessionID)
	}
	return meta, nil
}

func (m *MemMap) ListMetadata(ctx context.Context) ([]SessionMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]SessionMetadata, 0, len(m.data))
	for _, meta := range m.data {
		result = append(result, meta)
	}
	return result, nil
}

func (m *MemMap) ListMetadataByUserID(ctx context.Context, userID string) ([]SessionMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []SessionMetadata
	for _, meta := range m.data {
		if meta.UserID == userID {
			result = append(result, meta)
		}
	}
	return result, nil
}

// DegradedCrossChatRecallEvidence returns a degraded evidence record when the
// session directory is unavailable or cannot resolve user identity.
func DegradedCrossChatRecallEvidence(filter SearchFilter, reason string) CrossChatRecallEvidence {
	return CrossChatRecallEvidence{
		Decision:       CrossChatDecisionDegraded,
		Scope:          "user",
		FallbackScope:  CrossChatFallbackSameChat,
		Reason:         reason,
		UserID:         strings.TrimSpace(filter.UserID),
		CurrentSessionID: strings.TrimSpace(filter.CurrentSessionID),
		CurrentChatKey:   strings.TrimSpace(filter.CurrentChatKey),
		SourceAllowlist:  normalizeStrings(filter.Sources),
	}
}

// ExplainCrossChatRecall mirrors the session catalog scope selection logic and
// returns the evidence operators need to audit a user-scoped widening decision.
func ExplainCrossChatRecall(metas []SessionMetadata, filter SearchFilter) CrossChatRecallEvidence {
	evidence := CrossChatRecallEvidence{
		Scope:            "user",
		UserID:           strings.TrimSpace(filter.UserID),
		CurrentSessionID: strings.TrimSpace(filter.CurrentSessionID),
		CurrentChatKey:   strings.TrimSpace(filter.CurrentChatKey),
		SourceAllowlist:  normalizeStrings(filter.Sources),
	}
	if evidence.UserID == "" {
		return deniedCrossChatEvidence(evidence, "unresolved user_id; same-chat fallback scope used")
	}

	requireCurrentBinding := evidence.CurrentSessionID != "" || evidence.CurrentChatKey != ""
	currentBindingMatched := !requireCurrentBinding
	for _, meta := range metas {
		if !metadataMatchesCurrent(meta, evidence.CurrentSessionID, evidence.CurrentChatKey) {
			continue
		}
		binding := crossChatSessionEvidenceFromMeta(meta, true)
		evidence.CurrentBinding = &binding
		metaUserID := strings.TrimSpace(meta.UserID)
		if metaUserID == "" {
			return deniedCrossChatEvidence(evidence, "unresolved current binding: current chat/session has no user_id; same-chat fallback scope used")
		}
		if metaUserID != evidence.UserID {
			return deniedCrossChatEvidence(evidence, fmt.Sprintf("conflicting current binding: current binding belongs to %q; same-chat fallback scope used", metaUserID))
		}
		currentBindingMatched = true
	}
	if !currentBindingMatched {
		return deniedCrossChatEvidence(evidence, "unknown current binding: current chat/session is not bound to the requested user_id; same-chat fallback scope used")
	}

	allowedSources := normalizeStrings(filter.Sources)
	allowedSessions := normalizeStrings(filter.SessionIDs)
	for _, meta := range metas {
		metaUserID := strings.TrimSpace(meta.UserID)
		if metaUserID != evidence.UserID {
			continue
		}
		if len(allowedSources) > 0 && !containsString(allowedSources, strings.ToLower(strings.TrimSpace(meta.Source))) {
			continue
		}
		if len(allowedSessions) > 0 && !containsString(allowedSessions, strings.TrimSpace(meta.SessionID)) {
			continue
		}
		item := crossChatSessionEvidenceFromMeta(meta, metadataMatchesCurrent(meta, evidence.CurrentSessionID, evidence.CurrentChatKey))
		evidence.Sessions = append(evidence.Sessions, item)
		if !item.Current {
			evidence.WidenedSessionsConsidered++
		}
	}
	if len(evidence.Sessions) == 0 {
		return deniedCrossChatEvidence(evidence, fmt.Sprintf("unresolved user binding: no sessions matched user_id %q and source/session filters; same-chat fallback scope used", evidence.UserID))
	}

	evidence.Decision = CrossChatDecisionAllowed
	evidence.Reason = "resolved user_id and current binding; user-scope recall may widen across listed sessions"
	evidence.SessionsConsidered = len(evidence.Sessions)
	return evidence
}

// SearchMessages performs a cross-session search over the goncho messages table.
func SearchMessages(ctx context.Context, db *sql.DB, metas []SessionMetadata, filter SearchFilter, limit int) ([]MessageSearchHit, error) {
	// Build session ID set from metadata
	sessionIDs := make(map[string]string) // session_key -> user_id
	for _, meta := range metas {
		metaUserID := strings.TrimSpace(meta.UserID)
		if metaUserID != strings.TrimSpace(filter.UserID) {
			continue
		}
		if len(filter.Sources) > 0 && !containsString(normalizeStrings(filter.Sources), strings.ToLower(strings.TrimSpace(meta.Source))) {
			continue
		}
		if len(filter.SessionIDs) > 0 && !containsString(normalizeStrings(filter.SessionIDs), strings.TrimSpace(meta.SessionID)) {
			continue
		}
		key := sessionKey(meta.SessionID, meta.Source, meta.ChatID)
		sessionIDs[key] = metaUserID
	}

	// Check current binding
	if filter.CurrentSessionID != "" || filter.CurrentChatKey != "" {
		found := false
		for _, meta := range metas {
			if metadataMatchesCurrent(meta, filter.CurrentSessionID, filter.CurrentChatKey) {
				if strings.TrimSpace(meta.UserID) != strings.TrimSpace(filter.UserID) {
					return nil, fmt.Errorf("%w: current binding belongs to %q", ErrUserScopeDenied, strings.TrimSpace(meta.UserID))
				}
				found = true
				break
			}
		}
		if !found {
			return nil, ErrUserScopeDenied
		}
	}

	if len(sessionIDs) == 0 {
		return nil, ErrUserScopeDenied
	}

	// Build query
	query := `SELECT id, session_key, role, content, created_at, seq_in_session
	          FROM goncho_messages
	          WHERE session_key IN (`
	args := make([]any, 0, len(sessionIDs)+3)
	i := 0
	for key := range sessionIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, key)
		i++
	}
	query += ")"

	if filter.Query != "" {
		query += " AND content LIKE ?"
		args = append(args, "%"+filter.Query+"%")
	}
	if len(filter.Roles) > 0 {
		query += " AND role IN ("
		for j, role := range filter.Roles {
			if j > 0 {
				query += ","
			}
			query += "?"
			args = append(args, role)
		}
		query += ")"
	}

	query += " ORDER BY created_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []MessageSearchHit
	for rows.Next() {
		var msg SearchMessage
		err := rows.Scan(&msg.ID, &msg.SessionKey, &msg.Role, &msg.Content, &msg.CreatedAt, &msg.Sequence)
		if err != nil {
			return nil, err
		}
		hits = append(hits, MessageSearchHit{
			Message: msg,
			Lineage: SearchLineage{Status: SearchLineageStatusUnavailable},
		})
	}
	return hits, rows.Err()
}

// Helper functions

func deniedCrossChatEvidence(evidence CrossChatRecallEvidence, reason string) CrossChatRecallEvidence {
	evidence.Decision = CrossChatDecisionDenied
	evidence.Reason = reason
	evidence.FallbackScope = CrossChatFallbackSameChat
	return evidence
}

func metadataMatchesCurrent(meta SessionMetadata, currentSessionID, currentChatKey string) bool {
	if currentSessionID != "" && strings.TrimSpace(meta.SessionID) == strings.TrimSpace(currentSessionID) {
		return true
	}
	if currentChatKey != "" {
		key := sessionKey(meta.SessionID, meta.Source, meta.ChatID)
		if key == strings.TrimSpace(currentChatKey) {
			return true
		}
	}
	return false
}

func crossChatSessionEvidenceFromMeta(meta SessionMetadata, current bool) CrossChatSessionEvidence {
	return CrossChatSessionEvidence{
		SessionID: meta.SessionID,
		Source:    meta.Source,
		ChatID:    meta.ChatID,
		ChatKey:   sessionKey(meta.SessionID, meta.Source, meta.ChatID),
		Current:   current,
	}
}

func sessionKey(sessionID, source, chatID string) string {
	parts := []string{sessionID, source, chatID}
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return strings.Join(parts, ":")
}

func normalizeStrings(s []string) []string {
	out := make([]string, 0, len(s))
	for _, v := range s {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func containsString(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
