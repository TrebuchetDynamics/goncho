package goncho

import (
	"context"
	"strings"
	"time"
)

// Config controls the minimal Goncho service defaults for a runtime.
type Config struct {
	Enabled                      bool
	WorkspaceID                  string
	ObserverPeerID               string
	RecentMessages               int
	MaxMessageSize               int
	MaxFileSize                  int
	GetContextMaxTokens          int
	ReasoningEnabled             bool
	PeerCardEnabled              bool
	SummaryEnabled               bool
	DreamEnabled                 bool
	DreamIdleTimeout             time.Duration
	DeriverWorkers               int
	RepresentationBatchMaxTokens int
	DialecticDefaultLevel        DialecticLevel
	SessionDirectory             SessionDirectory
}

type DialecticLevel string

const (
	DialecticLevelMinimal DialecticLevel = "minimal"
	DialecticLevelLow     DialecticLevel = "low"
	DialecticLevelMedium  DialecticLevel = "medium"
	DialecticLevelHigh    DialecticLevel = "high"
	DialecticLevelMax     DialecticLevel = "max"
)

const (
	MemoryScopeProfile   = "profile"
	MemoryScopeWorkspace = "workspace"
	MemoryScopeShared    = "shared"
	MemoryScopeSession   = "session"
	MemoryScopeGlobal    = "global"
)

type MemoryNamespace struct {
	WorkspaceID      string `json:"workspace_id"`
	ProfileID        string `json:"profile_id"`
	PeerID           string `json:"peer_id,omitempty"`
	Scope            string `json:"scope,omitempty"`
	ProfileDirectory string `json:"profile_directory,omitempty"`
}

const (
	DefaultRecentMessages               = 4
	DefaultMaxMessageSize               = 25_000
	DefaultMaxFileSize                  = 5_242_880
	DefaultGetContextMaxTokens          = 100_000
	DefaultDeriverWorkers               = 1
	DefaultRepresentationBatchMaxTokens = 1024
	DefaultDreamMinConclusions          = 50
	DefaultDreamCooldown                = 8 * time.Hour
	DefaultDreamIdleTimeout             = time.Hour
)

// Effective fills the Go-native Goncho defaults used when older callers still
// construct Config directly instead of going through internal/config.
func (c Config) Effective() Config {
	out := c
	peerCardExplicitlyDisabled := out.Enabled && !out.PeerCardEnabled
	out.Enabled = true
	if strings.TrimSpace(out.WorkspaceID) == "" {
		out.WorkspaceID = DefaultWorkspaceID
	}
	if strings.TrimSpace(out.ObserverPeerID) == "" {
		out.ObserverPeerID = DefaultObserverPeerID
	}
	if out.RecentMessages <= 0 {
		out.RecentMessages = DefaultRecentMessages
	}
	if out.MaxMessageSize <= 0 {
		out.MaxMessageSize = DefaultMaxMessageSize
	}
	if out.MaxFileSize <= 0 {
		out.MaxFileSize = DefaultMaxFileSize
	}
	if out.GetContextMaxTokens <= 0 {
		out.GetContextMaxTokens = DefaultGetContextMaxTokens
	}
	out.ReasoningEnabled = true
	out.PeerCardEnabled = !peerCardExplicitlyDisabled
	out.SummaryEnabled = true
	if out.DreamIdleTimeout <= 0 {
		out.DreamIdleTimeout = DefaultDreamIdleTimeout
	}
	if out.DeriverWorkers <= 0 {
		out.DeriverWorkers = DefaultDeriverWorkers
	}
	if out.RepresentationBatchMaxTokens <= 0 {
		out.RepresentationBatchMaxTokens = DefaultRepresentationBatchMaxTokens
	}
	if !ValidDialecticLevel(string(out.DialecticDefaultLevel)) {
		out.DialecticDefaultLevel = DialecticLevelLow
	}
	return out
}

func ValidDialecticLevel(level string) bool {
	switch DialecticLevel(strings.ToLower(strings.TrimSpace(level))) {
	case DialecticLevelMinimal, DialecticLevelLow, DialecticLevelMedium, DialecticLevelHigh, DialecticLevelMax:
		return true
	default:
		return false
	}
}

// SessionMetadata is the goncho-owned subset of session directory metadata
// needed for user-scoped cross-chat search. This type exists so goncho can be
// extracted as a standalone repository without importing internal/session.
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

// SessionDirectory exposes the canonical user->session metadata seam needed
// for user-scoped cross-chat search.
type SessionDirectory interface {
	ListMetadataByUserID(ctx context.Context, userID string) ([]SessionMetadata, error)
}

// ProfileResult is the external shape used by profile reads and updates.
type ProfileResult struct {
	WorkspaceID    string       `json:"workspace_id"`
	ProfileID      string       `json:"profile_id,omitempty"`
	Peer           string       `json:"peer"`
	Target         string       `json:"target,omitempty"`
	ObserverPeerID string       `json:"observer_peer_id,omitempty"`
	ObservedPeerID string       `json:"observed_peer_id,omitempty"`
	Card           []string     `json:"card"`
	Result         string       `json:"result,omitempty"`
	Hint           *ProfileHint `json:"hint,omitempty"`
}

// ProfileHint gives honcho_profile callers actionable guidance when an empty
// peer card is a valid non-error state.
type ProfileHint struct {
	Code         string   `json:"code"`
	Message      string   `json:"message"`
	Alternatives []string `json:"alternatives"`
}

// ConcludeParams controls manual conclusion writes and deletes.
type ConcludeParams struct {
	ProfileID  string `json:"profile_id,omitempty"`
	Peer       string `json:"peer"`
	Conclusion string `json:"conclusion,omitempty"`
	DeleteID   int64  `json:"delete_id,omitempty"`
	SessionKey string `json:"session_key,omitempty"`
	Scope      string `json:"scope,omitempty"`
}

// ConcludeResult is the stable JSON shape for honcho_conclude.
// ConcludeResult represents the outcome of a create/delete conclusion operation.
type ConcludeResult struct {
	WorkspaceID string `json:"workspace_id"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	ID          int64  `json:"id,omitempty"`
	Status      string `json:"status"`
	Deleted     bool   `json:"deleted,omitempty"`
}

// SearchParams controls retrieval for honcho_search.
type SearchParams struct {
	ProfileID  string         `json:"profile_id,omitempty"`
	Peer       string         `json:"peer"`
	Query      string         `json:"query"`
	MaxTokens  int            `json:"max_tokens,omitempty"`
	SessionKey string         `json:"session_key,omitempty"`
	Scope      string         `json:"scope,omitempty"`
	Sources    []string       `json:"sources,omitempty"`
	Filters    map[string]any `json:"filters,omitempty"`
	Limit      int            `json:"limit,omitempty"`
}

// SearchHit is one result entry returned by search.
type SearchHit struct {
	ID           int64          `json:"id,omitempty"`
	Source       string         `json:"source"`
	OriginSource string         `json:"origin_source,omitempty"`
	Content      string         `json:"content"`
	SessionKey   string         `json:"session_key,omitempty"`
	Lineage      *SearchLineage `json:"lineage,omitempty"`
}

// SearchLineage is operator evidence for the session lineage attached to a
// search hit.
type SearchLineage struct {
	ParentSessionID string   `json:"parent_session_id,omitempty"`
	LineageKind     string   `json:"lineage_kind,omitempty"`
	ChildSessionIDs []string `json:"child_session_ids,omitempty"`
	Status          string   `json:"status"`
}

// SearchResultSet is the stable JSON shape for honcho_search.
type SearchResultSet struct {
	WorkspaceID   string                   `json:"workspace_id"`
	ProfileID     string                   `json:"profile_id,omitempty"`
	Peer          string                   `json:"peer"`
	Query         string                   `json:"query"`
	ScopeEvidence *CrossChatRecallEvidence `json:"scope_evidence,omitempty"`
	Results       []SearchHit              `json:"results"`
}

// ContextParams controls honcho_context reads.
type ContextParams struct {
	ProfileID           string   `json:"profile_id,omitempty"`
	Peer                string   `json:"peer"`
	Query               string   `json:"query,omitempty"`
	SearchQuery         string   `json:"search_query,omitempty"`
	MaxTokens           int      `json:"max_tokens,omitempty"`
	Tokens              int      `json:"tokens,omitempty"`
	Summary             *bool    `json:"summary,omitempty"`
	SessionKey          string   `json:"session_key,omitempty"`
	Scope               string   `json:"scope,omitempty"`
	Sources             []string `json:"sources,omitempty"`
	PeerTarget          string   `json:"peer_target,omitempty"`
	PeerPerspective     string   `json:"peer_perspective,omitempty"`
	LimitToSession      *bool    `json:"limit_to_session,omitempty"`
	SearchTopK          *int     `json:"search_top_k,omitempty"`
	SearchMaxDistance   *float64 `json:"search_max_distance,omitempty"`
	IncludeMostFrequent *bool    `json:"include_most_frequent,omitempty"`
	MaxConclusions      *int     `json:"max_conclusions,omitempty"`
	IncludeDreamStatus  *bool    `json:"include_dream_status,omitempty"`
}

// MessageSlice is one recent message excerpt included in context responses.
type MessageSlice struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Message is a goncho-owned chat message type used by session summary extraction.
// It replaces hermes.Message to enable standalone extraction.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StructuredSummary captures session-end facts for cross-session continuity.
type StructuredSummary struct {
	FilesModified []string `json:"files_modified,omitempty"`
	DecisionsMade []string `json:"decisions_made,omitempty"`
	OpenQuestions []string `json:"open_questions,omitempty"`
	SkillOutcomes []string `json:"skill_outcomes,omitempty"`
	NextSteps     []string `json:"next_steps,omitempty"`
}

// SessionSummary is the summary component returned by session context when a
// short or long summary slot fits inside the requested summary budget.
type SessionSummary struct {
	Content     string `json:"content"`
	MessageID   int64  `json:"message_id"`
	SummaryType string `json:"summary_type"`
	CreatedAt   int64  `json:"created_at"`
	TokenCount  int    `json:"token_count"`
}

// ContextUnavailableEvidence names a requested context capability that Goncho
// accepted but cannot yet fulfill with the current local storage model.
type ContextUnavailableEvidence struct {
	Field      string `json:"field"`
	Capability string `json:"capability"`
	Reason     string `json:"reason"`
}

// ContextResult is the stable JSON shape for honcho_context.
type ContextResult struct {
	WorkspaceID       string                       `json:"workspace_id"`
	ProfileID         string                       `json:"profile_id,omitempty"`
	Peer              string                       `json:"peer"`
	ObserverPeerID    string                       `json:"observer_peer_id,omitempty"`
	ObservedPeerID    string                       `json:"observed_peer_id,omitempty"`
	SessionKey        string                       `json:"session_key,omitempty"`
	PeerCard          []string                     `json:"peer_card"`
	Representation    string                       `json:"representation"`
	Summary           *SessionSummary              `json:"summary,omitempty"`
	StructuredSummary *StructuredSummary           `json:"structured_summary,omitempty"`
	Conclusions       []string                     `json:"conclusions,omitempty"`
	SearchResults     []SearchHit                  `json:"search_results,omitempty"`
	ScopeEvidence     *CrossChatRecallEvidence     `json:"scope_evidence,omitempty"`
	RecentMessages    []MessageSlice               `json:"recent_messages,omitempty"`
	Unavailable       []ContextUnavailableEvidence `json:"unavailable,omitempty"`
}

// ChatParams mirrors Honcho's DialecticOptions request body for peer.chat().
// The peer itself is path/tool context, so it is passed separately to Service.Chat.
type ChatParams struct {
	SessionID      string `json:"session_id,omitempty"`
	Target         string `json:"target,omitempty"`
	Query          string `json:"query"`
	Stream         bool   `json:"stream,omitempty"`
	ReasoningLevel string `json:"reasoning_level,omitempty"`
}

// ChatResult is Honcho's non-streaming dialectic response shape.
type ChatResult struct {
	Content string `json:"content"`
}

// CreateMessagesParams mirrors Honcho's session message creation contract at
// the local Goncho service boundary.
type CreateMessagesParams struct {
	SessionKey string          `json:"session_key"`
	Messages   []CreateMessage `json:"messages"`
}

type CreateMessage struct {
	ProfileID string         `json:"profile_id,omitempty"`
	Peer      string         `json:"peer_id"`
	Role      string         `json:"role,omitempty"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
}

type CreateMessagesResult struct {
	WorkspaceID string          `json:"workspace_id"`
	SessionKey  string          `json:"session_key"`
	Messages    []MessageRecord `json:"messages"`
}

type MessageRecord struct {
	ID          int64          `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	SessionKey  string         `json:"session_key"`
	Peer        string         `json:"peer_id"`
	Role        string         `json:"role"`
	Content     string         `json:"content"`
	Sequence    int            `json:"seq_in_session"`
	CreatedAt   int64          `json:"created_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type SessionDeletionResult struct {
	WorkspaceID        string `json:"workspace_id"`
	SessionKey         string `json:"session_key"`
	MessagesDeleted    int64  `json:"messages_deleted"`
	ConclusionsDeleted int64  `json:"conclusions_deleted"`
	SummariesDeleted   int64  `json:"summaries_deleted"`
}

type WorkspaceDeletionResult struct {
	WorkspaceID        string `json:"workspace_id"`
	MessagesDeleted    int64  `json:"messages_deleted"`
	PeerCardsDeleted   int64  `json:"peer_cards_deleted"`
	ConclusionsDeleted int64  `json:"conclusions_deleted"`
	SummariesDeleted   int64  `json:"summaries_deleted"`
	DreamsDeleted      int64  `json:"dreams_deleted"`
}

type DialecticCaller interface {
	Chat(ctx context.Context, peer string, systemPrompt string, query string) (string, error)
}
