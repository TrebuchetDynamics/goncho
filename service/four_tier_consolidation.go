package goncho

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

type MemoryConsolidationTier string

const (
	ConsolidationTierWorking    MemoryConsolidationTier = "working"
	ConsolidationTierEpisodic   MemoryConsolidationTier = "episodic"
	ConsolidationTierSemantic   MemoryConsolidationTier = "semantic"
	ConsolidationTierProcedural MemoryConsolidationTier = "procedural"
)

type FourTierConsolidationParams struct {
	ProfileID  string `json:"profile_id,omitempty"`
	Peer       string `json:"peer"`
	SessionKey string `json:"session_key"`
	Scope      string `json:"scope,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type ConsolidatedMemory struct {
	Tier       MemoryConsolidationTier `json:"tier"`
	MemoryID   int64                   `json:"memory_id"`
	Content    string                  `json:"content"`
	Provenance []EvidenceItem          `json:"provenance"`
}

type FourTierConsolidationResult struct {
	WorkspaceID string               `json:"workspace_id"`
	ProfileID   string               `json:"profile_id,omitempty"`
	Peer        string               `json:"peer"`
	SessionKey  string               `json:"session_key"`
	Items       []ConsolidatedMemory `json:"items"`
}

func (s *Service) ExecuteFourTierConsolidation(ctx context.Context, params FourTierConsolidationParams) (FourTierConsolidationResult, error) {
	peer := strings.TrimSpace(params.Peer)
	sessionKey := strings.TrimSpace(params.SessionKey)
	if peer == "" || sessionKey == "" {
		return FourTierConsolidationResult{}, fmt.Errorf("goncho: peer and session_key are required")
	}
	turns, err := readSessionTurns(ctx, s.db, s.workspaceID, sessionKey)
	if err != nil {
		return FourTierConsolidationResult{}, err
	}
	conclusions, err := readAllConclusions(ctx, s.db, s.workspaceID)
	if err != nil {
		return FourTierConsolidationResult{}, err
	}
	items := s.buildFourTierConsolidationItems(sessionKey, peer, turns, conclusions)
	out := FourTierConsolidationResult{WorkspaceID: s.workspaceID, ProfileID: strings.TrimSpace(params.ProfileID), Peer: peer, SessionKey: sessionKey, Items: []ConsolidatedMemory{}}
	for _, item := range items {
		id, err := s.persistConsolidatedMemory(ctx, params, item)
		if err != nil {
			return FourTierConsolidationResult{}, err
		}
		item.MemoryID = id
		out.Items = append(out.Items, item)
	}
	return out, nil
}

func (s *Service) buildFourTierConsolidationItems(sessionKey, peer string, turns []string, conclusions []conclusionEntry) []ConsolidatedMemory {
	joinedTurns := strings.Join(turns, "\n")
	semantic := firstConclusionContent(conclusions)
	procedure := extractProceduralMemory(joinedTurns)
	if procedure == "" {
		procedure = "procedural consolidation: no explicit procedure observed; verify live state before acting."
	}
	items := []ConsolidatedMemory{
		{
			Tier:       ConsolidationTierWorking,
			Content:    "working consolidation: " + compactText(lastNonBlank(turns), 220),
			Provenance: []EvidenceItem{consolidationEvidence(ConsolidationTierWorking, sessionKey, "turns", len(turns))},
		},
		{
			Tier:       ConsolidationTierEpisodic,
			Content:    "episodic consolidation: session " + sessionKey + " captured " + compactText(joinedTurns, 260),
			Provenance: []EvidenceItem{consolidationEvidence(ConsolidationTierEpisodic, sessionKey, "session", len(turns))},
		},
		{
			Tier:       ConsolidationTierSemantic,
			Content:    "semantic consolidation: " + compactText(semantic, 240),
			Provenance: []EvidenceItem{consolidationEvidence(ConsolidationTierSemantic, sessionKey, "conclusions", len(conclusions))},
		},
		{
			Tier:       ConsolidationTierProcedural,
			Content:    "procedural consolidation: " + compactText(procedure, 240),
			Provenance: []EvidenceItem{consolidationEvidence(ConsolidationTierProcedural, sessionKey, "procedure_extractor", len(turns))},
		},
	}
	return items
}

func (s *Service) persistConsolidatedMemory(ctx context.Context, params FourTierConsolidationParams, item ConsolidatedMemory) (int64, error) {
	evidenceJSON, err := json.Marshal(map[string]any{"provenance": item.Provenance, "tier": item.Tier})
	if err != nil {
		return 0, fmt.Errorf("goncho: marshal consolidation provenance: %w", err)
	}
	peer := strings.TrimSpace(params.Peer)
	profileID := strings.TrimSpace(params.ProfileID)
	scope := normalizeMemoryScope(params.Scope, profileID)
	id, _, err := upsertConclusion(ctx, s.db, conclusionRow{
		WorkspaceID:    s.workspaceID,
		ProfileID:      profileID,
		ObserverPeerID: s.observer,
		PeerID:         peer,
		SessionKey:     params.SessionKey,
		Content:        item.Content,
		Kind:           "consolidation_" + string(item.Tier),
		Status:         "processed",
		Source:         "local_consolidation",
		IdempotencyKey: makeIdempotencyKey(s.workspaceID, profileID, s.observer, peer, params.SessionKey, string(item.Tier)+":"+item.Content),
		EvidenceJSON:   string(evidenceJSON),
		Scope:          scope,
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func consolidationEvidence(tier MemoryConsolidationTier, sessionKey, source string, count int) EvidenceItem {
	return EvidenceItem{
		Kind:   "consolidation",
		Source: source,
		ID:     string(tier),
		Score:  1,
		Note:   "four-tier local consolidation",
		Metadata: map[string]string{
			"tier":         string(tier),
			"session_key":  sessionKey,
			"source_count": fmt.Sprintf("%d", count),
		},
	}
}

func firstConclusionContent(conclusions []conclusionEntry) string {
	for _, conclusion := range conclusions {
		if strings.TrimSpace(conclusion.Conclusion) != "" {
			return conclusion.Conclusion
		}
	}
	return "no prior semantic conclusion available"
}

func extractProceduralMemory(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "procedure:") {
			return strings.TrimSpace(trimmed[len("Procedure:"):])
		}
		if strings.Contains(lower, "before ") || strings.Contains(lower, "always ") || strings.Contains(lower, "must ") {
			return trimmed
		}
	}
	return ""
}

func lastNonBlank(values []string) string {
	for i := len(values) - 1; i >= 0; i-- {
		if strings.TrimSpace(values[i]) != "" {
			return strings.TrimSpace(values[i])
		}
	}
	return "no working turns available"
}

func compactText(value string, limit int) string {
	value = textutil.CollapseWhitespace(value)
	if value == "" {
		return "empty"
	}
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit])
}
