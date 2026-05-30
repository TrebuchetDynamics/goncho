package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/hashutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
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
	db               *sql.DB
	workspaceID      string
	observer         string
	recentLimit      int
	maxMessageSize   int
	maxFileSize      int
	peerCardEnabled  bool
	dreamEnabled     bool
	dreamIdle        time.Duration
	sessions         SessionDirectory
	vectorStore      VectorStore
	searchReranker   SearchReranker
	providerRegistry *ProviderHealthRegistry
	log              *slog.Logger
	dialecticCaller  DialecticCaller
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
		db:               db,
		workspaceID:      workspaceID,
		observer:         observer,
		recentLimit:      recentLimit,
		maxMessageSize:   cfg.MaxMessageSize,
		maxFileSize:      cfg.MaxFileSize,
		peerCardEnabled:  cfg.PeerCardEnabled,
		dreamEnabled:     cfg.DreamEnabled,
		dreamIdle:        cfg.DreamIdleTimeout,
		sessions:         cfg.SessionDirectory,
		vectorStore:      cfg.VectorStore,
		searchReranker:   cfg.SearchReranker,
		providerRegistry: NewProviderHealthRegistry(providerResilienceConfigFromServiceConfig(cfg), cfg.VectorStore),
		log:              log,
	}
}

func (s *Service) SetDialecticCaller(dc DialecticCaller) { s.dialecticCaller = dc }

func (s *Service) DialecticCaller() DialecticCaller { return s.dialecticCaller }

func (s *Service) SetProfile(ctx context.Context, peer string, card []string) error {
	scope, err := s.defaultPeerCardScope(peer)
	if err != nil {
		return err
	}
	return upsertPeerCard(ctx, s.db, s.workspaceID, "", scope.Observer, scope.Observed, normalizePeerCard(card))
}

func (s *Service) SetProfileForTarget(ctx context.Context, peer, target string, card []string) error {
	scope, err := directionalPeerCardScope(peer, target)
	if err != nil {
		return err
	}
	return upsertPeerCard(ctx, s.db, s.workspaceID, "", scope.Observer, scope.Observed, normalizePeerCard(card))
}

func (s *Service) SetProfileInNamespace(ctx context.Context, ns MemoryNamespace, card []string) error {
	profileID := strings.TrimSpace(ns.ProfileID)
	peer := strings.TrimSpace(ns.PeerID)
	if peer == "" {
		return fmt.Errorf("goncho: peer_id is required")
	}
	workspaceID := strings.TrimSpace(ns.WorkspaceID)
	if workspaceID == "" {
		workspaceID = s.workspaceID
	}
	return upsertPeerCard(ctx, s.db, workspaceID, profileID, s.observer, peer, normalizePeerCard(card))
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

func (s *Service) ProfileInNamespace(ctx context.Context, ns MemoryNamespace) (ProfileResult, error) {
	profileID := strings.TrimSpace(ns.ProfileID)
	peer := strings.TrimSpace(ns.PeerID)
	if peer == "" {
		return ProfileResult{}, fmt.Errorf("goncho: peer_id is required")
	}
	workspaceID := strings.TrimSpace(ns.WorkspaceID)
	if workspaceID == "" {
		workspaceID = s.workspaceID
	}
	out := ProfileResult{WorkspaceID: workspaceID, ProfileID: profileID, Peer: peer, ObserverPeerID: s.observer, ObservedPeerID: peer, Card: []string{}}
	if !s.peerCardEnabled {
		out.Result = emptyProfileResultText
		out.Hint = profileHint("peer_card_disabled", "This is not an error. Peer-card support is disabled in Goncho config, so no curated card can be read for this peer.")
		return out, nil
	}
	card, err := getPeerCard(ctx, s.db, workspaceID, profileID, s.observer, peer)
	if err != nil {
		return ProfileResult{}, err
	}
	out.Card = card
	if len(card) == 0 {
		out.Result = emptyProfileResultText
		out.Hint = profileHint("peer_card_empty_unknown", "This is not an error. The peer card has no facts yet; Goncho builds cards over time from observed turns, and local or self-hosted deployments may not have run enough derivation work yet.")
	}
	return out, nil
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
	card, err := getPeerCard(ctx, s.db, s.workspaceID, "", scope.Observer, scope.Observed)
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
	return sliceutil.Clone(card)
}

func (s *Service) Conclude(ctx context.Context, params ConcludeParams) (ConcludeResult, error) {
	return s.conclusions().Conclude(ctx, params)
}

func (s *Service) Search(ctx context.Context, params SearchParams) (SearchResultSet, error) {
	return s.retrieval().Search(ctx, params)
}

func (s *Service) Context(ctx context.Context, params ContextParams) (ContextResult, error) {
	return s.retrieval().Context(ctx, params)
}

// Recall runs the full scored recall pipeline against stored conclusions and
// returns a deterministic trace with candidates, scores, selection reasoning,
// and warnings. Unlike Search, which returns flat result rows, Recall exposes
// the scoring and provenance chain so hosts can audit why each memory was
// selected or rejected.
func (s *Service) Recall(ctx context.Context, q RecallQuery) (RecallTrace, error) {
	return s.recallWithOptions(ctx, q, recallPipelineOptions{})
}

// RecallWithScoringConfig runs Recall with an explicit scoring configuration.
// It is intended for evaluation harnesses and advanced integrations that need
// to compare ranking profiles without changing the default Service.Recall
// behavior.
func (s *Service) RecallWithScoringConfig(ctx context.Context, q RecallQuery, config RecallScoringConfig) (RecallTrace, error) {
	return s.recallWithOptions(ctx, q, recallPipelineOptions{scoringConfig: config})
}

func (s *Service) recallWithOptions(ctx context.Context, q RecallQuery, opts recallPipelineOptions) (RecallTrace, error) {
	if strings.TrimSpace(q.Peer) == "" {
		return RecallTrace{}, fmt.Errorf("goncho: peer is required")
	}
	if strings.TrimSpace(q.WorkspaceID) == "" {
		q.WorkspaceID = s.workspaceID
	}
	engine := newRecallPipelineEngine(s.retrieval(), opts)
	return engine.Run(ctx, q)
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

	card, err := getPeerCard(ctx, s.db, s.workspaceID, "", s.observer, peer)
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
	return s.lifecycle().CreateMessages(ctx, params)
}

func (s *Service) DeleteSession(ctx context.Context, sessionKey string) (SessionDeletionResult, error) {
	return s.lifecycle().DeleteSession(ctx, sessionKey)
}

func (s *Service) DeleteWorkspace(ctx context.Context) (WorkspaceDeletionResult, error) {
	return s.lifecycle().DeleteWorkspace(ctx)
}

func normalizeReasoningLevel(level string) string {
	level = strings.ToLower(strings.TrimSpace(level))
	if level == "" {
		return string(DialecticLevelLow)
	}
	return level
}

func normalizeScope(scope string) string {
	return normalizeMemoryScope(scope, "")
}

func normalizeMemoryScope(scope, profileID string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	switch scope {
	case MemoryScopeProfile, MemoryScopeWorkspace, MemoryScopeShared, MemoryScopeSession, MemoryScopeGlobal:
		return scope
	}
	if strings.TrimSpace(profileID) != "" {
		return MemoryScopeProfile
	}
	return MemoryScopeWorkspace
}

func profileScopeEvidence(profileID, scope string) *CrossChatRecallEvidence {
	return &CrossChatRecallEvidence{
		Decision: CrossChatDecisionAllowed,
		Scope:    scope,
		Reason:   fmt.Sprintf("profile_id %q resolved; no cross-profile widening unless an explicit shared/workspace scope is requested", profileID),
		UserID:   profileID,
	}
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
	b.WriteString("Representation for ")
	b.WriteString(peer)
	b.WriteString(":")
	if len(card) > 0 {
		b.WriteString("\n\nProfile facts:")
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
	conclusions := conclusionsFromSearchHits(hits)
	otherEvidence := make([]SearchHit, 0)
	for _, hit := range hits {
		if hit.Source == "conclusion" {
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

func makeIdempotencyKey(workspaceID, profileID, observer, peer, sessionKey, conclusion string) string {
	normalized := strings.ToLower(strings.TrimSpace(conclusion))
	seed := strings.Join([]string{
		workspaceID,
		profileID,
		observer,
		peer,
		strings.TrimSpace(sessionKey),
		normalized,
	}, "\x1f")
	return hashutil.SHA256HexString(seed)
}

type turnFallbackResult struct {
	Results       []SearchHit
	ScopeEvidence *CrossChatRecallEvidence
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
		ChildSessionIDs: cloneStrings(lineage.ChildSessionIDs),
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
	if n := textutil.WordCount(text); n > 0 {
		return n
	}
	return 1
}
