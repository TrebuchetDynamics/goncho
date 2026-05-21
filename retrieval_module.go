package goncho

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type retrievalModule struct {
	db              *sql.DB
	workspaceID     string
	observer        string
	recentLimit     int
	peerCardEnabled bool
	dreamEnabled    bool
	sessions        SessionDirectory
}

func (s *Service) retrieval() retrievalModule {
	return retrievalModule{
		db:              s.db,
		workspaceID:     s.workspaceID,
		observer:        s.observer,
		recentLimit:     s.recentLimit,
		peerCardEnabled: s.peerCardEnabled,
		dreamEnabled:    s.dreamEnabled,
		sessions:        s.sessions,
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
