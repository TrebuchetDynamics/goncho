package goncho

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// VectorStore is the optional host-owned semantic retrieval seam. Implementations
// may use local embeddings, an in-process ANN index, or deterministic fakes in
// tests; Goncho treats returned hits as semantic evidence and fuses them with
// lexical/graph recall through the existing RRF scorer.
type VectorStore interface {
	Search(ctx context.Context, query VectorSearchQuery) ([]VectorSearchHit, error)
}

// VectorSearchQuery is the host-neutral request passed to an optional
// VectorStore during recall candidate generation.
type VectorSearchQuery struct {
	WorkspaceID string   `json:"workspace_id"`
	ProfileID   string   `json:"profile_id,omitempty"`
	Peer        string   `json:"peer"`
	Query       string   `json:"query"`
	SessionKey  string   `json:"session_key,omitempty"`
	ScopeID     string   `json:"scope_id,omitempty"`
	Sources     []string `json:"sources,omitempty"`
	Limit       int      `json:"limit,omitempty"`
}

// VectorSearchHit is one semantic hit returned by a VectorStore.
type VectorSearchHit struct {
	MemoryID   string            `json:"memory_id"`
	SourceType string            `json:"source_type,omitempty"`
	Content    string            `json:"content"`
	SessionID  string            `json:"session_id,omitempty"`
	AgentID    string            `json:"agent_id,omitempty"`
	ScopeID    string            `json:"scope_id,omitempty"`
	CreatedAt  time.Time         `json:"created_at,omitempty"`
	Importance float64           `json:"importance,omitempty"`
	Score      float64           `json:"score"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

func vectorSearchQueryFromRecall(q RecallQuery, workspaceID, profileID, peer, scopeID string) VectorSearchQuery {
	return VectorSearchQuery{
		WorkspaceID: workspaceID,
		ProfileID:   profileID,
		Peer:        peer,
		Query:       q.Query,
		SessionKey:  q.SessionKey,
		ScopeID:     scopeID,
		Sources:     cloneStrings(q.Sources),
		Limit:       recallCandidateSearchLimit(q.Limit),
	}
}

func recallCandidateFromVectorHit(hit VectorSearchHit, observer, scopeID string) RecallCandidate {
	memoryID := strings.TrimSpace(hit.MemoryID)
	if memoryID == "" {
		memoryID = semanticMemoryID(hit)
	}
	sourceType := strings.TrimSpace(hit.SourceType)
	if sourceType == "" {
		sourceType = "vector"
	}
	agentID := strings.TrimSpace(hit.AgentID)
	if agentID == "" {
		agentID = observer
	}
	candidateScope := strings.TrimSpace(hit.ScopeID)
	if candidateScope == "" {
		candidateScope = scopeID
	}
	return RecallCandidate{
		MemoryID:   memoryID,
		SourceType: sourceType,
		Content:    hit.Content,
		SessionID:  hit.SessionID,
		AgentID:    agentID,
		ScopeID:    candidateScope,
		CreatedAt:  hit.CreatedAt,
		Importance: hit.Importance,
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

func semanticMemoryID(hit VectorSearchHit) string {
	seed := strings.TrimSpace(hit.SourceType) + "\x00" + strings.TrimSpace(hit.SessionID) + "\x00" + strings.TrimSpace(hit.Content)
	sum := sha256.Sum256([]byte(seed))
	return "vector:" + hex.EncodeToString(sum[:8])
}

func (r retrievalModule) mergeVectorRecall(ctx context.Context, q RecallQuery, workspaceID, profileID, peer, scopeID string, base []RecallCandidate) ([]RecallCandidate, error) {
	if r.vectorStore == nil || strings.TrimSpace(q.Query) == "" {
		return base, nil
	}
	var hits []VectorSearchHit
	query := vectorSearchQueryFromRecall(q, workspaceID, profileID, peer, scopeID)
	if maxPayload := r.providers.MaxPayloadBytes(string(ProviderKindEmbedding)); maxPayload > 0 && len(query.Query) > maxPayload {
		r.recallWarnings.append(RecallWarning{Code: RecallWarningSemanticUnavailable, Stage: RecallStageGenerate, Severity: RecallWarningDegraded, Message: "optional semantic provider skipped because query exceeds configured provider payload limit; lexical/graph recall fallback remained active", Evidence: map[string]string{"provider": string(ProviderKindEmbedding), "error": "max_payload_exceeded", "max_payload_bytes": fmt.Sprintf("%d", maxPayload)}})
		return base, nil
	}
	err := r.providers.Execute(ctx, string(ProviderKindEmbedding), func(providerCtx context.Context) error {
		var searchErr error
		hits, searchErr = r.vectorStore.Search(providerCtx, query)
		return searchErr
	})
	if err != nil {
		r.recallWarnings.append(RecallWarning{Code: RecallWarningSemanticUnavailable, Stage: RecallStageGenerate, Severity: RecallWarningDegraded, Message: "optional semantic provider unavailable; lexical/graph recall fallback remained active", Evidence: map[string]string{"provider": string(ProviderKindEmbedding), "error": err.Error()}})
		return base, nil
	}
	out := append([]RecallCandidate(nil), base...)
	indexByID := make(map[string]int, len(out)+len(hits))
	for i, candidate := range out {
		if strings.TrimSpace(candidate.MemoryID) != "" {
			indexByID[candidate.MemoryID] = i
		}
	}
	for _, hit := range hits {
		if strings.TrimSpace(hit.Content) == "" || !vectorSourceAllowed(q.Sources, hit.SourceType) {
			continue
		}
		candidate := recallCandidateFromVectorHit(hit, r.observer, scopeID)
		if idx, ok := indexByID[candidate.MemoryID]; ok {
			out[idx] = mergeRecallCandidateEvidence(out[idx], candidate)
			continue
		}
		indexByID[candidate.MemoryID] = len(out)
		out = append(out, candidate)
	}
	return out, nil
}

func mergeRecallCandidateEvidence(existing, incoming RecallCandidate) RecallCandidate {
	if existing.Content == "" {
		existing.Content = incoming.Content
	}
	if existing.SourceType == "" {
		existing.SourceType = incoming.SourceType
	}
	if existing.SessionID == "" {
		existing.SessionID = incoming.SessionID
	}
	if existing.AgentID == "" {
		existing.AgentID = incoming.AgentID
	}
	if existing.ScopeID == "" {
		existing.ScopeID = incoming.ScopeID
	}
	if existing.CreatedAt.IsZero() {
		existing.CreatedAt = incoming.CreatedAt
	}
	if existing.Importance == 0 {
		existing.Importance = incoming.Importance
	}
	for _, evidence := range incoming.Provenance {
		if !recallCandidateHasEvidence(existing, evidence.Kind, evidence.ID) {
			existing.Provenance = append(existing.Provenance, evidence)
		}
	}
	return existing
}

func vectorSourceAllowed(sources []string, sourceType string) bool {
	if len(sources) == 0 || filterHasWildcard(sources) {
		return true
	}
	if strings.TrimSpace(sourceType) == "" {
		return true
	}
	for _, source := range sources {
		if strings.EqualFold(strings.TrimSpace(source), strings.TrimSpace(sourceType)) {
			return true
		}
	}
	return false
}

func cloneVectorMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
