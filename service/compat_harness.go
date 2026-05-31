package goncho

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

// HonchoSDKCompatibilityHarness is a hermetic adapter for proving Honcho SDK
// request/response flows against the local Goncho service.
type HonchoSDKCompatibilityHarness struct {
	service *Service
}

func NewHonchoSDKCompatibilityHarness(service *Service) *HonchoSDKCompatibilityHarness {
	return &HonchoSDKCompatibilityHarness{service: service}
}

type HonchoSDKSessionSeed struct {
	PeerID      string
	SessionID   string
	PeerCard    []string
	Conclusions []string
	Messages    []HonchoSDKMessageInput
}

type HonchoSDKMessageInput struct {
	PeerID    string
	Role      string
	Content   string
	Metadata  map[string]any
	CreatedAt time.Time
}

type HonchoSDKSeedResult struct {
	Workspace   HonchoSDKWorkspace    `json:"workspace"`
	Peer        HonchoSDKPeer         `json:"peer"`
	Session     HonchoSDKSession      `json:"session"`
	Messages    []HonchoSDKMessage    `json:"messages"`
	Conclusions []HonchoSDKConclusion `json:"conclusions,omitempty"`
}

type HonchoSDKWorkspace struct {
	ID string `json:"id"`
}

type HonchoSDKPeer struct {
	ID string `json:"id"`
}

type HonchoSDKSession struct {
	ID string `json:"id"`
}

type HonchoSDKMessage struct {
	ID          int64          `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	SessionID   string         `json:"session_id"`
	PeerID      string         `json:"peer_id"`
	Role        string         `json:"role"`
	Content     string         `json:"content"`
	Sequence    int            `json:"seq_in_session"`
	CreatedAt   int64          `json:"created_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type HonchoSDKConclusion struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

type HonchoSDKSearchRequest struct {
	PeerID    string
	SessionID string
	Query     string
	Limit     int
}

type HonchoSDKSearchResponse struct {
	WorkspaceID string               `json:"workspace_id"`
	PeerID      string               `json:"peer_id"`
	Query       string               `json:"query"`
	Results     []HonchoSDKSearchHit `json:"results"`
}

type HonchoSDKSearchHit struct {
	ID           int64          `json:"id,omitempty"`
	Source       string         `json:"source"`
	OriginSource string         `json:"origin_source,omitempty"`
	Content      string         `json:"content"`
	SessionID    string         `json:"session_id,omitempty"`
	Lineage      *SearchLineage `json:"lineage,omitempty"`
}

type HonchoSDKContextPreviewRequest struct {
	PeerID    string
	SessionID string
	Query     string
	Tokens    int
}

type HonchoSDKContextPreview struct {
	WorkspaceID    string                     `json:"workspace_id"`
	PeerID         string                     `json:"peer_id"`
	SessionID      string                     `json:"session_id,omitempty"`
	PeerCard       []string                   `json:"peer_card"`
	Representation string                     `json:"representation"`
	Summary        *HonchoSDKContextSummary   `json:"summary,omitempty"`
	SearchResults  []HonchoSDKSearchHit       `json:"search_results,omitempty"`
	RecentMessages []MessageSlice             `json:"recent_messages,omitempty"`
	Unsupported    []HonchoSDKUnsupportedFlow `json:"unsupported,omitempty"`
}

type HonchoSDKContextSummary struct {
	Content    string `json:"content"`
	MessageID  int64  `json:"message_id"`
	Type       string `json:"summary_type"`
	CreatedAt  int64  `json:"created_at"`
	TokenCount int    `json:"token_count"`
}

type HonchoSDKUnsupportedFlow struct {
	Code     string   `json:"code"`
	Method   string   `json:"method"`
	Endpoint string   `json:"endpoint"`
	Fields   []string `json:"fields"`
}

func (h *HonchoSDKCompatibilityHarness) SeedSession(ctx context.Context, seed HonchoSDKSessionSeed) (HonchoSDKSeedResult, error) {
	if err := h.requireService(); err != nil {
		return HonchoSDKSeedResult{}, err
	}
	peerID := strings.TrimSpace(seed.PeerID)
	if peerID == "" {
		return HonchoSDKSeedResult{}, fmt.Errorf("goncho sdk compatibility: peer_id is required")
	}
	sessionID := strings.TrimSpace(seed.SessionID)
	if sessionID == "" {
		return HonchoSDKSeedResult{}, fmt.Errorf("goncho sdk compatibility: session_id is required")
	}
	if seed.PeerCard != nil {
		if err := h.service.SetProfile(ctx, peerID, seed.PeerCard); err != nil {
			return HonchoSDKSeedResult{}, err
		}
	}

	conclusions := make([]HonchoSDKConclusion, 0, len(seed.Conclusions))
	for _, conclusion := range seed.Conclusions {
		result, err := h.service.Conclude(ctx, ConcludeParams{
			Peer:       peerID,
			Conclusion: conclusion,
			SessionKey: sessionID,
		})
		if err != nil {
			return HonchoSDKSeedResult{}, err
		}
		conclusions = append(conclusions, HonchoSDKConclusion{
			ID:     result.ID,
			Status: result.Status,
		})
	}

	inputs := sliceutil.Map(seed.Messages, func(msg HonchoSDKMessageInput) CreateMessage {
		msgPeer := strings.TrimSpace(msg.PeerID)
		if msgPeer == "" {
			msgPeer = peerID
		}
		return CreateMessage{
			Peer:      msgPeer,
			Role:      msg.Role,
			Content:   msg.Content,
			Metadata:  msg.Metadata,
			CreatedAt: msg.CreatedAt,
		}
	})
	created, err := h.service.CreateMessages(ctx, CreateMessagesParams{
		SessionKey: sessionID,
		Messages:   inputs,
	})
	if err != nil {
		return HonchoSDKSeedResult{}, err
	}

	return HonchoSDKSeedResult{
		Workspace:   HonchoSDKWorkspace{ID: created.WorkspaceID},
		Peer:        HonchoSDKPeer{ID: peerID},
		Session:     HonchoSDKSession{ID: created.SessionKey},
		Messages:    mapHonchoSDKMessages(created.Messages),
		Conclusions: conclusions,
	}, nil
}

func (h *HonchoSDKCompatibilityHarness) Search(ctx context.Context, req HonchoSDKSearchRequest) (HonchoSDKSearchResponse, error) {
	if err := h.requireService(); err != nil {
		return HonchoSDKSearchResponse{}, err
	}
	result, err := h.service.Search(ctx, SearchParams{
		Peer:       req.PeerID,
		Query:      req.Query,
		SessionKey: req.SessionID,
		Limit:      req.Limit,
	})
	if err != nil {
		return HonchoSDKSearchResponse{}, err
	}
	return HonchoSDKSearchResponse{
		WorkspaceID: result.WorkspaceID,
		PeerID:      result.Peer,
		Query:       result.Query,
		Results:     mapHonchoSDKSearchHits(result.Results),
	}, nil
}

func (h *HonchoSDKCompatibilityHarness) ContextPreview(ctx context.Context, req HonchoSDKContextPreviewRequest) (HonchoSDKContextPreview, error) {
	if err := h.requireService(); err != nil {
		return HonchoSDKContextPreview{}, err
	}
	includeSummary := true
	result, err := h.service.Context(ctx, ContextParams{
		Peer:       req.PeerID,
		Query:      req.Query,
		SessionKey: req.SessionID,
		Tokens:     req.Tokens,
		Summary:    &includeSummary,
	})
	if err != nil {
		return HonchoSDKContextPreview{}, err
	}
	return HonchoSDKContextPreview{
		WorkspaceID:    result.WorkspaceID,
		PeerID:         result.Peer,
		SessionID:      result.SessionKey,
		PeerCard:       cloneStrings(result.PeerCard),
		Representation: result.Representation,
		Summary:        mapHonchoSDKSummary(result.Summary),
		SearchResults:  mapHonchoSDKSearchHits(result.SearchResults),
		RecentMessages: sliceutil.Clone(result.RecentMessages),
		Unsupported:    mapContextUnsupportedFlows(result.Unavailable),
	}, nil
}

func UnsupportedHonchoSDKFlow(method, endpoint string, fields ...string) HonchoSDKUnsupportedFlow {
	copiedFields := cloneStrings(fields)
	if copiedFields == nil {
		copiedFields = []string{}
	}
	return HonchoSDKUnsupportedFlow{
		Code:     "sdk_flow_unsupported",
		Method:   strings.ToUpper(strings.TrimSpace(method)),
		Endpoint: strings.TrimSpace(endpoint),
		Fields:   copiedFields,
	}
}

func (h *HonchoSDKCompatibilityHarness) requireService() error {
	if h == nil || h.service == nil {
		return fmt.Errorf("goncho sdk compatibility: service is required")
	}
	return nil
}

func mapHonchoSDKMessages(messages []MessageRecord) []HonchoSDKMessage {
	return sliceutil.Map(messages, func(msg MessageRecord) HonchoSDKMessage {
		return HonchoSDKMessage{
			ID:          msg.ID,
			WorkspaceID: msg.WorkspaceID,
			SessionID:   msg.SessionKey,
			PeerID:      msg.Peer,
			Role:        msg.Role,
			Content:     msg.Content,
			Sequence:    msg.Sequence,
			CreatedAt:   msg.CreatedAt,
			Metadata:    copyMetadata(msg.Metadata),
		}
	})
}

func mapHonchoSDKSearchHits(hits []SearchHit) []HonchoSDKSearchHit {
	return sliceutil.Map(hits, func(hit SearchHit) HonchoSDKSearchHit {
		return HonchoSDKSearchHit{
			ID:           hit.ID,
			Source:       hit.Source,
			OriginSource: hit.OriginSource,
			Content:      hit.Content,
			SessionID:    hit.SessionKey,
			Lineage:      hit.Lineage,
		}
	})
}

func mapHonchoSDKSummary(summary *SessionSummary) *HonchoSDKContextSummary {
	if summary == nil {
		return nil
	}
	return &HonchoSDKContextSummary{
		Content:    summary.Content,
		MessageID:  summary.MessageID,
		Type:       summary.SummaryType,
		CreatedAt:  summary.CreatedAt,
		TokenCount: summary.TokenCount,
	}
}

func mapContextUnsupportedFlows(unavailable []ContextUnavailableEvidence) []HonchoSDKUnsupportedFlow {
	if len(unavailable) == 0 {
		return nil
	}
	fields := sliceutil.FilterMap(unavailable, func(item ContextUnavailableEvidence) (string, bool) {
		return item.Field, item.Field != ""
	})
	return []HonchoSDKUnsupportedFlow{
		UnsupportedHonchoSDKFlow("GET", "/v3/workspaces/{workspace_id}/peers/{peer_id}/context", fields...),
	}
}
