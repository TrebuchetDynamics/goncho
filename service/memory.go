package goncho

import (
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
	"gopkg.in/yaml.v3"
)

// Cross-chat recall constants.
const (
	CrossChatDecisionAllowed       = "allowed"
	CrossChatDecisionDenied        = "denied"
	CrossChatDecisionDegraded      = "degraded"
	CrossChatFallbackSameChat      = "same-chat"
	SearchLineageStatusUnavailable = "unavailable"
)

// Lineage constants (mirrors internal/session).
const (
	LineageKindPrimary     = "primary"
	LineageKindCompression = "compression"
	LineageKindFork        = "fork"
	LineageStatusOK        = "ok"
	LineageStatusMissing   = "missing"
	LineageStatusOrphan    = "orphan"
	LineageStatusLoop      = "loop"
	LineageStatusError     = "error"
)

var ErrUserScopeDenied = errors.New("memory: user scope denied")

// SearchFilter narrows cross-session search to one canonical user.
type SearchFilter struct {
	UserID           string
	Sources          []string
	SessionIDs       []string
	Query            string
	Roles            []string
	CurrentSessionID string
	CurrentChatKey   string
}

// CrossChatSessionEvidence is the operator-readable identity for one session.
type CrossChatSessionEvidence struct {
	SessionID string `json:"session_id"`
	Source    string `json:"source,omitempty"`
	ChatID    string `json:"chat_id,omitempty"`
	ChatKey   string `json:"chat_key,omitempty"`
	Current   bool   `json:"current,omitempty"`
}

// CrossChatRecallEvidence explains why a user-scoped recall/search request was
// allowed, denied, or degraded.
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

// SearchLineage is the lineage evidence attached to one matched session.

// MessageSearchHit is one turn-level result from the session catalog.
type MessageSearchHit struct {
	SessionID string
	ChatID    string
	Source    string
	Role      string
	Content   string
	TSUnix    int64
	Lineage   SearchLineage
}

// ExplainCrossChatRecall returns evidence for a user-scoped widening decision.
func ExplainCrossChatRecall(metas []SessionMetadata, filter SearchFilter) CrossChatRecallEvidence {
	evidence := CrossChatRecallEvidence{
		Scope:            "user",
		UserID:           strings.TrimSpace(filter.UserID),
		CurrentSessionID: strings.TrimSpace(filter.CurrentSessionID),
		CurrentChatKey:   strings.TrimSpace(filter.CurrentChatKey),
		SourceAllowlist:  normalizeSources(filter.Sources),
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
		binding := crossChatSessionEvidence(meta, true)
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

	allowedSources := normalizeSources(filter.Sources)
	allowedSessions := normalizeSessionIDs(filter.SessionIDs)
	for _, meta := range metas {
		metaUserID := strings.TrimSpace(meta.UserID)
		if metaUserID != evidence.UserID {
			continue
		}
		if len(allowedSources) > 0 && !slices.Contains(allowedSources, strings.ToLower(strings.TrimSpace(meta.Source))) {
			continue
		}
		if len(allowedSessions) > 0 && !slices.Contains(allowedSessions, strings.TrimSpace(meta.SessionID)) {
			continue
		}
		item := crossChatSessionEvidence(meta, metadataMatchesCurrent(meta, evidence.CurrentSessionID, evidence.CurrentChatKey))
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

// DegradedCrossChatRecallEvidence reports that a caller asked for user scope
// but the diagnostic path could not inspect the session binding dependency.
func DegradedCrossChatRecallEvidence(filter SearchFilter, reason string) CrossChatRecallEvidence {
	evidence := CrossChatRecallEvidence{
		Decision:         CrossChatDecisionDegraded,
		Scope:            "user",
		FallbackScope:    CrossChatFallbackSameChat,
		Reason:           strings.TrimSpace(reason),
		UserID:           strings.TrimSpace(filter.UserID),
		CurrentSessionID: strings.TrimSpace(filter.CurrentSessionID),
		CurrentChatKey:   strings.TrimSpace(filter.CurrentChatKey),
		SourceAllowlist:  normalizeSources(filter.Sources),
	}
	if evidence.Reason == "" {
		evidence.Reason = "cross-chat recall evidence unavailable; same-chat fallback scope used"
	}
	return evidence
}

func deniedCrossChatEvidence(evidence CrossChatRecallEvidence, reason string) CrossChatRecallEvidence {
	evidence.Decision = CrossChatDecisionDenied
	evidence.FallbackScope = CrossChatFallbackSameChat
	evidence.Reason = reason
	evidence.Sessions = nil
	evidence.SessionsConsidered = 0
	evidence.WidenedSessionsConsidered = 0
	return evidence
}

func crossChatSessionEvidence(meta SessionMetadata, current bool) CrossChatSessionEvidence {
	item := CrossChatSessionEvidence{
		SessionID: strings.TrimSpace(meta.SessionID),
		Source:    strings.TrimSpace(meta.Source),
		ChatID:    strings.TrimSpace(meta.ChatID),
		Current:   current,
	}
	item.ChatKey = canonicalChatKey(meta)
	return item
}

func metadataMatchesCurrent(meta SessionMetadata, currentSessionID, currentChatKey string) bool {
	if currentSessionID != "" && strings.TrimSpace(meta.SessionID) == currentSessionID {
		return true
	}
	return currentChatKey != "" && sameChatKey(canonicalChatKey(meta), currentChatKey)
}

func canonicalChatKey(meta SessionMetadata) string {
	source := strings.TrimSpace(meta.Source)
	chatID := strings.TrimSpace(meta.ChatID)
	if source == "" || chatID == "" {
		return ""
	}
	return source + ":" + chatID
}

func sameChatKey(a, b string) bool {
	return strings.ToLower(strings.TrimSpace(a)) == strings.ToLower(strings.TrimSpace(b))
}

func normalizeSources(sources []string) []string {
	return textutil.UniqueLowerTrimmed(sources, false)
}

func normalizeSessionIDs(sessionIDs []string) []string {
	out := textutil.UniqueTrimmed(sessionIDs, false)
	if slices.Contains(out, "*") {
		return nil
	}
	return out
}

func metadataIndexes(metas []SessionMetadata) ([]string, []string, map[string]SessionMetadata, map[string]SessionMetadata) {
	sessionIDs := make([]string, 0, len(metas))
	chatKeys := make([]string, 0, len(metas))
	metaBySession := make(map[string]SessionMetadata, len(metas))
	metaByChat := make(map[string]SessionMetadata, len(metas))
	for _, meta := range metas {
		if sessionID := strings.TrimSpace(meta.SessionID); sessionID != "" {
			sessionIDs = append(sessionIDs, sessionID)
			metaBySession[sessionID] = meta
		}
		if chatKey := canonicalChatKey(meta); chatKey != "" {
			chatKeys = append(chatKeys, chatKey)
			metaByChat[chatKey] = meta
		}
	}
	return sessionIDs, chatKeys, metaBySession, metaByChat
}

func selectMetadata(metas []SessionMetadata, filter SearchFilter) ([]SessionMetadata, error) {
	userID := strings.TrimSpace(filter.UserID)
	if userID == "" {
		return nil, nil
	}
	currentSessionID := strings.TrimSpace(filter.CurrentSessionID)
	currentChatKey := strings.TrimSpace(filter.CurrentChatKey)
	requireCurrentBinding := currentSessionID != "" || currentChatKey != ""
	currentBindingMatched := !requireCurrentBinding

	allowedSources := normalizeSources(filter.Sources)
	allowedSessions := normalizeSessionIDs(filter.SessionIDs)
	selected := make([]SessionMetadata, 0, len(metas))
	for _, meta := range metas {
		metaUserID := strings.TrimSpace(meta.UserID)
		if metadataMatchesCurrent(meta, currentSessionID, currentChatKey) {
			if metaUserID != userID {
				return nil, fmt.Errorf("%w: current binding belongs to %q", ErrUserScopeDenied, metaUserID)
			}
			currentBindingMatched = true
		}
		if metaUserID != userID {
			continue
		}
		if len(allowedSources) > 0 && !slices.Contains(allowedSources, strings.ToLower(strings.TrimSpace(meta.Source))) {
			continue
		}
		if len(allowedSessions) > 0 && !slices.Contains(allowedSessions, strings.TrimSpace(meta.SessionID)) {
			continue
		}
		selected = append(selected, meta)
	}
	if !currentBindingMatched {
		return nil, ErrUserScopeDenied
	}
	return selected, nil
}

type searchLineageIndex struct {
	bySession map[string]SessionMetadata
	children  map[string][]string
}

func buildSearchLineageIndex(metas []SessionMetadata) searchLineageIndex {
	idx := searchLineageIndex{
		bySession: make(map[string]SessionMetadata, len(metas)),
		children:  make(map[string][]string, len(metas)),
	}
	for _, meta := range metas {
		meta = normalizeSearchLineageMetadata(meta)
		if meta.SessionID == "" {
			continue
		}
		idx.bySession[meta.SessionID] = meta
	}
	for _, meta := range idx.bySession {
		if meta.ParentSessionID == "" {
			continue
		}
		childIDs := idx.children[meta.ParentSessionID]
		if !slices.Contains(childIDs, meta.SessionID) {
			idx.children[meta.ParentSessionID] = append(childIDs, meta.SessionID)
		}
	}
	for parentID := range idx.children {
		slices.Sort(idx.children[parentID])
	}
	return idx
}

func normalizeSearchLineageMetadata(meta SessionMetadata) SessionMetadata {
	meta.SessionID = strings.TrimSpace(meta.SessionID)
	meta.ParentSessionID = strings.TrimSpace(meta.ParentSessionID)
	meta.LineageKind = strings.ToLower(strings.TrimSpace(meta.LineageKind))
	return meta
}

func (idx searchLineageIndex) contextFor(sessionID string) SearchLineage {
	sessionID = strings.TrimSpace(sessionID)
	meta, ok := idx.bySession[sessionID]
	if !ok {
		return SearchLineage{Status: SearchLineageStatusUnavailable}
	}
	children := textutil.CloneStrings(idx.children[sessionID])
	return SearchLineage{
		ParentSessionID: meta.ParentSessionID,
		LineageKind:     searchLineageKind(meta),
		ChildSessionIDs: children,
		Status:          idx.statusFor(sessionID),
	}
}

func searchLineageKind(meta SessionMetadata) string {
	if meta.LineageKind == "" {
		return LineageKindPrimary
	}
	return meta.LineageKind
}

func (idx searchLineageIndex) statusFor(sessionID string) string {
	meta, ok := idx.bySession[sessionID]
	if !ok {
		return SearchLineageStatusUnavailable
	}
	seen := map[string]struct{}{}
	seen[sessionID] = struct{}{}
	for current := meta.ParentSessionID; current != ""; {
		if _, ok := seen[current]; ok {
			return LineageStatusLoop
		}
		seen[current] = struct{}{}

		parent, ok := idx.bySession[current]
		if !ok {
			return LineageStatusOrphan
		}
		current = parent.ParentSessionID
	}
	return LineageStatusOK
}

func sanitizeFTS5Pattern(raw string) string {
	re := regexp.MustCompile(`[^\w\s\*+"]`)
	return strings.TrimSpace(re.ReplaceAllString(raw, " "))
}

func buildTurnSearchQuery(rawQuery string, sessionIDs, chatKeys, roles []string, limit int, sessionsOnly bool) (string, []any) {
	var b strings.Builder
	args := make([]any, 0, len(sessionIDs)+len(chatKeys)+len(roles)+2)
	if sessionsOnly {
		b.WriteString(`SELECT t.session_id, t.chat_id, MAX(t.ts_unix) AS latest_turn_unix FROM turns t`)
	} else {
		b.WriteString(`SELECT t.session_id, t.chat_id, t.role, t.content, t.ts_unix FROM turns t`)
	}

	query := sanitizeFTS5Pattern(rawQuery)
	if query != "" {
		b.WriteString(` JOIN turns_fts fts ON fts.rowid = t.id WHERE turns_fts MATCH ?`)
		args = append(args, query)
	} else {
		b.WriteString(` WHERE 1=1`)
	}

	b.WriteString(` AND (`)
	appendInClause(&b, "t.session_id", sessionIDs, &args)
	if len(chatKeys) > 0 {
		b.WriteString(` OR `)
		appendInClause(&b, "t.chat_id", chatKeys, &args)
	}
	b.WriteString(`)`)
	b.WriteString(` AND t.memory_sync_status = 'ready'`)
	if normalizedRoles := normalizeRoles(roles); len(normalizedRoles) > 0 {
		b.WriteString(` AND `)
		appendInClause(&b, "t.role", normalizedRoles, &args)
	}

	if sessionsOnly {
		b.WriteString(` GROUP BY t.session_id, t.chat_id ORDER BY latest_turn_unix DESC, t.session_id ASC LIMIT ?`)
	} else {
		b.WriteString(` ORDER BY t.ts_unix DESC, t.id DESC LIMIT ?`)
	}
	args = append(args, limit)
	return b.String(), args
}

func normalizeRoles(roles []string) []string {
	return textutil.UniqueLowerTrimmed(roles, false)
}

// SearchMessages returns matching turns across the canonical sessions bound to
// one user, optionally narrowed to a subset of sources.
func SearchMessages(ctx context.Context, db *sql.DB, metas []SessionMetadata, filter SearchFilter, limit int) ([]MessageSearchHit, error) {
	selected, err := selectMetadata(metas, filter)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 || limit == 0 {
		return nil, nil
	}

	sessionIDs, chatKeys, metaBySession, metaByChat := metadataIndexes(selected)
	lineage := buildSearchLineageIndex(selected)
	query, args := buildTurnSearchQuery(filter.Query, sessionIDs, chatKeys, filter.Roles, limit, false)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("session catalog: search messages: %w", err)
	}
	defer rows.Close()

	var hits []MessageSearchHit
	for rows.Next() {
		var hit MessageSearchHit
		if err := rows.Scan(&hit.SessionID, &hit.ChatID, &hit.Role, &hit.Content, &hit.TSUnix); err != nil {
			return nil, fmt.Errorf("session catalog: scan message hit: %w", err)
		}
		if meta, ok := metaBySession[hit.SessionID]; ok {
			hit.Source = meta.Source
		} else if meta, ok := metaByChat[hit.ChatID]; ok {
			hit.Source = meta.Source
		}
		hit.Lineage = lineage.contextFor(hit.SessionID)
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("session catalog: iterate message hits: %w", err)
	}
	return hits, nil
}

// Goncho Memory V1 constants.
const (
	GonchoMemoryV1ContractVersion = "1"
	GonchoMemoryV1MarkdownFormat  = "1"
	GonchoMemoryV1MCPToolContract = "1"
)

// GonchoMemoryV1Item is a single memory entry in the V1 contract.
type GonchoMemoryV1Document struct {
	FormatVersion   string               `json:"format_version"`
	ContractVersion string               `json:"contract_version"`
	Items           []GonchoMemoryV1Item `json:"items"`
}

type GonchoMemoryV1Item struct {
	MemoryID        string   `json:"memory_id" yaml:"memory_id"`
	Revision        int      `json:"revision" yaml:"revision"`
	AgentID         string   `json:"agent_id" yaml:"agent_id"`
	WorkspaceID     string   `json:"workspace_id" yaml:"workspace_id"`
	PeerID          string   `json:"peer_id" yaml:"peer_id"`
	SessionID       string   `json:"session_id" yaml:"session_id"`
	Scope           string   `json:"scope" yaml:"scope"`
	State           string   `json:"state" yaml:"state"`
	SourceKind      string   `json:"source_kind" yaml:"source_kind"`
	SourceTurnID    string   `json:"source_turn_id,omitempty" yaml:"source_turn_id,omitempty"`
	TombstonedAt    string   `json:"tombstoned_at,omitempty" yaml:"tombstoned_at,omitempty"`
	TombstoneReason string   `json:"tombstone_reason,omitempty" yaml:"tombstone_reason,omitempty"`
	Checksum        string   `json:"checksum" yaml:"checksum"`
	Tags            []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Importance      float64  `json:"importance" yaml:"importance"`
	CreatedAt       string   `json:"created_at" yaml:"created_at"`
	UpdatedAt       string   `json:"updated_at" yaml:"updated_at"`
	ProvenanceJSON  string   `json:"provenance_json,omitempty" yaml:"provenance_json,omitempty"`
	Content         string   `json:"content" yaml:"-"`
}

// GonchoMemoryV1Checksum returns a SHA-256 hex digest of the content.
func GonchoMemoryV1Checksum(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// ValidateGonchoMemoryV1Item checks required fields on a V1 item.
func ValidateGonchoMemoryV1Item(item GonchoMemoryV1Item) error {
	if strings.TrimSpace(item.MemoryID) == "" {
		return errors.New("goncho: memory_id is required")
	}
	if strings.TrimSpace(item.Content) == "" {
		return errors.New("goncho: content is required")
	}
	if strings.TrimSpace(item.WorkspaceID) == "" {
		return errors.New("goncho: workspace_id is required")
	}
	if strings.TrimSpace(item.PeerID) == "" {
		return errors.New("goncho: peer_id is required")
	}
	return nil
}

// GonchoMemoryV1RecallRequest controls a V1 recall operation.
type GonchoMemoryV1RecallRequest struct {
	WorkspaceID string
	PeerID      string
	Query       string
	Limit       int
	Scope       string
	Sources     []string
	SessionID   string
}

// ParseGonchoMemoryV1Markdown parses markdown into V1 documents.
func ParseGonchoMemoryV1Markdown(raw string) (GonchoMemoryV1Document, error) {
	doc := GonchoMemoryV1Document{FormatVersion: GonchoMemoryV1MarkdownFormat, ContractVersion: GonchoMemoryV1ContractVersion}
	scanner := bufio.NewScanner(strings.NewReader(raw))
	var current *GonchoMemoryV1Item
	var content strings.Builder
	inBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, gonchoMemoryV1MarkerStart) {
			inBlock = true
			current = &GonchoMemoryV1Item{State: "active", Scope: "private"}
			content.Reset()
			continue
		}
		if strings.HasPrefix(line, gonchoMemoryV1ClosingMarker) {
			if current != nil {
				current.Content = strings.TrimSpace(content.String())
				current.Checksum = GonchoMemoryV1Checksum(current.Content)
				doc.Items = append(doc.Items, *current)
			}
			inBlock = false
			current = nil
			continue
		}
		if inBlock && current != nil {
			if strings.HasPrefix(line, gonchoMemoryV1MarkerEnd) {
				continue
			}
			if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "#") {
				if err := parseV1YAMLMeta(current, line); err != nil {
					return doc, err
				}
			} else {
				if content.Len() > 0 {
					content.WriteByte('\n')
				}
				content.WriteString(line)
			}
		}
	}
	return doc, scanner.Err()
}

func parseV1YAMLMeta(item *GonchoMemoryV1Item, line string) error {
	line = strings.TrimPrefix(line, "- ")
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	var m map[string]any
	if err := yaml.Unmarshal([]byte(line), &m); err != nil {
		return nil
	}
	for k, v := range m {
		switch k {
		case "memory_id":
			if s, ok := v.(string); ok {
				item.MemoryID = s
			}
		case "revision":
			if n, ok := v.(int); ok {
				item.Revision = n
			}
		case "agent_id":
			if s, ok := v.(string); ok {
				item.AgentID = s
			}
		case "workspace_id":
			if s, ok := v.(string); ok {
				item.WorkspaceID = s
			}
		case "peer_id":
			if s, ok := v.(string); ok {
				item.PeerID = s
			}
		case "session_id":
			if s, ok := v.(string); ok {
				item.SessionID = s
			}
		case "scope":
			if s, ok := v.(string); ok {
				item.Scope = s
			}
		case "state":
			if s, ok := v.(string); ok {
				item.State = s
			}
		case "source_kind":
			if s, ok := v.(string); ok {
				item.SourceKind = s
			}
		case "importance":
			if f, ok := v.(float64); ok {
				item.Importance = f
			}
		case "tags":
			if tags, ok := v.([]any); ok {
				item.Tags = make([]string, len(tags))
				for i, t := range tags {
					if s, ok := t.(string); ok {
						item.Tags[i] = s
					}
				}
			}
		}
	}
	return nil
}

// GonchoMemoryV1ContractInfo returns the V1 contract metadata.
func GonchoMemoryV1ContractInfo() map[string]any {
	return map[string]any{
		"contract_version":                   GonchoMemoryV1ContractVersion,
		"markdown_format_version":            GonchoMemoryV1MarkdownFormat,
		"mcp_tool_contract_version":          GonchoMemoryV1MCPToolContract,
		"private_agent_memory_default":       true,
		"self_improvement_per_agent_default": true,
		"foreign_config_runtime_reads":       "denied",
		"fast_recall_path":                   []string{"sqlite", "fts"},
		"optional_quality_layers":            []string{"embeddings", "qmd", "dialectic"},
	}
}

// CanRecallGonchoMemoryV1 checks if the V1 memory tables exist.
func CanRecallGonchoMemoryV1(ctx context.Context, db *sql.DB) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('goncho_memory_items','goncho_memory_items_fts','goncho_memory_eval_artifacts')`).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 3, nil
}

// GonchoMarkdownStoreConfig controls the markdown-backed memory store.
type GonchoMarkdownStoreConfig struct {
	WorkspaceID string
	ObserverID  string
	FilePath    string
}

// GonchoMarkdownStore is the markdown-backed memory store.
type GonchoMarkdownStore struct {
	db          *sql.DB
	workspaceID string
	observerID  string
	filePath    string
}

// NewGonchoMarkdownStore creates a new markdown-backed memory store.
func NewGonchoMarkdownStore(db *sql.DB, cfg GonchoMarkdownStoreConfig) (*GonchoMarkdownStore, error) {
	return &GonchoMarkdownStore{
		db:          db,
		workspaceID: cfg.WorkspaceID,
		observerID:  cfg.ObserverID,
		filePath:    cfg.FilePath,
	}, nil
}

const (
	gonchoMemoryV1MarkerStart   = "<!-- goncho-memory"
	gonchoMemoryV1MarkerEnd     = "-->"
	gonchoMemoryV1ClosingMarker = "<!-- /goncho-memory -->"
)
