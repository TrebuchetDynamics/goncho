package goncho

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type retrievalModule struct {
	db             *sql.DB
	workspaceID    string
	observer       string
	recentLimit    int
	dreamEnabled   bool
	sessions       SessionDirectory
	vectorStore    VectorStore
	searchReranker SearchReranker
	providers      *ProviderHealthRegistry
	recallWarnings *recallWarningBuffer
}

func (s *Service) retrieval() retrievalModule {
	return retrievalModule{
		db:             s.db,
		workspaceID:    s.workspaceID,
		observer:       s.observer,
		recentLimit:    s.recentLimit,
		dreamEnabled:   s.dreamEnabled,
		sessions:       s.sessions,
		vectorStore:    s.vectorStore,
		searchReranker: s.searchReranker,
		providers:      s.providerRegistry,
		recallWarnings: &recallWarningBuffer{},
	}
}

func (r retrievalModule) RecallWarnings() []RecallWarning {
	return r.recallWarnings.list()
}

func (r retrievalModule) Generate(ctx context.Context, q RecallQuery) ([]RecallCandidate, error) {
	peer := strings.TrimSpace(q.Peer)
	if peer == "" {
		return nil, fmt.Errorf("goncho: peer is required")
	}
	if !recallSourcesAllowConclusions(q.Sources) {
		return []RecallCandidate{}, nil
	}
	workspaceID := strings.TrimSpace(q.WorkspaceID)
	if workspaceID == "" {
		workspaceID = r.workspaceID
	}
	memoryScope := normalizeMemoryScope(q.ScopeID, "")
	hits, err := findConclusions(ctx, r.db, workspaceID, "", r.observer, peer, q.Query, q.SessionKey, memoryScope, compiledSearchFilter{}, recallCandidateSearchLimit(q.Limit))
	if err != nil {
		return nil, err
	}
	out := make([]RecallCandidate, 0, len(hits))
	for _, hit := range hits {
		out = append(out, recallCandidateFromSearchHit(q, hit, r.observer, memoryScope))
	}
	out, err = r.mergeVectorRecall(ctx, q, workspaceID, "", peer, memoryScope, out)
	if err != nil {
		return nil, err
	}
	out, err = r.expandAnnotationGraphRecall(ctx, q, workspaceID, peer, memoryScope, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func recallCandidateSearchLimit(selectionLimit int) int {
	limit := selectionLimit * 5
	if selectionLimit <= 0 {
		limit = 25
	}
	if limit < 10 {
		limit = 10
	}
	return normalizeSearchLimit(limit)
}

func recallSourcesAllowConclusions(sources []string) bool {
	if len(sources) == 0 || filterHasWildcard(sources) {
		return true
	}
	for _, source := range sources {
		if strings.EqualFold(strings.TrimSpace(source), "conclusion") {
			return true
		}
	}
	return false
}

func recallCandidateFromSearchHit(q RecallQuery, hit SearchHit, observer, scopeID string) RecallCandidate {
	provenance := append([]EvidenceItem(nil), hit.Provenance...)
	keywordScore := roundRecallFloat(keywordRecallScore(hit.Content, q.Query))
	expansion := expandSearchQuery(q.Query)
	expandedKeywordScore := keywordScore
	if expansion.Applied() {
		expandedKeywordScore = roundRecallFloat(keywordRecallScore(hit.Content, expansion.Expanded))
	}
	if keywordScore > 0 {
		provenance = append(provenance, EvidenceItem{
			Kind:   "keyword",
			Source: "goncho_conclusions",
			ID:     strconv.FormatInt(hit.ID, 10),
			Score:  keywordScore,
			Note:   "matched conclusion content",
		})
	}
	if expansion.Applied() && expandedKeywordScore > keywordScore {
		if !evidenceListHas(provenance, "query_expansion", strings.ToLower(strings.TrimSpace(expansion.Original))) {
			provenance = append(provenance, queryExpansionEvidence(expansion))
		}
		provenance = append(provenance, EvidenceItem{
			Kind:   "keyword",
			Source: "goncho_query_expansion",
			ID:     "expanded:" + strings.ToLower(strings.TrimSpace(expansion.Original)),
			Score:  expandedKeywordScore,
			Note:   "matched expanded query terms",
			Metadata: map[string]string{
				"original_query": strings.TrimSpace(expansion.Original),
				"expanded_terms": strings.Join(expansion.Terms, ","),
			},
		})
	}
	for _, fact := range hit.factAnnotations {
		if strings.TrimSpace(fact.Value) == "" {
			continue
		}
		provenance = append(provenance, annotationFactEvidence(q.Query, fact))
	}
	return RecallCandidate{
		MemoryID:   strconv.FormatInt(hit.ID, 10),
		SourceType: hit.Source,
		Content:    hit.Content,
		SessionID:  hit.SessionKey,
		AgentID:    observer,
		ScopeID:    scopeID,
		Provenance: provenance,
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

	results, err = r.mergeVectorSearch(ctx, params, profileID, peer, memoryScope, results, limit)
	if err != nil {
		return SearchResultSet{}, err
	}
	if len(results) == 0 {
		fallback, err := r.searchTurnFallback(ctx, params, compiled, limit)
		if err != nil {
			return SearchResultSet{}, err
		}
		results = fallback.Results
		scopeEvidence = fallback.ScopeEvidence
	}
	results = applySearchReranker(ctx, r.searchReranker, params.Query, results)
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

func (r retrievalModule) mergeVectorSearch(ctx context.Context, params SearchParams, profileID, peer, scopeID string, base []SearchHit, limit int) ([]SearchHit, error) {
	if r.vectorStore == nil || strings.TrimSpace(params.Query) == "" || !vectorSourceAllowed(params.Sources, "conclusion") {
		return base, nil
	}
	query := VectorSearchQuery{
		WorkspaceID: r.workspaceID,
		ProfileID:   profileID,
		Peer:        peer,
		Query:       params.Query,
		SessionKey:  params.SessionKey,
		ScopeID:     scopeID,
		Sources:     append([]string(nil), params.Sources...),
		Limit:       recallCandidateSearchLimit(limit),
	}
	if maxPayload := r.providers.MaxPayloadBytes(string(ProviderKindEmbedding)); maxPayload > 0 && len(query.Query) > maxPayload {
		return base, nil
	}
	var hits []VectorSearchHit
	err := r.providers.Execute(ctx, string(ProviderKindEmbedding), func(providerCtx context.Context) error {
		var searchErr error
		hits, searchErr = r.vectorStore.Search(providerCtx, query)
		return searchErr
	})
	if err != nil {
		return base, nil
	}
	out := append([]SearchHit(nil), base...)
	index := map[string]int{}
	for i, hit := range out {
		index[searchHitVectorMergeKey(hit)] = i
	}
	sort.SliceStable(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})
	for _, hit := range hits {
		if strings.TrimSpace(hit.Content) == "" || !vectorSourceAllowed(params.Sources, hit.SourceType) {
			continue
		}
		searchHit := searchHitFromVectorHit(hit)
		key := searchHitVectorMergeKey(searchHit)
		if idx, ok := index[key]; ok {
			if len(searchHit.Provenance) > 0 && !evidenceListHas(out[idx].Provenance, "semantic", searchHit.Provenance[0].ID) {
				out[idx].Provenance = append(out[idx].Provenance, searchHit.Provenance...)
			}
			continue
		}
		index[key] = len(out)
		out = append(out, searchHit)
	}
	return trimSearchHits(out, limit), nil
}

func trimSearchHits(hits []SearchHit, limit int) []SearchHit {
	if limit > 0 && len(hits) > limit {
		return append([]SearchHit(nil), hits[:limit]...)
	}
	return hits
}

func searchHitFromVectorHit(hit VectorSearchHit) SearchHit {
	id, _ := strconv.ParseInt(strings.TrimSpace(hit.MemoryID), 10, 64)
	source := strings.TrimSpace(hit.SourceType)
	if source == "" {
		source = "vector"
	}
	memoryID := strings.TrimSpace(hit.MemoryID)
	if memoryID == "" {
		memoryID = semanticMemoryID(hit)
	}
	return SearchHit{
		ID:         id,
		Source:     source,
		Content:    hit.Content,
		SessionKey: hit.SessionID,
		Provenance: []EvidenceItem{{
			Kind:     "semantic",
			Source:   "vector_store",
			ID:       memoryID,
			Score:    clampRecall(hit.Score),
			Note:     "matched optional vector store",
			Metadata: cloneVectorMetadata(hit.Metadata),
		}},
	}
}

func searchHitVectorMergeKey(hit SearchHit) string {
	if hit.ID > 0 {
		return "id:" + strconv.FormatInt(hit.ID, 10)
	}
	return "content:" + strings.TrimSpace(hit.Content)
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
	return reviewRequiredUnavailableEvidence(reviewItemsForContextSession(items.Items, sessionKey), sessionKey), nil
}
