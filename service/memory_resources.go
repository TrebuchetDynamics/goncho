package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

type MemoryResourceKind string

const (
	MemoryResourceKindResource MemoryResourceKind = "resource"
	MemoryResourceKindPrompt   MemoryResourceKind = "prompt"
)

type MemoryResourceDescriptor struct {
	URI         string             `json:"uri"`
	Name        string             `json:"name"`
	Kind        MemoryResourceKind `json:"kind"`
	Description string             `json:"description"`
	MimeType    string             `json:"mime_type"`
}

type MemoryResourceRequest struct {
	URI        string `json:"uri"`
	ProfileID  string `json:"profile_id,omitempty"`
	Peer       string `json:"peer,omitempty"`
	Query      string `json:"query,omitempty"`
	SessionKey string `json:"session_key,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type MemoryResourceContent struct {
	URI      string         `json:"uri"`
	MimeType string         `json:"mime_type"`
	Payload  map[string]any `json:"payload"`
}

type MemoryResourceRegistry struct {
	svc *Service
}

func NewMemoryResourceRegistry(svc *Service) *MemoryResourceRegistry {
	return &MemoryResourceRegistry{svc: svc}
}

func (r *MemoryResourceRegistry) Descriptors() []MemoryResourceDescriptor {
	out := []MemoryResourceDescriptor{
		{URI: "goncho://status", Name: "status", Kind: MemoryResourceKindResource, Description: "Goncho memory status and capability summary.", MimeType: "application/json"},
		{URI: "goncho://profile", Name: "profile", Kind: MemoryResourceKindResource, Description: "Peer profile facts for the requested peer/profile scope.", MimeType: "application/json"},
		{URI: "goncho://latest", Name: "latest_memories", Kind: MemoryResourceKindResource, Description: "Latest local memories visible to the requested peer/session scope.", MimeType: "application/json"},
		{URI: "goncho://graph/stats", Name: "graph_stats", Kind: MemoryResourceKindResource, Description: "Local annotation graph counts and relation-like fact counts.", MimeType: "application/json"},
		{URI: "goncho://recall/prompt", Name: "recall_prompt", Kind: MemoryResourceKindPrompt, Description: "Prompt template for evidence-first recall with provenance.", MimeType: "text/plain"},
		{URI: "goncho://handoff/prompt", Name: "session_handoff", Kind: MemoryResourceKindPrompt, Description: "Prompt template for session handoff with evidence and next actions.", MimeType: "text/plain"},
		{URI: "goncho://review/prompt", Name: "review_resolution", Kind: MemoryResourceKindPrompt, Description: "Prompt template for resolving review items with explicit evidence.", MimeType: "text/plain"},
		{URI: "goncho://verify/prompt", Name: "verification_before_action", Kind: MemoryResourceKindPrompt, Description: "Prompt template requiring verification before consequential actions.", MimeType: "text/plain"},
	}
	sort.Slice(out, func(i, j int) bool { return out[i].URI < out[j].URI })
	return out
}

func (r *MemoryResourceRegistry) Read(ctx context.Context, req MemoryResourceRequest) (MemoryResourceContent, error) {
	if r == nil || r.svc == nil {
		return MemoryResourceContent{}, fmt.Errorf("goncho: memory resource registry requires service")
	}
	uri := strings.TrimSpace(req.URI)
	switch uri {
	case "goncho://status":
		return r.status(ctx, req)
	case "goncho://profile":
		return r.profile(ctx, req)
	case "goncho://latest":
		return r.latest(ctx, req)
	case "goncho://graph/stats":
		return r.graphStats(ctx, req)
	case "goncho://recall/prompt":
		return r.recallPrompt(ctx, req)
	case "goncho://handoff/prompt":
		return r.sessionHandoffPrompt(ctx, req)
	case "goncho://review/prompt":
		return r.reviewResolutionPrompt(ctx, req)
	case "goncho://verify/prompt":
		return r.verificationBeforeActionPrompt(ctx, req)
	default:
		return MemoryResourceContent{}, fmt.Errorf("goncho: unknown memory resource %q", req.URI)
	}
}

func (r *MemoryResourceRegistry) status(ctx context.Context, req MemoryResourceRequest) (MemoryResourceContent, error) {
	queue, err := ReadQueueStatus(ctx, r.svc.db)
	if err != nil {
		return MemoryResourceContent{}, err
	}
	payload := map[string]any{
		"workspace_id":     r.svc.workspaceID,
		"observer_peer_id": r.svc.observer,
		"peer":             strings.TrimSpace(req.Peer),
		"capabilities": []string{
			"context", "search", "recall", "remember", "profile", "observations", "hook_capture", "query_expansion", "vector_store_optional", "resources", "prompts",
		},
		"queue_status": queue,
	}
	return memoryResourceJSON("goncho://status", payload), nil
}

func (r *MemoryResourceRegistry) profile(ctx context.Context, req MemoryResourceRequest) (MemoryResourceContent, error) {
	profile, err := r.svc.ProfileInNamespace(ctx, MemoryNamespace{WorkspaceID: r.svc.workspaceID, ProfileID: req.ProfileID, PeerID: req.Peer, Scope: MemoryScopeProfile})
	if err != nil {
		return MemoryResourceContent{}, err
	}
	return memoryResourceJSON("goncho://profile", map[string]any{
		"workspace_id": profile.WorkspaceID,
		"profile_id":   profile.ProfileID,
		"peer":         profile.Peer,
		"card":         profile.Card,
		"hint":         profile.Hint,
	}), nil
}

func (r *MemoryResourceRegistry) latest(ctx context.Context, req MemoryResourceRequest) (MemoryResourceContent, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	result, err := r.svc.Search(ctx, SearchParams{ProfileID: req.ProfileID, Peer: req.Peer, Query: "", SessionKey: req.SessionKey, Scope: req.Scope, Limit: limit})
	if err != nil {
		return MemoryResourceContent{}, err
	}
	return memoryResourceJSON("goncho://latest", map[string]any{
		"workspace_id": result.WorkspaceID,
		"profile_id":   result.ProfileID,
		"peer":         result.Peer,
		"session_key":  strings.TrimSpace(req.SessionKey),
		"count":        len(result.Results),
		"results":      result.Results,
	}), nil
}

func (r *MemoryResourceRegistry) graphStats(ctx context.Context, req MemoryResourceRequest) (MemoryResourceContent, error) {
	annotationCount, relationCount, err := memoryResourceGraphCounts(ctx, r.svc.db, r.svc.workspaceID, req.ProfileID, r.svc.observer, req.Peer)
	if err != nil {
		return MemoryResourceContent{}, err
	}
	return memoryResourceJSON("goncho://graph/stats", map[string]any{
		"workspace_id":       r.svc.workspaceID,
		"profile_id":         strings.TrimSpace(req.ProfileID),
		"peer":               strings.TrimSpace(req.Peer),
		"annotation_count":   annotationCount,
		"relation_count":     relationCount,
		"graph_source":       "goncho_memory_annotations",
		"relation_heuristic": "annotation values containing relation phrases: uses, depends on, runs on, owns, located at",
	}), nil
}

func (r *MemoryResourceRegistry) recallPrompt(ctx context.Context, req MemoryResourceRequest) (MemoryResourceContent, error) {
	query := strings.TrimSpace(req.Query)
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}
	prompt := fmt.Sprintf("Use goncho_recall before answering. Query: %q. Peer: %q. Session: %q. Limit: %d. Require provenance: cite selected RecallTrace candidate memory_id values, evidence kinds, warnings, and any query_expansion/semantic/graph signals. If recall is empty, say no memory evidence was found and do not invent facts.", query, strings.TrimSpace(req.Peer), strings.TrimSpace(req.SessionKey), limit)
	return MemoryResourceContent{URI: "goncho://recall/prompt", MimeType: "text/plain", Payload: map[string]any{"prompt": prompt, "query": query, "peer": strings.TrimSpace(req.Peer), "limit": limit}}, nil
}

func (r *MemoryResourceRegistry) sessionHandoffPrompt(ctx context.Context, req MemoryResourceRequest) (MemoryResourceContent, error) {
	prompt := fmt.Sprintf("Create an evidence-backed session handoff for peer %q and session %q. Include decisions, files or memory IDs cited from Goncho observations/summaries, unresolved risks, and concrete next actions. Do not invent state that is not supported by Goncho evidence.", strings.TrimSpace(req.Peer), strings.TrimSpace(req.SessionKey))
	return MemoryResourceContent{URI: "goncho://handoff/prompt", MimeType: "text/plain", Payload: map[string]any{"prompt": prompt, "peer": strings.TrimSpace(req.Peer), "session_key": strings.TrimSpace(req.SessionKey)}}, nil
}

func (r *MemoryResourceRegistry) reviewResolutionPrompt(ctx context.Context, req MemoryResourceRequest) (MemoryResourceContent, error) {
	prompt := fmt.Sprintf("Review open Goncho memory items for peer %q. Resolve only with explicit evidence IDs, explain accepted/rejected/superseded/verified decisions, and leave uncertain items open for operator review.", strings.TrimSpace(req.Peer))
	return MemoryResourceContent{URI: "goncho://review/prompt", MimeType: "text/plain", Payload: map[string]any{"prompt": prompt, "peer": strings.TrimSpace(req.Peer)}}, nil
}

func (r *MemoryResourceRegistry) verificationBeforeActionPrompt(ctx context.Context, req MemoryResourceRequest) (MemoryResourceContent, error) {
	prompt := fmt.Sprintf("Before taking consequential action for peer %q, verify relevant Goncho recall evidence for query %q. Cite memory IDs, warnings, stale/superseded evidence, and say what remains unverified before acting.", strings.TrimSpace(req.Peer), strings.TrimSpace(req.Query))
	return MemoryResourceContent{URI: "goncho://verify/prompt", MimeType: "text/plain", Payload: map[string]any{"prompt": prompt, "peer": strings.TrimSpace(req.Peer), "query": strings.TrimSpace(req.Query)}}, nil
}

func memoryResourceGraphCounts(ctx context.Context, db *sql.DB, workspaceID, profileID, observer, peer string) (int, int, error) {
	present, err := sqliteTableExists(ctx, db, "goncho_memory_annotations")
	if err != nil || !present {
		return 0, 0, err
	}
	args := []any{strings.TrimSpace(workspaceID), strings.TrimSpace(profileID), strings.TrimSpace(observer), strings.TrimSpace(peer)}
	where := `workspace_id = ? AND profile_id = ? AND observer_peer_id = ? AND peer_id = ?`
	var annotations int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM goncho_memory_annotations WHERE `+where, args...).Scan(&annotations); err != nil {
		return 0, 0, fmt.Errorf("goncho: count memory annotations: %w", err)
	}
	var relations int
	relationArgs := append([]any{}, args...)
	relationArgs = append(relationArgs, "% uses %", "% depends on %", "% runs on %", "% owns %", "% located at %")
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM goncho_memory_annotations WHERE `+where+` AND (lower(value) LIKE ? OR lower(value) LIKE ? OR lower(value) LIKE ? OR lower(value) LIKE ? OR lower(value) LIKE ?)`, relationArgs...).Scan(&relations); err != nil {
		return 0, 0, fmt.Errorf("goncho: count memory annotation relations: %w", err)
	}
	return annotations, relations, nil
}

func memoryResourceJSON(uri string, payload map[string]any) MemoryResourceContent {
	return MemoryResourceContent{URI: uri, MimeType: "application/json", Payload: payload}
}
