package goncho

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type retrievalModule struct {
	db           *sql.DB
	workspaceID  string
	observer     string
	recentLimit  int
	dreamEnabled bool
	sessions     SessionDirectory
}

func (s *Service) retrieval() retrievalModule {
	return retrievalModule{
		db:           s.db,
		workspaceID:  s.workspaceID,
		observer:     s.observer,
		recentLimit:  s.recentLimit,
		dreamEnabled: s.dreamEnabled,
		sessions:     s.sessions,
	}
}

func (r retrievalModule) Search(ctx context.Context, params SearchParams) (SearchResultSet, error) {
	peer := strings.TrimSpace(params.Peer)
	if peer == "" {
		return SearchResultSet{}, fmt.Errorf("goncho: peer is required")
	}
	profileID := strings.TrimSpace(params.ProfileID)
	memoryScope := normalizeMemoryScope(params.Scope, profileID)
	compiled, err := parseAndCompileSearchFilter(params.Filters, peer)
	if err != nil {
		return SearchResultSet{}, err
	}
	sources, denySources := mergeSearchSources(params.Sources, compiled.Sources)
	if denySources || compiled.DenyAll || filterValuesDenyAll(compiled.SessionIDs) {
		return SearchResultSet{
			WorkspaceID: r.workspaceID,
			ProfileID:   profileID,
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
		results, err = findConclusions(ctx, r.db, r.workspaceID, profileID, r.observer, peer, params.Query, params.SessionKey, memoryScope, compiled, limit)
		if err != nil {
			return SearchResultSet{}, err
		}
		if len(results) == 0 && strings.TrimSpace(params.Query) != "" {
			results, err = findConclusions(ctx, r.db, r.workspaceID, profileID, r.observer, peer, "", params.SessionKey, memoryScope, compiled, limit)
			if err != nil {
				return SearchResultSet{}, err
			}
		}
	}

	if len(results) == 0 {
		fallback, err := r.searchTurnFallback(ctx, params, compiled, limit)
		if err != nil {
			return SearchResultSet{}, err
		}
		results = fallback.Results
		scopeEvidence = fallback.ScopeEvidence
	}
	results = limitHitsByTokens(results, params.MaxTokens)

	if scopeEvidence == nil && profileID != "" {
		scopeEvidence = profileScopeEvidence(profileID, memoryScope)
	}
	return SearchResultSet{
		WorkspaceID:   r.workspaceID,
		ProfileID:     profileID,
		Peer:          peer,
		Query:         params.Query,
		ScopeEvidence: scopeEvidence,
		Results:       results,
	}, nil
}

func (r retrievalModule) searchTurnFallback(ctx context.Context, params SearchParams, compiled compiledSearchFilter, limit int) (turnFallbackResult, error) {
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
		if r.sessions == nil {
			evidence := DegradedCrossChatRecallEvidence(filter, "session directory unavailable; same-chat fallback scope used")
			fallback, err := findTurns(ctx, r.db, params.Query, params.SessionKey, compiled, limit)
			if err != nil {
				return turnFallbackResult{}, err
			}
			fallback = attachUnavailableLineageToTurnHits(fallback)
			return turnFallbackResult{Results: fallback, ScopeEvidence: &evidence}, nil
		}
		metas, err := r.sessions.ListMetadataByUserID(ctx, userID)
		if err != nil {
			return turnFallbackResult{}, err
		}
		evidenceMetas, err := r.crossChatEvidenceMetadata(ctx, userID, params.SessionKey, metas)
		if err != nil {
			return turnFallbackResult{}, err
		}
		evidence := ExplainCrossChatRecall(evidenceMetas, filter)
		if evidence.Decision != CrossChatDecisionAllowed {
			fallback, err := findTurns(ctx, r.db, params.Query, params.SessionKey, compiled, limit)
			if err != nil {
				return turnFallbackResult{}, err
			}
			fallback = attachUnavailableLineageToTurnHits(fallback)
			return turnFallbackResult{Results: fallback, ScopeEvidence: &evidence}, nil
		}
		hits, err := SearchMessages(ctx, r.db, metas, filter, limit)
		if errors.Is(err, ErrUserScopeDenied) {
			fallback, err := findTurns(ctx, r.db, params.Query, params.SessionKey, compiled, limit)
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
	results, err := findTurns(ctx, r.db, params.Query, params.SessionKey, compiled, limit)
	if err != nil {
		return turnFallbackResult{}, err
	}
	results = attachUnavailableLineageToTurnHits(results)
	return turnFallbackResult{Results: results}, nil
}

func (r retrievalModule) crossChatEvidenceMetadata(ctx context.Context, userID, currentKey string, metas []SessionMetadata) ([]SessionMetadata, error) {
	out := append([]SessionMetadata(nil), metas...)
	resolver, ok := r.sessions.(userBindingResolver)
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

func (r retrievalModule) Context(ctx context.Context, params ContextParams) (ContextResult, error) {
	peer := strings.TrimSpace(params.Peer)
	if peer == "" {
		return ContextResult{}, fmt.Errorf("goncho: peer is required")
	}
	profileID := strings.TrimSpace(params.ProfileID)
	sessionKey := strings.TrimSpace(params.SessionKey)
	query := effectiveContextQuery(params)
	tokenLimit := effectiveContextTokenLimit(params)
	unavailable := contextUnavailableEvidence(params, r.observer, peer)
	if includeDreamStatus(params) {
		dreamEvidence, err := r.dreamContextUnavailableEvidence(ctx, peer)
		if err != nil {
			return ContextResult{}, err
		}
		unavailable = append(unavailable, dreamEvidence...)
	}
	reviewEvidence, err := r.reviewContextUnavailableEvidence(ctx, peer, sessionKey)
	if err != nil {
		return ContextResult{}, err
	}
	unavailable = append(unavailable, reviewEvidence...)
	quarantineEvidence, err := promptInjectionQuarantineEvidenceForSession(ctx, r.db, sessionKey)
	if err != nil {
		return ContextResult{}, err
	}
	unavailable = append(unavailable, quarantineEvidence...)

	card, err := getPeerCard(ctx, r.db, r.workspaceID, profileID, r.observer, peer)
	if err != nil {
		return ContextResult{}, err
	}

	searchResult := SearchResultSet{
		WorkspaceID: r.workspaceID,
		ProfileID:   profileID,
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
		searchResult, err = r.Search(ctx, SearchParams{
			ProfileID:  profileID,
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
		turnCount, err := r.refreshSessionSummaries(ctx, sessionKey)
		if err != nil {
			return ContextResult{}, err
		}

		messageBudget := tokenLimit
		messageStartID := int64(0)
		if includeSummaryComponent(params) {
			var reason string
			summary, reason, err = selectSessionContextSummary(ctx, r.db, r.workspaceID, sessionKey, tokenLimit)
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
			recentMessages, err = recentTurnsByTokenBudget(ctx, r.db, sessionKey, messageStartID, messageBudget)
			if err != nil {
				return ContextResult{}, err
			}
		} else {
			recentMessages, err = recentTurnsAfter(ctx, r.db, sessionKey, messageStartID, r.recentLimit)
			if err != nil {
				return ContextResult{}, err
			}
		}
	}

	return ContextResult{
		WorkspaceID:    r.workspaceID,
		ProfileID:      profileID,
		Peer:           peer,
		ObserverPeerID: r.observer,
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

func (r retrievalModule) refreshSessionSummaries(ctx context.Context, sessionKey string) (int, error) {
	count, err := countReadySessionTurns(ctx, r.db, sessionKey)
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
		if err := r.refreshSessionSummarySlot(ctx, sessionKey, cfg.summaryType, cfg.cadence, count); err != nil {
			return 0, err
		}
	}
	return count, nil
}

func (r retrievalModule) refreshSessionSummarySlot(ctx context.Context, sessionKey, summaryType string, cadence, turnCount int) error {
	if cadence <= 0 || turnCount < cadence {
		return nil
	}
	coveredCount := (turnCount / cadence) * cadence
	messageID, err := readySessionTurnIDAtPosition(ctx, r.db, sessionKey, coveredCount)
	if err != nil {
		return err
	}
	if messageID == 0 {
		return nil
	}

	existing, err := getSessionSummary(ctx, r.db, r.workspaceID, sessionKey, summaryType)
	if err != nil {
		return err
	}
	if existing != nil && existing.MessageID >= messageID {
		return nil
	}

	content := deterministicSummaryContent(sessionKey, summaryType, coveredCount, messageID)
	return upsertSessionSummary(ctx, r.db, sessionSummaryRow{
		WorkspaceID: r.workspaceID,
		SessionKey:  sessionKey,
		SummaryType: summaryType,
		Content:     content,
		MessageID:   messageID,
		TokenCount:  approxTokens(content),
	})
}

func (r retrievalModule) dreamContextUnavailableEvidence(ctx context.Context, peer string) ([]ContextUnavailableEvidence, error) {
	if !r.dreamEnabled {
		return []ContextUnavailableEvidence{{
			Field:      "dream",
			Capability: "dream_disabled",
			Reason:     "dreaming is disabled; no background dream reasoning is active",
		}}, nil
	}
	present, err := sqliteTableExists(ctx, r.db, "goncho_dreams")
	if err != nil {
		return nil, err
	}
	if !present {
		return []ContextUnavailableEvidence{{
			Field:      "dream",
			Capability: "dream_unavailable",
			Reason:     "goncho_dreams scheduler table is unavailable; no background dream reasoning is active for " + peer,
		}}, nil
	}
	return nil, nil
}

func (r retrievalModule) reviewContextUnavailableEvidence(ctx context.Context, peer, sessionKey string) ([]ContextUnavailableEvidence, error) {
	items, err := ListReviewItems(ctx, r.db, ReviewQuery{WorkspaceID: r.workspaceID, PeerID: peer, Status: ReviewStatusOpen})
	if err != nil {
		return nil, err
	}
	return reviewRequiredUnavailableEvidence(reviewItemsForContextSession(items.Items, sessionKey)), nil
}
