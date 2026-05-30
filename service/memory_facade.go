package goncho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
)

var ErrMemoryNotFound = errors.New("goncho: memory not found")

// MemoryFacade is a mem0-style tiny API over Goncho's evidence-backed service
// APIs. It keeps caller-supplied IDs stable by storing each memory as a named
// memory slot, and records add/update/delete history as observations.
type MemoryFacade struct{ svc *Service }

func NewMemoryFacade(svc *Service) *MemoryFacade { return &MemoryFacade{svc: svc} }

type MemoryAddParams struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	AgentID     string            `json:"agent_id,omitempty"`
	RunID       string            `json:"run_id,omitempty"`
	SessionKey  string            `json:"session_key,omitempty"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
	ProfileID   string            `json:"profile_id,omitempty"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type MemoryUpdateParams struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	AgentID     string            `json:"agent_id,omitempty"`
	RunID       string            `json:"run_id,omitempty"`
	SessionKey  string            `json:"session_key,omitempty"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
	ProfileID   string            `json:"profile_id,omitempty"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type MemoryDeleteParams struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	AgentID     string `json:"agent_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	SessionKey  string `json:"session_key,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
}

type MemoryGetParams struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
}

type MemorySearchParams struct {
	UserID      string            `json:"user_id"`
	AgentID     string            `json:"agent_id,omitempty"`
	RunID       string            `json:"run_id,omitempty"`
	SessionKey  string            `json:"session_key,omitempty"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
	ProfileID   string            `json:"profile_id,omitempty"`
	Query       string            `json:"query,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Limit       int               `json:"limit,omitempty"`
}

type MemoryHistoryParams struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type MemoryItem struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	AgentID     string            `json:"agent_id,omitempty"`
	RunID       string            `json:"run_id,omitempty"`
	SessionKey  string            `json:"session_key,omitempty"`
	WorkspaceID string            `json:"workspace_id"`
	ProfileID   string            `json:"profile_id,omitempty"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Revision    int               `json:"revision"`
	Deleted     bool              `json:"deleted,omitempty"`
	CreatedAt   int64             `json:"created_at"`
	UpdatedAt   int64             `json:"updated_at"`
	EvidenceIDs []string          `json:"evidence_ids,omitempty"`
}

type MemorySearchResult struct {
	Items []MemoryItem `json:"items"`
	Count int          `json:"count"`
}

type MemoryHistoryResult struct {
	ID     string               `json:"id"`
	Events []MemoryHistoryEvent `json:"events"`
	Count  int                  `json:"count"`
}

type MemoryHistoryEvent struct {
	EvidenceID      string            `json:"evidence_id"`
	Action          string            `json:"action"`
	MemoryID        string            `json:"memory_id"`
	UserID          string            `json:"user_id"`
	AgentID         string            `json:"agent_id,omitempty"`
	RunID           string            `json:"run_id,omitempty"`
	SessionKey      string            `json:"session_key,omitempty"`
	PreviousContent string            `json:"previous_content,omitempty"`
	NewContent      string            `json:"new_content,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	ObservedAt      time.Time         `json:"observed_at"`
}

type memoryFacadeEnvelope struct {
	ID         string            `json:"id"`
	UserID     string            `json:"user_id"`
	AgentID    string            `json:"agent_id,omitempty"`
	RunID      string            `json:"run_id,omitempty"`
	SessionKey string            `json:"session_key,omitempty"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

func (f *MemoryFacade) Add(ctx context.Context, p MemoryAddParams) (MemoryItem, error) {
	if err := f.validate(); err != nil {
		return MemoryItem{}, err
	}
	env, slot, err := f.addEnvelope(p)
	if err != nil {
		return MemoryItem{}, err
	}
	created, err := f.svc.CreateMemorySlot(ctx, slot)
	if err != nil {
		return MemoryItem{}, err
	}
	item := memoryItemFromSlot(created, env)
	obs, err := f.recordMemoryHistory(ctx, "add", item, "", env.Content, env.Metadata)
	if err != nil {
		return MemoryItem{}, err
	}
	item.EvidenceIDs = []string{obs.Observation.ID, memorySlotEvidenceID(item)}
	return item, nil
}

func (f *MemoryFacade) Update(ctx context.Context, p MemoryUpdateParams) (MemoryItem, error) {
	if err := f.validate(); err != nil {
		return MemoryItem{}, err
	}
	old, err := f.Get(ctx, MemoryGetParams{ID: p.ID, UserID: p.UserID, WorkspaceID: p.WorkspaceID, ProfileID: p.ProfileID})
	if err != nil {
		return MemoryItem{}, err
	}
	env, slot, err := f.updateEnvelope(p, old)
	if err != nil {
		return MemoryItem{}, err
	}
	updated, err := f.svc.ReplaceMemorySlot(ctx, slot)
	if err != nil {
		return MemoryItem{}, err
	}
	item := memoryItemFromSlot(updated, env)
	obs, err := f.recordMemoryHistory(ctx, "update", item, old.Content, env.Content, env.Metadata)
	if err != nil {
		return MemoryItem{}, err
	}
	item.EvidenceIDs = []string{obs.Observation.ID, memorySlotEvidenceID(item)}
	return item, nil
}

func (f *MemoryFacade) Delete(ctx context.Context, p MemoryDeleteParams) (MemoryItem, error) {
	if err := f.validate(); err != nil {
		return MemoryItem{}, err
	}
	old, err := f.Get(ctx, MemoryGetParams{ID: p.ID, UserID: p.UserID, WorkspaceID: p.WorkspaceID, ProfileID: p.ProfileID})
	if err != nil {
		return MemoryItem{}, err
	}
	deleted, err := f.svc.DeleteMemorySlot(ctx, MemorySlotQuery{WorkspaceID: p.WorkspaceID, ProfileID: p.ProfileID, Peer: p.UserID, Scope: normalizeMemoryScope("", p.ProfileID), Name: p.ID})
	if err != nil {
		if errors.Is(err, ErrMemorySlotNotFound) {
			return MemoryItem{}, ErrMemoryNotFound
		}
		return MemoryItem{}, err
	}
	item := old
	item.Deleted = true
	item.Revision = deleted.Revision
	item.UpdatedAt = deleted.UpdatedAt
	obs, err := f.recordMemoryHistory(ctx, "delete", item, old.Content, "", old.Metadata)
	if err != nil {
		return MemoryItem{}, err
	}
	item.EvidenceIDs = []string{obs.Observation.ID, memorySlotEvidenceID(item)}
	return item, nil
}

func (f *MemoryFacade) Get(ctx context.Context, p MemoryGetParams) (MemoryItem, error) {
	if err := f.validate(); err != nil {
		return MemoryItem{}, err
	}
	slot, err := f.svc.GetMemorySlot(ctx, MemorySlotQuery{WorkspaceID: p.WorkspaceID, ProfileID: p.ProfileID, Peer: p.UserID, Scope: normalizeMemoryScope("", p.ProfileID), Name: p.ID})
	if err != nil {
		if errors.Is(err, ErrMemorySlotNotFound) {
			return MemoryItem{}, ErrMemoryNotFound
		}
		return MemoryItem{}, err
	}
	env, err := decodeMemoryFacadeEnvelope(slot.Value)
	if err != nil {
		return MemoryItem{}, err
	}
	item := memoryItemFromSlot(slot, env)
	item.EvidenceIDs = []string{memorySlotEvidenceID(item)}
	return item, nil
}

func (f *MemoryFacade) Search(ctx context.Context, p MemorySearchParams) (MemorySearchResult, error) {
	if err := f.validate(); err != nil {
		return MemorySearchResult{}, err
	}
	limit := limitutil.Default(p.Limit, 10)
	list, err := f.svc.ListMemorySlots(ctx, MemorySlotQuery{WorkspaceID: p.WorkspaceID, ProfileID: p.ProfileID, Peer: p.UserID, Scope: normalizeMemoryScope("", p.ProfileID), Limit: max(limit*4, limit)})
	if err != nil {
		return MemorySearchResult{}, err
	}
	items := []MemoryItem{}
	for _, slot := range list.Slots {
		env, err := decodeMemoryFacadeEnvelope(slot.Value)
		if err != nil || !memoryFacadeEnvelopeMatches(env, p) {
			continue
		}
		item := memoryItemFromSlot(slot, env)
		item.EvidenceIDs = []string{memorySlotEvidenceID(item)}
		items = append(items, item)
		if len(items) >= limit {
			break
		}
	}
	return MemorySearchResult{Items: items, Count: len(items)}, nil
}

func (f *MemoryFacade) History(ctx context.Context, p MemoryHistoryParams) (MemoryHistoryResult, error) {
	if err := f.validate(); err != nil {
		return MemoryHistoryResult{}, err
	}
	limit := limitutil.Default(p.Limit, 50)
	list, err := f.svc.ListObservations(ctx, ObservationQuery{WorkspaceID: p.WorkspaceID, ProfileID: p.ProfileID, PeerID: p.UserID, Kinds: []ObservationKind{ObservationKindCustom}, Limit: max(limit*4, limit)})
	if err != nil {
		return MemoryHistoryResult{}, err
	}
	events := []MemoryHistoryEvent{}
	for _, obs := range list.Observations {
		if obs.Metadata["custom_kind"] != "mem0_facade_memory" || obs.Metadata["memory_id"] != strings.TrimSpace(p.ID) {
			continue
		}
		events = append(events, MemoryHistoryEvent{
			EvidenceID:      obs.ID,
			Action:          obs.Metadata["action"],
			MemoryID:        obs.Metadata["memory_id"],
			UserID:          obs.PeerID,
			AgentID:         obs.Metadata["agent_id"],
			RunID:           obs.Metadata["run_id"],
			SessionKey:      obs.SessionKey,
			PreviousContent: obs.Input,
			NewContent:      obs.Output,
			Metadata:        memoryFacadeMetadataFromObservation(obs.Metadata),
			ObservedAt:      obs.ObservedAt,
		})
		if len(events) >= limit {
			break
		}
	}
	return MemoryHistoryResult{ID: strings.TrimSpace(p.ID), Events: events, Count: len(events)}, nil
}

func (f *MemoryFacade) validate() error {
	if f == nil || f.svc == nil {
		return errors.New("goncho: memory facade service is required")
	}
	return nil
}

func (f *MemoryFacade) addEnvelope(p MemoryAddParams) (memoryFacadeEnvelope, MemorySlotParams, error) {
	env := memoryFacadeEnvelope{ID: strings.TrimSpace(p.ID), UserID: strings.TrimSpace(p.UserID), AgentID: firstNonBlank(p.AgentID, f.svc.observer), RunID: firstNonBlank(p.RunID, p.SessionKey), SessionKey: firstNonBlank(p.SessionKey, p.RunID), Content: strings.TrimSpace(p.Content), Metadata: cloneStringMap(p.Metadata)}
	if env.ID == "" || env.UserID == "" || env.Content == "" {
		return memoryFacadeEnvelope{}, MemorySlotParams{}, errors.New("goncho: memory id, user_id, and content are required")
	}
	value, err := encodeMemoryFacadeEnvelope(env)
	if err != nil {
		return memoryFacadeEnvelope{}, MemorySlotParams{}, err
	}
	return env, MemorySlotParams{WorkspaceID: p.WorkspaceID, ProfileID: p.ProfileID, Peer: env.UserID, Scope: normalizeMemoryScope("", p.ProfileID), Name: env.ID, Kind: "mem0_facade", Value: value}, nil
}

func (f *MemoryFacade) updateEnvelope(p MemoryUpdateParams, old MemoryItem) (memoryFacadeEnvelope, MemorySlotParams, error) {
	env := memoryFacadeEnvelope{ID: strings.TrimSpace(p.ID), UserID: strings.TrimSpace(p.UserID), AgentID: firstNonBlank(p.AgentID, old.AgentID, f.svc.observer), RunID: firstNonBlank(p.RunID, p.SessionKey, old.RunID), SessionKey: firstNonBlank(p.SessionKey, p.RunID, old.SessionKey), Content: strings.TrimSpace(p.Content), Metadata: cloneStringMap(p.Metadata)}
	if env.Metadata == nil {
		env.Metadata = cloneStringMap(old.Metadata)
	}
	if env.ID == "" || env.UserID == "" || env.Content == "" {
		return memoryFacadeEnvelope{}, MemorySlotParams{}, errors.New("goncho: memory id, user_id, and content are required")
	}
	value, err := encodeMemoryFacadeEnvelope(env)
	if err != nil {
		return memoryFacadeEnvelope{}, MemorySlotParams{}, err
	}
	return env, MemorySlotParams{WorkspaceID: p.WorkspaceID, ProfileID: p.ProfileID, Peer: env.UserID, Scope: normalizeMemoryScope("", p.ProfileID), Name: env.ID, Kind: "mem0_facade", Value: value}, nil
}

func memoryItemFromSlot(slot MemorySlot, env memoryFacadeEnvelope) MemoryItem {
	return MemoryItem{ID: env.ID, UserID: env.UserID, AgentID: env.AgentID, RunID: env.RunID, SessionKey: env.SessionKey, WorkspaceID: slot.WorkspaceID, ProfileID: slot.ProfileID, Content: env.Content, Metadata: cloneStringMap(env.Metadata), Revision: slot.Revision, Deleted: slot.Deleted, CreatedAt: slot.CreatedAt, UpdatedAt: slot.UpdatedAt}
}

func (f *MemoryFacade) recordMemoryHistory(ctx context.Context, action string, item MemoryItem, previous, next string, metadata map[string]string) (ObservationResult, error) {
	obsMetadata := map[string]string{"custom_kind": "mem0_facade_memory", "action": action, "memory_id": item.ID, "agent_id": item.AgentID, "run_id": item.RunID, "revision": fmt.Sprintf("%d", item.Revision)}
	for key, value := range metadata {
		obsMetadata["metadata."+key] = value
	}
	return f.svc.Observe(ctx, ObservationParams{Kind: ObservationKindCustom, WorkspaceID: item.WorkspaceID, ProfileID: item.ProfileID, PeerID: item.UserID, SessionKey: item.SessionKey, Input: previous, Output: next, Reason: "mem0_facade_" + action, Metadata: obsMetadata})
}

func encodeMemoryFacadeEnvelope(env memoryFacadeEnvelope) (string, error) {
	raw, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeMemoryFacadeEnvelope(value string) (memoryFacadeEnvelope, error) {
	var env memoryFacadeEnvelope
	if err := json.Unmarshal([]byte(value), &env); err != nil {
		return memoryFacadeEnvelope{}, fmt.Errorf("goncho: decode memory facade envelope: %w", err)
	}
	return env, nil
}

func memoryFacadeEnvelopeMatches(env memoryFacadeEnvelope, p MemorySearchParams) bool {
	if strings.TrimSpace(p.AgentID) != "" && env.AgentID != strings.TrimSpace(p.AgentID) {
		return false
	}
	if runID := firstNonBlank(p.RunID, p.SessionKey); runID != "" && env.RunID != runID && env.SessionKey != runID {
		return false
	}
	for key, value := range p.Metadata {
		if env.Metadata[key] != value {
			return false
		}
	}
	return memoryFacadeQueryMatches(env.Content, p.Query)
}

func memoryFacadeQueryMatches(content, query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return true
	}
	content = strings.ToLower(content)
	for _, token := range strings.Fields(query) {
		if !strings.Contains(content, strings.Trim(token, ".,;:!?()[]{}\"'")) {
			return false
		}
	}
	return true
}

func memoryFacadeMetadataFromObservation(metadata map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range metadata {
		if strings.HasPrefix(key, "metadata.") {
			out[strings.TrimPrefix(key, "metadata.")] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func memorySlotEvidenceID(item MemoryItem) string {
	return fmt.Sprintf("memory_slot:%s:rev:%d", item.ID, item.Revision)
}

func sortMemoryItemsByUpdatedAtDesc(items []MemoryItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].UpdatedAt != items[j].UpdatedAt {
			return items[i].UpdatedAt > items[j].UpdatedAt
		}
		return items[i].ID < items[j].ID
	})
}
