package goncho

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/hashutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/maputil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sourcefilter"
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
		Sources:     sliceutil.Clone(q.Sources),
		Limit:       recallCandidateSearchLimit(q.Limit),
	}
}

func recallCandidateFromVectorHit(hit VectorSearchHit, observer, scopeID string) RecallCandidate {
	memoryID := vectorHitMemoryID(hit)
	sourceType := vectorHitSourceType(hit)
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
		Provenance: []EvidenceItem{semanticVectorEvidence(hit, memoryID)},
	}
}

func vectorHitMemoryID(hit VectorSearchHit) string {
	memoryID := strings.TrimSpace(hit.MemoryID)
	if memoryID == "" {
		return semanticMemoryID(hit)
	}
	return memoryID
}

func vectorHitSourceType(hit VectorSearchHit) string {
	sourceType := strings.TrimSpace(hit.SourceType)
	if sourceType == "" {
		return "vector"
	}
	return sourceType
}

func semanticVectorEvidence(hit VectorSearchHit, memoryID string) EvidenceItem {
	return EvidenceItem{
		Kind:     "semantic",
		Source:   "vector_store",
		ID:       memoryID,
		Score:    clampRecall(hit.Score),
		Note:     "matched optional vector store",
		Metadata: maputil.CloneStringStringNilIfEmpty(hit.Metadata),
	}
}

func semanticMemoryID(hit VectorSearchHit) string {
	seed := strings.TrimSpace(hit.SourceType) + "\x00" + strings.TrimSpace(hit.SessionID) + "\x00" + strings.TrimSpace(hit.Content)
	return "vector:" + hashutil.SHA256HexStringPrefix(seed, 8)
}

func mergeVectorRecallCandidates(base []RecallCandidate, hits []VectorSearchHit, observer, scopeID string, sources []string) []RecallCandidate {
	out := sliceutil.Clone(base)
	indexByID := sliceutil.IndexBy(out, recallCandidateStableMemoryID)
	for _, hit := range vectorHitsByScoreDesc(hits) {
		if strings.TrimSpace(hit.Content) == "" || !vectorHitAllowedBySources(sources, hit) {
			continue
		}
		candidate := recallCandidateFromVectorHit(hit, observer, scopeID)
		memoryID, stable := recallCandidateStableMemoryID(candidate)
		if !stable {
			out = append(out, candidate)
			continue
		}
		if idx, ok := indexByID[memoryID]; ok {
			out[idx] = mergeRecallCandidateEvidence(out[idx], candidate)
			continue
		}
		indexByID[memoryID] = len(out)
		out = append(out, candidate)
	}
	return out
}

func (r retrievalModule) mergeVectorRecall(ctx context.Context, q RecallQuery, workspaceID, profileID, peer, scopeID string, base []RecallCandidate) ([]RecallCandidate, error) {
	if r.vectorStore == nil || strings.TrimSpace(q.Query) == "" {
		return base, nil
	}
	var hits []VectorSearchHit
	query := vectorSearchQueryFromRecall(q, workspaceID, profileID, peer, scopeID)
	if decision := vectorProviderPayloadDecision(query.Query, r.providers.MaxPayloadBytes(string(ProviderKindEmbedding))); decision.Skip {
		r.recallWarnings.append(decision.RecallWarning())
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
	return mergeVectorRecallCandidates(base, hits, r.observer, scopeID, q.Sources), nil
}

func mergeRecallCandidateEvidence(existing, incoming RecallCandidate) RecallCandidate {
	existing = mergeRecallCandidateFields(existing, incoming)
	indexByEvidence := make(map[string]int, len(existing.Provenance)+len(incoming.Provenance))
	for i, evidence := range existing.Provenance {
		indexByEvidence[recallCandidateEvidenceKey(evidence)] = i
	}
	for _, evidence := range incoming.Provenance {
		key := recallCandidateEvidenceKey(evidence)
		if idx, ok := indexByEvidence[key]; ok {
			existing.Provenance[idx] = mergeRecallEvidenceItem(existing.Provenance[idx], evidence)
			continue
		}
		indexByEvidence[key] = len(existing.Provenance)
		existing.Provenance = append(existing.Provenance, evidence)
	}
	return existing
}

func mergeRecallCandidateFields(existing, incoming RecallCandidate) RecallCandidate {
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
	if incoming.Importance > existing.Importance {
		existing.Importance = incoming.Importance
	}
	return existing
}

func mergeRecallEvidenceItem(existing, incoming EvidenceItem) EvidenceItem {
	merged := existing
	preferredMetadata := existing.Metadata
	fallbackMetadata := incoming.Metadata
	if incoming.Score > existing.Score {
		merged = incoming
		preferredMetadata = incoming.Metadata
		fallbackMetadata = existing.Metadata
	}
	merged.Metadata = mergeEvidenceMetadata(preferredMetadata, fallbackMetadata)
	return merged
}

func mergeEvidenceMetadata(preferred, fallback map[string]string) map[string]string {
	if len(preferred) == 0 && len(fallback) == 0 {
		return nil
	}
	merged := make(map[string]string, len(preferred)+len(fallback))
	for key, value := range fallback {
		merged[key] = value
	}
	for key, value := range preferred {
		merged[key] = value
	}
	return merged
}

type vectorProviderDecision struct {
	Skip            bool
	MaxPayloadBytes int
}

func vectorProviderPayloadDecision(query string, maxPayloadBytes int) vectorProviderDecision {
	return vectorProviderDecision{
		Skip:            maxPayloadBytes > 0 && len(query) > maxPayloadBytes,
		MaxPayloadBytes: maxPayloadBytes,
	}
}

func (d vectorProviderDecision) RecallWarning() RecallWarning {
	return RecallWarning{
		Code:     RecallWarningSemanticUnavailable,
		Stage:    RecallStageGenerate,
		Severity: RecallWarningDegraded,
		Message:  "optional semantic provider skipped because query exceeds configured provider payload limit; lexical/graph recall fallback remained active",
		Evidence: map[string]string{
			"provider":          string(ProviderKindEmbedding),
			"error":             "max_payload_exceeded",
			"max_payload_bytes": fmt.Sprintf("%d", d.MaxPayloadBytes),
		},
	}
}

func vectorHitsByScoreDesc(hits []VectorSearchHit) []VectorSearchHit {
	out := sliceutil.Clone(hits)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	return out
}

func vectorHitAllowedBySources(sources []string, hit VectorSearchHit) bool {
	return vectorSourceAllowed(sources, vectorHitSourceType(hit))
}

func vectorSourceAllowed(sources []string, sourceType string) bool {
	return sourcefilter.Allows(sources, sourceType, true)
}
