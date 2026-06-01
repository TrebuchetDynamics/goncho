package goncho

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/contexttokens"
	"github.com/TrebuchetDynamics/goncho/service/internal/idutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/recallscore"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sourcefilter"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
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
	queryAliases   map[string][]string
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
		queryAliases:   cloneQueryAliases(s.queryAliases),
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
	sourcePlan := recallSourcePlan(q.Sources)
	if !sourcePlan.IncludeConclusions && !sourcePlan.IncludeVector {
		return []RecallCandidate{}, nil
	}
	workspaceID := strings.TrimSpace(q.WorkspaceID)
	if workspaceID == "" {
		workspaceID = r.workspaceID
	}
	memoryScope := normalizeMemoryScope(q.ScopeID, "")
	var out []RecallCandidate
	if sourcePlan.IncludeConclusions {
		hits, err := findConclusions(ctx, r.db, workspaceID, "", r.observer, peer, q.Query, q.SessionKey, memoryScope, compiledSearchFilter{}, recallCandidateSearchLimit(q.Limit), r.queryAliases)
		if err != nil {
			return nil, err
		}
		out = sliceutil.Map(hits, func(hit SearchHit) RecallCandidate {
			return recallCandidateFromSearchHit(q, hit, r.observer, memoryScope, r.queryAliases)
		})
	}
	if sourcePlan.IncludeVector {
		var err error
		out, err = r.mergeVectorRecall(ctx, q, workspaceID, "", peer, memoryScope, out)
		if err != nil {
			return nil, err
		}
	}
	out, err := r.expandAnnotationGraphRecall(ctx, q, workspaceID, peer, memoryScope, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func recallCandidateSearchLimit(selectionLimit int) int {
	limit := limitutil.Default(selectionLimit*5, 25)
	if limit < 10 {
		limit = 10
	}
	return normalizeSearchLimit(limit)
}

type recallCandidateSourcePlan struct {
	IncludeConclusions bool
	IncludeVector      bool
}

func recallSourcePlan(sources []string) recallCandidateSourcePlan {
	includeConclusions := recallSourcesAllowConclusions(sources)
	return recallCandidateSourcePlan{
		IncludeConclusions: includeConclusions,
		IncludeVector:      includeConclusions || sourcefilter.Allows(sources, "vector", true),
	}
}

func recallSourcesAllowConclusions(sources []string) bool {
	return sourcefilter.Allows(sources, "conclusion", false)
}

func recallCandidateFromSearchHit(q RecallQuery, hit SearchHit, observer, scopeID string, queryAliases map[string][]string) RecallCandidate {
	provenance := sliceutil.Clone(hit.Provenance)
	keywordScore := roundRecallFloat(recallscore.Keyword(hit.Content, q.Query))
	expansion := expandSearchQueryWithAliases(q.Query, queryAliases)
	expandedKeywordScore := keywordScore
	if expansion.Applied() {
		expandedKeywordScore = roundRecallFloat(recallscore.Keyword(hit.Content, expansion.Expanded))
	}
	if keywordScore > 0 {
		provenance = append(provenance, EvidenceItem{
			Kind:   "keyword",
			Source: "goncho_conclusions",
			ID:     idutil.Decimal(hit.ID),
			Score:  keywordScore,
			Note:   "matched conclusion content",
		})
	}
	if expansion.Applied() && expandedKeywordScore > keywordScore {
		if !evidenceListHas(provenance, "query_expansion", textutil.LowerTrimmed(expansion.Original)) {
			provenance = append(provenance, queryExpansionEvidence(expansion))
		}
		provenance = append(provenance, EvidenceItem{
			Kind:     "keyword",
			Source:   "goncho_query_expansion",
			ID:       "expanded:" + textutil.LowerTrimmed(expansion.Original),
			Score:    expandedKeywordScore,
			Note:     "matched expanded query terms",
			Metadata: queryExpansionEvidenceMetadata(expansion),
		})
	}
	for _, fact := range hit.factAnnotations {
		if strings.TrimSpace(fact.Value) == "" {
			continue
		}
		provenance = append(provenance, annotationFactEvidence(q.Query, fact))
	}
	return RecallCandidate{
		MemoryID:   idutil.Decimal(hit.ID),
		SourceType: hit.Source,
		Content:    hit.Content,
		SessionID:  hit.SessionKey,
		AgentID:    observer,
		ScopeID:    scopeID,
		CreatedAt:  hit.updatedAt,
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
	if denySources || compiled.DenyAll {
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
		results, err = findConclusions(ctx, r.db, r.workspaceID, profileID, r.observer, peer, params.Query, params.SessionKey, memoryScope, compiled, limit, r.queryAliases)
		if err != nil {
			return SearchResultSet{}, err
		}
		if len(results) == 0 && strings.TrimSpace(params.Query) != "" {
			results, err = findConclusions(ctx, r.db, r.workspaceID, profileID, r.observer, peer, "", params.SessionKey, memoryScope, compiled, limit, r.queryAliases)
			if err != nil {
				return SearchResultSet{}, err
			}
		}
	}

	results, err = r.mergeVectorSearch(ctx, params, profileID, peer, memoryScope, compiled.Sources, results, limit)
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
	results = finalizeSearchResults(ctx, r.searchReranker, params.Query, results, limit, params.MaxTokens)

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

func (r retrievalModule) mergeVectorSearch(ctx context.Context, params SearchParams, profileID, peer, scopeID string, effectiveSources []string, base []SearchHit, limit int) ([]SearchHit, error) {
	vectorSources := vectorSearchSources(effectiveSources)
	if r.vectorStore == nil || strings.TrimSpace(params.Query) == "" || !vectorSearchLaneAllowed(vectorSources) {
		return base, nil
	}
	query := VectorSearchQuery{
		WorkspaceID: r.workspaceID,
		ProfileID:   profileID,
		Peer:        peer,
		Query:       params.Query,
		SessionKey:  params.SessionKey,
		ScopeID:     scopeID,
		Sources:     vectorSources,
		Limit:       recallCandidateSearchLimit(limit),
	}
	if decision := vectorProviderPayloadDecision(query.Query, r.providers.MaxPayloadBytes(string(ProviderKindEmbedding))); decision.Skip {
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
	out := sliceutil.Clone(base)
	index := sliceutil.IndexBy(out, func(hit SearchHit) (string, bool) {
		return searchHitVectorMergeKey(hit), true
	})
	for _, hit := range vectorHitsByScoreDesc(hits) {
		decision := vectorSearchMergeDecisionFor(hit, vectorSources, index)
		if decision.Skip {
			continue
		}
		searchHit := searchHitFromVectorHit(hit)
		if decision.Merge {
			if len(searchHit.Provenance) > 0 && !evidenceListHas(out[decision.Index].Provenance, "semantic", searchHit.Provenance[0].ID) {
				out[decision.Index].Provenance = append(out[decision.Index].Provenance, searchHit.Provenance...)
			}
			continue
		}
		index[decision.Key] = len(out)
		out = append(out, searchHit)
	}
	return out, nil
}

// vectorSearchSources is the explicit handoff from Honcho source filters to
// the optional semantic lane. Callers must pass already-merged SearchParams
// sources and compiled source filters; using only raw params.Sources can leak
// conclusion vector hits through a filter such as {"source":"turn"}.
func vectorSearchSources(effectiveSources []string) []string {
	return sliceutil.Clone(effectiveSources)
}

func vectorSearchLaneAllowed(sources []string) bool {
	return vectorSourceAllowed(sources, "conclusion") || vectorSourceAllowed(sources, "vector")
}

type vectorSearchMergeDecision struct {
	Skip  bool
	Merge bool
	Key   string
	Index int
}

func vectorSearchMergeDecisionFor(hit VectorSearchHit, vectorSources []string, index map[string]int) vectorSearchMergeDecision {
	if strings.TrimSpace(hit.Content) == "" || !vectorHitAllowedBySources(vectorSources, hit) {
		return vectorSearchMergeDecision{Skip: true}
	}
	searchHit := searchHitFromVectorHit(hit)
	key := searchHitVectorMergeKey(searchHit)
	if idx, ok := index[key]; ok {
		return vectorSearchMergeDecision{Merge: true, Key: key, Index: idx}
	}
	return vectorSearchMergeDecision{Key: key}
}

func finalizeSearchResults(ctx context.Context, reranker SearchReranker, query string, hits []SearchHit, limit, maxTokens int) []SearchHit {
	// Keep candidate-producing lanes separate from final top-K truncation so the
	// optional reranker can see semantic candidates added after lexical search.
	hits = applySearchReranker(ctx, reranker, query, hits)
	hits = trimSearchHits(hits, limit)
	return limitHitsByTokens(hits, maxTokens)
}

func trimSearchHits(hits []SearchHit, limit int) []SearchHit {
	return sliceutil.LimitClone(hits, limit)
}

func searchHitFromVectorHit(hit VectorSearchHit) SearchHit {
	memoryID := vectorHitMemoryID(hit)
	id, _ := vectorHitNumericMemoryID(memoryID)
	return SearchHit{
		ID:         id,
		Source:     vectorHitSourceType(hit),
		Content:    hit.Content,
		SessionKey: hit.SessionID,
		Provenance: []EvidenceItem{semanticVectorEvidence(hit, memoryID)},
	}
}

func vectorHitNumericMemoryID(memoryID string) (int64, bool) {
	if id, err := idutil.ParseDecimal(memoryID); err == nil {
		return id, true
	}
	if id, err := idutil.ParsePrefixed(memoryID, "id:"); err == nil {
		return id, true
	}
	return 0, false
}

func searchHitVectorMergeKey(hit SearchHit) string {
	if hit.ID > 0 {
		return idutil.Prefixed("id:", hit.ID)
	}
	if semanticID, ok := searchHitSemanticEvidenceID(hit); ok {
		return "semantic:" + semanticID
	}
	return "content:" + strings.TrimSpace(hit.Content)
}

func searchHitSemanticEvidenceID(hit SearchHit) (string, bool) {
	for _, evidence := range hit.Provenance {
		if evidence.Kind != "semantic" {
			continue
		}
		id := strings.TrimSpace(evidence.ID)
		if id != "" {
			return id, true
		}
	}
	return "", false
}

func (r retrievalModule) searchTurnFallback(ctx context.Context, params SearchParams, compiled compiledSearchFilter, limit int) (turnFallbackResult, error) {
	if textutil.EqualFoldTrimmed(params.Scope, "user") {
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
	out := sliceutil.Clone(metas)
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
	tokenLimit := contexttokens.EffectiveContextLimit(params.Tokens, params.MaxTokens)
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
			MaxTokens:  contexttokens.EffectiveSearchLimit(params.Tokens, params.MaxTokens),
			SessionKey: sessionKey,
			Scope:      scope,
			Sources:    params.Sources,
		})
		if err != nil {
			return ContextResult{}, err
		}
	}

	var summary *SessionSummary
	conclusions := conclusionsFromSearchHits(searchResult.Results)

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
					_, messageBudget = contexttokens.SplitSummaryMessageBudget(tokenLimit)
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

	result := ContextResult{
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
	}
	result.InclusionReasons = contextInclusionReasons(result, tokenLimit)
	return result, nil
}

func contextInclusionReasons(result ContextResult, tokenBudget int) []ContextInclusionReason {
	reasons := []ContextInclusionReason{
		{Section: "peer_card", Included: len(result.PeerCard) > 0, Reason: contextInclusionReason(len(result.PeerCard) > 0, "peer card facts available", "no peer card facts stored"), Count: len(result.PeerCard), Source: "goncho_peer_cards"},
		{Section: "summary", Included: result.Summary != nil, Reason: contextInclusionReason(result.Summary != nil, "session summary selected before recent messages", "no eligible session summary"), TokenBudget: tokenBudget, Source: "goncho_session_summaries"},
		{Section: "conclusions", Included: len(result.Conclusions) > 0, Reason: contextInclusionReason(len(result.Conclusions) > 0, "search/recall evidence matched context query", "no matching conclusions selected"), Count: len(result.Conclusions), Source: "goncho_conclusions"},
		{Section: "recent_messages", Included: len(result.RecentMessages) > 0, Reason: contextInclusionReason(len(result.RecentMessages) > 0, "recent session turns fit token budget", "no recent session turns selected"), TokenBudget: tokenBudget, Count: len(result.RecentMessages), Source: "turns"},
	}
	if len(result.Unavailable) > 0 {
		reasons = append(reasons, ContextInclusionReason{Section: "warnings", Included: true, Reason: "context unavailable evidence reported", Count: len(result.Unavailable), Source: "context_unavailable"})
	}
	return reasons
}

func contextInclusionReason(condition bool, yes, no string) string {
	if condition {
		return yes
	}
	return no
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
		TokenCount:  textutil.ApproxTokens(content),
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
