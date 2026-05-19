package goncho

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	defaultShortSummaryCadence = 20
	defaultLongSummaryCadence  = 60

	createMessagesLockRetryAttempts = 6
	createMessagesLockRetryMin      = 20 * time.Millisecond
	createMessagesLockRetryMax      = 150 * time.Millisecond
)

// Service is the first in-binary Goncho domain facade. It sits directly on
// top of the SQLite store used by Gormes today.
type Service struct {
	db              *sql.DB
	workspaceID     string
	observer        string
	recentLimit     int
	maxMessageSize  int
	maxFileSize     int
	peerCardEnabled bool
	dreamEnabled    bool
	dreamIdle       time.Duration
	sessions        SessionDirectory
	log             *slog.Logger
	dialecticCaller DialecticCaller
}

const maxPeerCardFacts = 40

type peerCardScope struct {
	Observer string
	Observed string
	Target   string
}

// NewService constructs a Goncho service with conservative defaults.
func NewService(db *sql.DB, cfg Config, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	cfg = cfg.Effective()
	workspaceID := strings.TrimSpace(cfg.WorkspaceID)
	if workspaceID == "" {
		workspaceID = DefaultWorkspaceID
	}
	observer := strings.TrimSpace(cfg.ObserverPeerID)
	if observer == "" {
		observer = DefaultObserverPeerID
	}
	recentLimit := cfg.RecentMessages
	if recentLimit <= 0 {
		recentLimit = DefaultRecentMessages
	}
	return &Service{
		db:              db,
		workspaceID:     workspaceID,
		observer:        observer,
		recentLimit:     recentLimit,
		maxMessageSize:  cfg.MaxMessageSize,
		maxFileSize:     cfg.MaxFileSize,
		peerCardEnabled: cfg.PeerCardEnabled,
		dreamEnabled:    cfg.DreamEnabled,
		dreamIdle:       cfg.DreamIdleTimeout,
		sessions:        cfg.SessionDirectory,
		log:             log,
	}
}

func (s *Service) SetDialecticCaller(dc DialecticCaller) { s.dialecticCaller = dc }

func (s *Service) DialecticCaller() DialecticCaller { return s.dialecticCaller }

func (s *Service) SetProfile(ctx context.Context, peer string, card []string) error {
	scope, err := s.defaultPeerCardScope(peer)
	if err != nil {
		return err
	}
	return upsertPeerCard(ctx, s.db, s.workspaceID, scope.Observer, scope.Observed, normalizePeerCard(card))
}

func (s *Service) SetProfileForTarget(ctx context.Context, peer, target string, card []string) error {
	scope, err := directionalPeerCardScope(peer, target)
	if err != nil {
		return err
	}
	return upsertPeerCard(ctx, s.db, s.workspaceID, scope.Observer, scope.Observed, normalizePeerCard(card))
}

func (s *Service) Profile(ctx context.Context, peer string) (ProfileResult, error) {
	scope, err := s.defaultPeerCardScope(peer)
	if err != nil {
		return ProfileResult{}, err
	}
	return s.profileForScope(ctx, scope)
}

func (s *Service) ProfileForTarget(ctx context.Context, peer, target string) (ProfileResult, error) {
	scope, err := directionalPeerCardScope(peer, target)
	if err != nil {
		return ProfileResult{}, err
	}
	return s.profileForScope(ctx, scope)
}

func (s *Service) defaultPeerCardScope(peer string) (peerCardScope, error) {
	peer = strings.TrimSpace(peer)
	if peer == "" {
		return peerCardScope{}, fmt.Errorf("goncho: peer is required")
	}
	return peerCardScope{
		Observer: s.observer,
		Observed: peer,
	}, nil
}

func directionalPeerCardScope(peer, target string) (peerCardScope, error) {
	peer = strings.TrimSpace(peer)
	target = strings.TrimSpace(target)
	if peer == "" {
		return peerCardScope{}, fmt.Errorf("goncho: peer is required")
	}
	if target == "" {
		return peerCardScope{}, fmt.Errorf("goncho: target is required")
	}
	return peerCardScope{
		Observer: peer,
		Observed: target,
		Target:   target,
	}, nil
}

func (s *Service) profileForScope(ctx context.Context, scope peerCardScope) (ProfileResult, error) {
	if scope.Observer == "" || scope.Observed == "" {
		return ProfileResult{}, fmt.Errorf("goncho: peer is required")
	}
	out := ProfileResult{
		WorkspaceID:    s.workspaceID,
		Peer:           scope.Observed,
		Target:         scope.Target,
		ObserverPeerID: scope.Observer,
		ObservedPeerID: scope.Observed,
		Card:           []string{},
	}
	if !s.peerCardEnabled {
		out.Result = emptyProfileResultText
		out.Hint = profileHint("peer_card_disabled",
			"This is not an error. Peer-card support is disabled in Goncho config, so no curated card can be read for this peer.",
		)
		return out, nil
	}
	card, err := getPeerCard(ctx, s.db, s.workspaceID, scope.Observer, scope.Observed)
	if err != nil {
		return ProfileResult{}, err
	}
	out.Card = card
	if len(card) == 0 {
		out.Result = emptyProfileResultText
		out.Hint = profileHint("peer_card_empty_unknown",
			"This is not an error. The peer card has no facts yet; Goncho builds cards over time from observed turns, and local or self-hosted deployments may not have run enough derivation work yet.",
		)
	}
	return out, nil
}

const emptyProfileResultText = "No profile facts available yet."

func profileHint(code, message string) *ProfileHint {
	return &ProfileHint{
		Code:         code,
		Message:      message + " Try honcho_reasoning for a synthesized answer, or honcho_search to query raw conversation excerpts.",
		Alternatives: []string{"honcho_reasoning", "honcho_search"},
	}
}

func normalizePeerCard(card []string) []string {
	if len(card) > maxPeerCardFacts {
		card = card[:maxPeerCardFacts]
	}
	out := make([]string, len(card))
	copy(out, card)
	return out
}

func (s *Service) Conclude(ctx context.Context, params ConcludeParams) (ConcludeResult, error) {
	peer := strings.TrimSpace(params.Peer)
	if peer == "" {
		return ConcludeResult{}, fmt.Errorf("goncho: peer is required")
	}
	if params.DeleteID > 0 {
		deleted, err := deleteConclusion(ctx, s.db, s.workspaceID, s.observer, peer, params.DeleteID)
		if err != nil {
			return ConcludeResult{}, err
		}
		if !deleted {
			return ConcludeResult{}, fmt.Errorf("goncho: conclusion %d not found", params.DeleteID)
		}
		return ConcludeResult{
			WorkspaceID: s.workspaceID,
			Peer:        peer,
			ID:          params.DeleteID,
			Status:      "processed",
			Deleted:     true,
		}, nil
	}

	conclusion := strings.TrimSpace(params.Conclusion)
	if conclusion == "" {
		return ConcludeResult{}, fmt.Errorf("goncho: conclusion is required when delete_id is absent")
	}

	idempotencyKey := makeIdempotencyKey(s.workspaceID, s.observer, peer, params.SessionKey, conclusion)
	scope := normalizeScope(params.Scope)
	id, status, err := upsertConclusion(ctx, s.db, conclusionRow{
		WorkspaceID:    s.workspaceID,
		ObserverPeerID: s.observer,
		PeerID:         peer,
		SessionKey:     params.SessionKey,
		Content:        conclusion,
		Kind:           "manual",
		Status:         "processed",
		Source:         "manual",
		IdempotencyKey: idempotencyKey,
		EvidenceJSON:   "[]",
		Scope:          scope,
	})
	if err != nil {
		return ConcludeResult{}, err
	}
	if s.dreamEnabled {
		if _, err := s.cancelPendingDreamsForObserved(ctx, peer, time.Now().Unix(), "new_activity"); err != nil {
			return ConcludeResult{}, err
		}
	}

	return ConcludeResult{
		WorkspaceID: s.workspaceID,
		Peer:        peer,
		ID:          id,
		Status:      status,
	}, nil
}

func (s *Service) Search(ctx context.Context, params SearchParams) (SearchResultSet, error) {
	peer := strings.TrimSpace(params.Peer)
	if peer == "" {
		return SearchResultSet{}, fmt.Errorf("goncho: peer is required")
	}
	compiled, err := parseAndCompileSearchFilter(params.Filters, peer)
	if err != nil {
		return SearchResultSet{}, err
	}
	sources, denySources := mergeSearchSources(params.Sources, compiled.Sources)
	if denySources || compiled.DenyAll || filterValuesDenyAll(compiled.SessionIDs) {
		return SearchResultSet{
			WorkspaceID: s.workspaceID,
			Peer:        peer,
			Query:       params.Query,
			Results:     []SearchHit{},
		}, nil
	}
	compiled.Sources = sources
	limit := normalizeSearchLimit(params.Limit)

	var results []SearchHit
	var scopeEvidence *CrossChatRecallEvidence
	if len(compiled.Sources) == 0 || filterHasWildcard(compiled.Sources) {
		results, err = findConclusions(ctx, s.db, s.workspaceID, s.observer, peer, params.Query, params.SessionKey, compiled, limit)
		if err != nil {
			return SearchResultSet{}, err
		}
		if len(results) == 0 && strings.TrimSpace(params.Query) != "" {
			results, err = findConclusions(ctx, s.db, s.workspaceID, s.observer, peer, "", params.SessionKey, compiled, limit)
			if err != nil {
				return SearchResultSet{}, err
			}
		}
	}

	if len(results) == 0 {
		fallback, err := s.searchTurnFallback(ctx, params, compiled, limit)
		if err != nil {
			return SearchResultSet{}, err
		}
		results = fallback.Results
		scopeEvidence = fallback.ScopeEvidence
	}
	results = limitHitsByTokens(results, params.MaxTokens)

	return SearchResultSet{
		WorkspaceID:   s.workspaceID,
		Peer:          peer,
		Query:         params.Query,
		ScopeEvidence: scopeEvidence,
		Results:       results,
	}, nil
}

func (s *Service) Context(ctx context.Context, params ContextParams) (ContextResult, error) {
	peer := strings.TrimSpace(params.Peer)
	if peer == "" {
		return ContextResult{}, fmt.Errorf("goncho: peer is required")
	}
	sessionKey := strings.TrimSpace(params.SessionKey)
	query := effectiveContextQuery(params)
	tokenLimit := effectiveContextTokenLimit(params)
	unavailable := contextUnavailableEvidence(params, s.observer, peer)
	if includeDreamStatus(params) {
		dreamEvidence, err := s.dreamContextUnavailableEvidence(ctx, peer)
		if err != nil {
			return ContextResult{}, err
		}
		unavailable = append(unavailable, dreamEvidence...)
	}

	card, err := getPeerCard(ctx, s.db, s.workspaceID, s.observer, peer)
	if err != nil {
		return ContextResult{}, err
	}

	searchResult := SearchResultSet{
		WorkspaceID: s.workspaceID,
		Peer:        peer,
		Query:       query,
	}
	if limitToSession(params) && sessionKey == "" {
		unavailable = append(unavailable, ContextUnavailableEvidence{
			Field:      "limit_to_session",
			Capability: "session_scoped_representation",
			Reason:     "limit_to_session requires session_key; recall was not widened through scope=user",
		})
	} else {
		scope := params.Scope
		if limitToSession(params) {
			scope = ""
		}
		searchResult, err = s.Search(ctx, SearchParams{
			Peer:       peer,
			Query:      query,
			MaxTokens:  effectiveSearchTokenLimit(params),
			SessionKey: sessionKey,
			Scope:      scope,
			Sources:    params.Sources,
		})
		if err != nil {
			return ContextResult{}, err
		}
	}

	var summary *SessionSummary
	conclusions := make([]string, 0, len(searchResult.Results))
	for _, hit := range searchResult.Results {
		if hit.Source != "conclusion" {
			continue
		}
		conclusions = append(conclusions, hit.Content)
	}

	recentMessages := []MessageSlice{}
	if sessionKey != "" {
		turnCount, err := s.refreshSessionSummaries(ctx, sessionKey)
		if err != nil {
			return ContextResult{}, err
		}

		messageBudget := tokenLimit
		messageStartID := int64(0)
		if includeSummaryComponent(params) {
			var reason string
			summary, reason, err = selectSessionContextSummary(ctx, s.db, s.workspaceID, sessionKey, tokenLimit)
			if err != nil {
				return ContextResult{}, err
			}
			if summary != nil {
				messageStartID = summary.MessageID
				if tokenLimit > 0 {
					_, messageBudget = splitContextTokenBudget(tokenLimit)
				}
			} else if tokenLimit > 0 && turnCount > 0 {
				unavailable = append(unavailable, summaryAbsentEvidence(reason))
			}
		}

		if tokenLimit > 0 {
			recentMessages, err = recentTurnsByTokenBudget(ctx, s.db, sessionKey, messageStartID, messageBudget)
			if err != nil {
				return ContextResult{}, err
			}
		} else {
			recentMessages, err = recentTurnsAfter(ctx, s.db, sessionKey, messageStartID, s.recentLimit)
			if err != nil {
				return ContextResult{}, err
			}
		}
	}

	return ContextResult{
		WorkspaceID:    s.workspaceID,
		Peer:           peer,
		ObserverPeerID: s.observer,
		ObservedPeerID: peer,
		SessionKey:     sessionKey,
		PeerCard:       card,
		Representation: buildRepresentation(peer, card, conclusions),
		Summary:        summary,
		Conclusions:    conclusions,
		SearchResults:  searchResult.Results,
		ScopeEvidence:  searchResult.ScopeEvidence,
		RecentMessages: recentMessages,
		Unavailable:    unavailable,
	}, nil
}

func (s *Service) Chat(ctx context.Context, peer string, params ChatParams) (ChatResult, error) {
	peer = strings.TrimSpace(peer)
	if peer == "" {
		return ChatResult{}, fmt.Errorf("goncho: peer is required")
	}
	query := strings.TrimSpace(params.Query)
	if query == "" {
		return ChatResult{}, fmt.Errorf("goncho: query is required")
	}
	reasoningLevel := normalizeReasoningLevel(params.ReasoningLevel)
	if !ValidDialecticLevel(reasoningLevel) {
		return ChatResult{}, fmt.Errorf("goncho: unsupported reasoning_level %q", params.ReasoningLevel)
	}

	card, err := getPeerCard(ctx, s.db, s.workspaceID, s.observer, peer)
	if err != nil {
		return ChatResult{}, err
	}
	searchResult, err := s.Search(ctx, SearchParams{
		Peer:       peer,
		Query:      query,
		SessionKey: params.SessionID,
	})
	if err != nil {
		return ChatResult{}, err
	}

	unavailable := chatUnavailableEvidence(params)
	content := buildChatContent(peer, query, reasoningLevel, card, searchResult.Results, unavailable)
	if err := insertAssistantChatTurn(ctx, s.db, params.SessionID, peer, content, ""); err != nil {
		return ChatResult{}, err
	}
	return ChatResult{
		Content: content,
	}, nil
}

func (s *Service) CreateMessages(ctx context.Context, params CreateMessagesParams) (CreateMessagesResult, error) {
	sessionKey := strings.TrimSpace(params.SessionKey)
	if sessionKey == "" {
		return CreateMessagesResult{}, fmt.Errorf("goncho: session_key is required")
	}

	var lastErr error
	for attempt := 0; attempt < createMessagesLockRetryAttempts; attempt++ {
		result, err := s.createMessagesOnce(ctx, sessionKey, params.Messages)
		if err == nil {
			return result, nil
		}
		if !isTransientSQLiteLockError(err) {
			return CreateMessagesResult{}, err
		}
		lastErr = err
		if attempt == createMessagesLockRetryAttempts-1 {
			break
		}
		if err := waitCreateMessagesLockRetry(ctx, attempt); err != nil {
			return CreateMessagesResult{}, fmt.Errorf("goncho: create messages retry canceled: %w; last error: %v", err, lastErr)
		}
	}
	return CreateMessagesResult{}, lastErr
}

func (s *Service) createMessagesOnce(ctx context.Context, sessionKey string, inputs []CreateMessage) (CreateMessagesResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CreateMessagesResult{}, fmt.Errorf("goncho: begin create messages: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	messages, err := createLifecycleMessages(ctx, tx, s.workspaceID, sessionKey, s.maxMessageSize, inputs)
	if err != nil {
		return CreateMessagesResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return CreateMessagesResult{}, fmt.Errorf("goncho: commit create messages: %w", err)
	}
	committed = true
	return CreateMessagesResult{
		WorkspaceID: s.workspaceID,
		SessionKey:  sessionKey,
		Messages:    messages,
	}, nil
}

func waitCreateMessagesLockRetry(ctx context.Context, attempt int) error {
	timer := time.NewTimer(createMessagesLockRetryDelay(attempt))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func createMessagesLockRetryDelay(attempt int) time.Duration {
	delay := createMessagesLockRetryMin * time.Duration(attempt+1)
	if delay > createMessagesLockRetryMax {
		return createMessagesLockRetryMax
	}
	return delay
}

func isTransientSQLiteLockError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "database is busy") ||
		strings.Contains(msg, "sqlite_busy") ||
		strings.Contains(msg, "sqlite_locked")
}

func (s *Service) DeleteSession(ctx context.Context, sessionKey string) (SessionDeletionResult, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return SessionDeletionResult{}, fmt.Errorf("goncho: session_key is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SessionDeletionResult{}, fmt.Errorf("goncho: begin delete session: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	result, err := deleteLifecycleSession(ctx, tx, s.workspaceID, sessionKey)
	if err != nil {
		return SessionDeletionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return SessionDeletionResult{}, fmt.Errorf("goncho: commit delete session: %w", err)
	}
	committed = true
	return result, nil
}

func (s *Service) DeleteWorkspace(ctx context.Context) (WorkspaceDeletionResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return WorkspaceDeletionResult{}, fmt.Errorf("goncho: begin delete workspace: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	result, err := deleteLifecycleWorkspace(ctx, tx, s.workspaceID)
	if err != nil {
		return WorkspaceDeletionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return WorkspaceDeletionResult{}, fmt.Errorf("goncho: commit delete workspace: %w", err)
	}
	committed = true
	return result, nil
}

func normalizeReasoningLevel(level string) string {
	level = strings.ToLower(strings.TrimSpace(level))
	if level == "" {
		return string(DialecticLevelLow)
	}
	return level
}

func normalizeScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "global" {
		return "global"
	}
	return "workspace"
}

func chatUnavailableEvidence(params ChatParams) []ContextUnavailableEvidence {
	var unavailable []ContextUnavailableEvidence
	if params.Stream {
		unavailable = append(unavailable, ContextUnavailableEvidence{
			Field:      "stream",
			Capability: "streaming_chat",
			Reason:     "streaming chat transport is unavailable; returning deterministic non-streaming content",
		})
	}
	if strings.TrimSpace(params.Target) != "" {
		unavailable = append(unavailable, ContextUnavailableEvidence{
			Field:      "target",
			Capability: "target_specific_reasoning",
			Reason:     "target-specific dialectic reasoning is unavailable; default observer recall was used",
		})
	}
	return unavailable
}

func effectiveContextQuery(params ContextParams) string {
	if trimmed := strings.TrimSpace(params.SearchQuery); trimmed != "" {
		return trimmed
	}
	return params.Query
}

func effectiveContextTokenLimit(params ContextParams) int {
	if params.Tokens > 0 {
		return params.Tokens
	}
	return params.MaxTokens
}

func effectiveSearchTokenLimit(params ContextParams) int {
	if params.MaxTokens > 0 {
		return params.MaxTokens
	}
	return params.Tokens
}

func includeSummaryComponent(params ContextParams) bool {
	return params.Summary == nil || *params.Summary
}

func splitContextTokenBudget(tokenLimit int) (summaryBudget, messageBudget int) {
	if tokenLimit <= 0 {
		return 0, 0
	}
	summaryBudget = int(float64(tokenLimit) * 0.4)
	messageBudget = tokenLimit - summaryBudget
	return summaryBudget, messageBudget
}

func selectSessionContextSummary(ctx context.Context, db *sql.DB, workspaceID, sessionKey string, tokenLimit int) (*SessionSummary, string, error) {
	shortSummary, longSummary, err := getSessionSummaries(ctx, db, workspaceID, sessionKey)
	if err != nil {
		return nil, "", err
	}
	if shortSummary == nil && longSummary == nil {
		return nil, "no session summary is available yet", nil
	}
	if tokenLimit <= 0 {
		if longSummary != nil {
			return longSummary, "", nil
		}
		return shortSummary, "", nil
	}

	summaryBudget, _ := splitContextTokenBudget(tokenLimit)
	if longSummary != nil && longSummary.TokenCount <= summaryBudget {
		return longSummary, "", nil
	}
	if shortSummary != nil && shortSummary.TokenCount <= summaryBudget {
		return shortSummary, "", nil
	}
	return nil, fmt.Sprintf("session summaries exceed the %d-token summary budget", summaryBudget), nil
}

func summaryAbsentEvidence(reason string) ContextUnavailableEvidence {
	if strings.TrimSpace(reason) == "" {
		reason = "no session summary could fit in the requested token budget"
	}
	return ContextUnavailableEvidence{
		Field:      "summary_absent",
		Capability: "session_summary",
		Reason:     reason,
	}
}

func (s *Service) refreshSessionSummaries(ctx context.Context, sessionKey string) (int, error) {
	count, err := countReadySessionTurns(ctx, s.db, sessionKey)
	if err != nil {
		return 0, err
	}
	for _, cfg := range []struct {
		summaryType string
		cadence     int
	}{
		{summaryType: "short", cadence: defaultShortSummaryCadence},
		{summaryType: "long", cadence: defaultLongSummaryCadence},
	} {
		if err := s.refreshSessionSummarySlot(ctx, sessionKey, cfg.summaryType, cfg.cadence, count); err != nil {
			return 0, err
		}
	}
	return count, nil
}

func (s *Service) refreshSessionSummarySlot(ctx context.Context, sessionKey, summaryType string, cadence, turnCount int) error {
	if cadence <= 0 || turnCount < cadence {
		return nil
	}
	coveredCount := (turnCount / cadence) * cadence
	messageID, err := readySessionTurnIDAtPosition(ctx, s.db, sessionKey, coveredCount)
	if err != nil {
		return err
	}
	if messageID == 0 {
		return nil
	}

	existing, err := getSessionSummary(ctx, s.db, s.workspaceID, sessionKey, summaryType)
	if err != nil {
		return err
	}
	if existing != nil && existing.MessageID >= messageID {
		return nil
	}

	content := deterministicSummaryContent(sessionKey, summaryType, coveredCount, messageID)
	return upsertSessionSummary(ctx, s.db, sessionSummaryRow{
		WorkspaceID: s.workspaceID,
		SessionKey:  sessionKey,
		SummaryType: summaryType,
		Content:     content,
		MessageID:   messageID,
		TokenCount:  approxTokens(content),
	})
}

func deterministicSummaryContent(sessionKey, summaryType string, coveredCount int, messageID int64) string {
	if summaryType == "long" {
		return fmt.Sprintf("long comprehensive summary for session %s covers %d messages through message %d.", sessionKey, coveredCount, messageID)
	}
	return fmt.Sprintf("short summary for session %s covers %d messages through message %d.", sessionKey, coveredCount, messageID)
}

func limitToSession(params ContextParams) bool {
	return params.LimitToSession != nil && *params.LimitToSession
}

func includeDreamStatus(params ContextParams) bool {
	return params.IncludeDreamStatus != nil && *params.IncludeDreamStatus
}

func contextUnavailableEvidence(params ContextParams, defaultObserver, observed string) []ContextUnavailableEvidence {
	var unavailable []ContextUnavailableEvidence
	directionalReason := fmt.Sprintf(
		"directional representation is unavailable; only the default %s observer view was used for %s",
		defaultObserver,
		observed,
	)

	if strings.TrimSpace(params.PeerTarget) != "" {
		unavailable = append(unavailable, ContextUnavailableEvidence{
			Field:      "peer_target",
			Capability: "directional_representation",
			Reason:     directionalReason,
		})
	}
	if strings.TrimSpace(params.PeerPerspective) != "" {
		unavailable = append(unavailable, ContextUnavailableEvidence{
			Field:      "peer_perspective",
			Capability: "directional_representation",
			Reason:     directionalReason,
		})
	}
	if params.SearchTopK != nil {
		unavailable = append(unavailable, unsupportedSemanticRepresentationOption("search_top_k"))
	}
	if params.SearchMaxDistance != nil {
		unavailable = append(unavailable, unsupportedSemanticRepresentationOption("search_max_distance"))
	}
	if params.IncludeMostFrequent != nil {
		unavailable = append(unavailable, unsupportedSemanticRepresentationOption("include_most_frequent"))
	}
	if params.MaxConclusions != nil {
		unavailable = append(unavailable, unsupportedSemanticRepresentationOption("max_conclusions"))
	}
	return unavailable
}

func unsupportedSemanticRepresentationOption(field string) ContextUnavailableEvidence {
	return ContextUnavailableEvidence{
		Field:      field,
		Capability: "semantic_representation_options",
		Reason:     "semantic representation options require the future observation table",
	}
}

func buildRepresentation(peer string, card, conclusions []string) string {
	if len(card) == 0 && len(conclusions) == 0 {
		return "No stored representation for " + peer + "."
	}

	var b strings.Builder
	if len(card) > 0 {
		b.WriteString("Profile facts:")
		for _, item := range card {
			b.WriteString("\n- ")
			b.WriteString(item)
		}
	}
	if len(conclusions) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("Current conclusions:")
		for _, item := range conclusions {
			b.WriteString("\n- ")
			b.WriteString(item)
		}
	}
	return b.String()
}

func buildChatContent(peer, query, reasoningLevel string, card []string, hits []SearchHit, unavailable []ContextUnavailableEvidence) string {
	conclusions := make([]string, 0, len(hits))
	otherEvidence := make([]SearchHit, 0)
	for _, hit := range hits {
		if hit.Source == "conclusion" {
			conclusions = append(conclusions, hit.Content)
			continue
		}
		otherEvidence = append(otherEvidence, hit)
	}

	var b strings.Builder
	b.WriteString("Query: ")
	b.WriteString(query)
	b.WriteString("\n\nReasoning level: ")
	b.WriteString(reasoningLevel)
	b.WriteString("\n\n")
	b.WriteString(buildRepresentation(peer, card, conclusions))

	if len(otherEvidence) > 0 {
		b.WriteString("\n\nRelevant evidence:")
		for _, hit := range otherEvidence {
			b.WriteString("\n- ")
			if strings.TrimSpace(hit.Source) != "" {
				b.WriteString(hit.Source)
				b.WriteString(": ")
			}
			b.WriteString(hit.Content)
		}
	}

	if len(unavailable) > 0 {
		b.WriteString("\n\nUnsupported evidence:")
		for _, item := range unavailable {
			b.WriteString("\n- field=")
			b.WriteString(item.Field)
			b.WriteString(" capability=")
			b.WriteString(item.Capability)
			b.WriteString(" reason=")
			b.WriteString(item.Reason)
		}
	}
	return b.String()
}

func makeIdempotencyKey(workspaceID, observer, peer, sessionKey, conclusion string) string {
	normalized := strings.ToLower(strings.TrimSpace(conclusion))
	sum := sha256.Sum256([]byte(strings.Join([]string{
		workspaceID,
		observer,
		peer,
		strings.TrimSpace(sessionKey),
		normalized,
	}, "\x1f")))
	return hex.EncodeToString(sum[:])
}

type turnFallbackResult struct {
	Results       []SearchHit
	ScopeEvidence *CrossChatRecallEvidence
}

func (s *Service) searchTurnFallback(ctx context.Context, params SearchParams, compiled compiledSearchFilter, limit int) (turnFallbackResult, error) {
	if strings.EqualFold(strings.TrimSpace(params.Scope), "user") {
		userID := strings.TrimSpace(params.Peer)
		filter := SearchFilter{
			UserID:           userID,
			Sources:          compiled.Sources,
			SessionIDs:       compiled.SessionIDs,
			Query:            params.Query,
			CurrentSessionID: params.SessionKey,
			CurrentChatKey:   params.SessionKey,
		}
		if s.sessions == nil {
			evidence := DegradedCrossChatRecallEvidence(filter, "session directory unavailable; same-chat fallback scope used")
			fallback, err := findTurns(ctx, s.db, params.Query, params.SessionKey, compiled, limit)
			if err != nil {
				return turnFallbackResult{}, err
			}
			fallback = attachUnavailableLineageToTurnHits(fallback)
			return turnFallbackResult{Results: fallback, ScopeEvidence: &evidence}, nil
		}
		metas, err := s.sessions.ListMetadataByUserID(ctx, userID)
		if err != nil {
			return turnFallbackResult{}, err
		}
		evidenceMetas, err := s.crossChatEvidenceMetadata(ctx, userID, params.SessionKey, metas)
		if err != nil {
			return turnFallbackResult{}, err
		}
		evidence := ExplainCrossChatRecall(evidenceMetas, filter)
		if evidence.Decision != CrossChatDecisionAllowed {
			fallback, err := findTurns(ctx, s.db, params.Query, params.SessionKey, compiled, limit)
			if err != nil {
				return turnFallbackResult{}, err
			}
			fallback = attachUnavailableLineageToTurnHits(fallback)
			return turnFallbackResult{Results: fallback, ScopeEvidence: &evidence}, nil
		}
		hits, err := SearchMessages(ctx, s.db, metas, filter, limit)
		if errors.Is(err, ErrUserScopeDenied) {
			fallback, err := findTurns(ctx, s.db, params.Query, params.SessionKey, compiled, limit)
			if err != nil {
				return turnFallbackResult{}, err
			}
			fallback = attachUnavailableLineageToTurnHits(fallback)
			return turnFallbackResult{Results: fallback, ScopeEvidence: &evidence}, nil
		}
		if err != nil {
			return turnFallbackResult{}, err
		}
		out := make([]SearchHit, 0, len(hits))
		for _, hit := range hits {
			out = append(out, SearchHit{
				Source:       "turn",
				OriginSource: hit.Source,
				Content:      hit.Content,
				SessionKey:   hit.SessionID,
				Lineage:      searchLineageFromMemory(hit.Lineage),
			})
		}
		return turnFallbackResult{Results: out, ScopeEvidence: &evidence}, nil
	}

	if strings.TrimSpace(params.SessionKey) == "" {
		return turnFallbackResult{}, nil
	}
	results, err := findTurns(ctx, s.db, params.Query, params.SessionKey, compiled, limit)
	if err != nil {
		return turnFallbackResult{}, err
	}
	results = attachUnavailableLineageToTurnHits(results)
	return turnFallbackResult{Results: results}, nil
}

func searchLineageFromMemory(lineage SearchLineage) *SearchLineage {
	status := strings.TrimSpace(lineage.Status)
	if status == "" &&
		strings.TrimSpace(lineage.ParentSessionID) == "" &&
		strings.TrimSpace(lineage.LineageKind) == "" &&
		len(lineage.ChildSessionIDs) == 0 {
		return nil
	}
	if status == "" {
		status = SearchLineageStatusUnavailable
	}
	return &SearchLineage{
		ParentSessionID: strings.TrimSpace(lineage.ParentSessionID),
		LineageKind:     strings.TrimSpace(lineage.LineageKind),
		ChildSessionIDs: append([]string(nil), lineage.ChildSessionIDs...),
		Status:          status,
	}
}

func attachUnavailableLineageToTurnHits(hits []SearchHit) []SearchHit {
	for i := range hits {
		if hits[i].Source != "turn" || hits[i].Lineage != nil {
			continue
		}
		lineage := SearchLineage{Status: SearchLineageStatusUnavailable}
		hits[i].Lineage = &lineage
	}
	return hits
}

type userBindingResolver interface {
	ResolveUserID(ctx context.Context, source, chatID string) (string, bool, error)
}

func (s *Service) crossChatEvidenceMetadata(ctx context.Context, userID, currentKey string, metas []SessionMetadata) ([]SessionMetadata, error) {
	out := append([]SessionMetadata(nil), metas...)
	resolver, ok := s.sessions.(userBindingResolver)
	if !ok {
		return out, nil
	}
	source, chatID, ok := splitChatKey(currentKey)
	if !ok {
		return out, nil
	}
	boundUserID, found, err := resolver.ResolveUserID(ctx, source, chatID)
	if err != nil {
		return nil, err
	}
	if !found || strings.TrimSpace(boundUserID) == "" || strings.TrimSpace(boundUserID) == userID {
		return out, nil
	}
	out = append(out, SessionMetadata{
		SessionID: strings.TrimSpace(currentKey),
		Source:    source,
		ChatID:    chatID,
		UserID:    boundUserID,
	})
	return out, nil
}

func splitChatKey(key string) (source, chatID string, ok bool) {
	source, chatID, ok = strings.Cut(strings.TrimSpace(key), ":")
	source = strings.TrimSpace(source)
	chatID = strings.TrimSpace(chatID)
	return source, chatID, ok && source != "" && chatID != ""
}

func limitHitsByTokens(hits []SearchHit, maxTokens int) []SearchHit {
	if maxTokens <= 0 || len(hits) == 0 {
		return hits
	}

	used := 0
	out := make([]SearchHit, 0, len(hits))
	for _, hit := range hits {
		cost := approxTokens(hit.Content)
		if used+cost > maxTokens && len(out) > 0 {
			break
		}
		out = append(out, hit)
		used += cost
	}
	return out
}

func approxTokens(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 1
	}
	if n := len(strings.Fields(text)); n > 0 {
		return n
	}
	return 1
}
