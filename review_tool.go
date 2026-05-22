package goncho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	toolmeta "github.com/TrebuchetDynamics/goncho/toolmeta"
)

type ReviewTool struct {
	svc *Service
}

func NewReviewTool(svc *Service) *ReviewTool {
	return &ReviewTool{svc: svc}
}

func (t *ReviewTool) Name() string { return "goncho_review" }

func (t *ReviewTool) Timeout() time.Duration { return 5 * time.Second }

func (t *ReviewTool) Description() string {
	return "Inspect and resolve Goncho memory review items. Use to list open conflict/stale items or mark a review item resolved after evidence review."
}

func (t *ReviewTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["list","resolve"]},"peer_id":{"type":"string"},"session_key":{"type":"string"},"subject_id":{"type":"string"},"related_id":{"type":"string"},"status":{"type":"string","enum":["open","resolved"]},"kind":{"type":"string","enum":["conflict","stale"]},"limit":{"type":"integer"},"id":{"type":"string"},"resolution":{"type":"string","enum":["accepted","rejected","superseded","verified"]},"resolved_by":{"type":"string"},"resolution_reason":{"type":"string"}},"required":["action"]}`)
}

func (t *ReviewTool) Spec() toolmeta.OperationSpec {
	return toolmeta.OperationSpec{
		ToolDescriptor: toolmeta.ToolDescriptor{Name: t.Name(), Description: t.Description(), Schema: t.Schema()},
		Mutating:       true,
		Idempotent:     false,
		PromptSafe:     true,
		TrustClass:     []string{"operator", "system"},
		AuditKind:      "review",
	}
}

func (t *ReviewTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t == nil || t.svc == nil {
		return nil, errors.New("goncho_review: service is required")
	}
	var in struct {
		Action           string `json:"action"`
		PeerID           string `json:"peer_id"`
		SessionKey       string `json:"session_key"`
		SubjectID        string `json:"subject_id"`
		RelatedID        string `json:"related_id"`
		Status           string `json:"status"`
		Kind             string `json:"kind"`
		Limit            int    `json:"limit"`
		ID               string `json:"id"`
		Resolution       string `json:"resolution"`
		ResolvedBy       string `json:"resolved_by"`
		ResolutionReason string `json:"resolution_reason"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("goncho_review: %w", err)
	}
	switch strings.TrimSpace(in.Action) {
	case "list":
		return t.executeList(ctx, in.PeerID, in.SessionKey, in.SubjectID, in.RelatedID, in.Status, in.Kind, in.Limit)
	case "resolve":
		return t.executeResolve(ctx, in.ID, in.Resolution, in.ResolvedBy, in.ResolutionReason)
	default:
		return nil, errors.New("goncho_review: action must be list or resolve")
	}
}

func (t *ReviewTool) executeList(ctx context.Context, peerID, sessionKey, subjectID, relatedID, status, kind string, limit int) (json.RawMessage, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		status = string(ReviewStatusOpen)
	}
	items, err := t.svc.ListReviewItems(ctx, ReviewQuery{
		PeerID:     peerID,
		SessionKey: sessionKey,
		SubjectID:  subjectID,
		RelatedID:  relatedID,
		Status:     ReviewStatus(status),
		Kind:       ReviewKind(kind),
		Limit:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("goncho_review list: %w", err)
	}
	return json.Marshal(map[string]any{
		"success": true,
		"action":  "list",
		"count":   items.Count,
		"items":   items.Items,
	})
}

func (t *ReviewTool) executeResolve(ctx context.Context, id, resolution, resolvedBy, reason string) (json.RawMessage, error) {
	item, err := t.svc.ResolveReviewItem(ctx, ReviewResolutionParams{
		ID:               id,
		Resolution:       ReviewResolution(resolution),
		ResolvedBy:       resolvedBy,
		ResolutionReason: reason,
	})
	if err != nil {
		return nil, fmt.Errorf("goncho_review resolve: %w", err)
	}
	out := map[string]any{
		"success":           true,
		"action":            "resolve",
		"id":                item.ID,
		"kind":              item.Kind,
		"peer_id":           item.PeerID,
		"session_key":       item.SessionKey,
		"subject_id":        item.SubjectID,
		"related_id":        item.RelatedID,
		"evidence_ids":      item.EvidenceIDs,
		"status":            item.Status,
		"resolution":        item.Resolution,
		"resolved_by":       item.ResolvedBy,
		"resolution_reason": item.ResolutionReason,
	}
	if item.ResolvedAt != nil {
		out["resolved_at"] = item.ResolvedAt.UTC().Format(time.RFC3339Nano)
	}
	return json.Marshal(out)
}
