package goncho

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
)

type NegativeEvidenceCandidateKind string

const (
	NegativeEvidenceRepeatedToolFailure NegativeEvidenceCandidateKind = "repeated_tool_failure"
)

type NegativeEvidenceCandidateInput struct {
	Projection   SessionEvidenceProjection `json:"projection"`
	Observations []Observation             `json:"observations,omitempty"`
	MinFailures  int                       `json:"min_failures,omitempty"`
}

type NegativeEvidenceReviewRequest struct {
	WorkspaceID string    `json:"workspace_id,omitempty"`
	ProfileID   string    `json:"profile_id,omitempty"`
	PeerID      string    `json:"peer_id,omitempty"`
	SessionKey  string    `json:"session_key,omitempty"`
	Limit       int       `json:"limit,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type NegativeEvidenceCandidate struct {
	Kind            NegativeEvidenceCandidateKind `json:"kind"`
	WorkspaceID     string                        `json:"workspace_id,omitempty"`
	ProfileID       string                        `json:"profile_id,omitempty"`
	PeerID          string                        `json:"peer_id,omitempty"`
	SessionKey      string                        `json:"session_key,omitempty"`
	ToolName        string                        `json:"tool_name,omitempty"`
	FailureCount    int                           `json:"failure_count"`
	EvidenceIDs     []string                      `json:"evidence_ids"`
	FirstObservedAt time.Time                     `json:"first_observed_at,omitempty"`
	LastObservedAt  time.Time                     `json:"last_observed_at,omitempty"`
	Recommendation  string                        `json:"recommendation"`
}

func (s *Service) NegativeEvidenceCandidates(ctx context.Context, q ObservationQuery) ([]NegativeEvidenceCandidate, error) {
	if s == nil {
		return nil, ErrObservationInvalid
	}
	q = negativeEvidenceObservationQuery(q)
	list, err := s.ListObservations(ctx, q)
	if err != nil {
		return nil, err
	}
	projection := ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: serviceObservationWorkspace(s.workspaceID, q.WorkspaceID)})
	return GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{Projection: projection, Observations: list.Observations, MinFailures: 2}), nil
}

func (s *Service) CreateNegativeEvidenceReviewItems(ctx context.Context, req NegativeEvidenceReviewRequest) ([]ReviewItem, error) {
	if s == nil {
		return nil, ErrObservationInvalid
	}
	workspaceID := serviceObservationWorkspace(s.workspaceID, req.WorkspaceID)
	candidates, err := s.NegativeEvidenceCandidates(ctx, negativeEvidenceReviewObservationQuery(workspaceID, req))
	if err != nil {
		return nil, err
	}
	created := []ReviewItem{}
	for _, candidate := range limitNegativeEvidenceCandidates(candidates, req.Limit) {
		subjectID := negativeEvidenceReviewSubjectID(candidate)
		existing, err := s.ListReviewItems(ctx, ReviewQuery{WorkspaceID: workspaceID, PeerID: candidate.PeerID, SessionKey: candidate.SessionKey, SubjectID: subjectID, Status: ReviewStatusOpen, Limit: 1})
		if err != nil {
			return nil, err
		}
		if len(existing.Items) > 0 {
			continue
		}
		kind := ReviewKindStale
		item, err := s.CreateReviewItem(ctx, ReviewItemCreateParams{
			Kind:        kind,
			WorkspaceID: workspaceID,
			PeerID:      candidate.PeerID,
			SessionKey:  candidate.SessionKey,
			SubjectID:   subjectID,
			Reason:      candidate.Recommendation,
			EvidenceIDs: candidate.EvidenceIDs,
			CreatedAt:   req.CreatedAt,
		})
		if err != nil {
			return nil, err
		}
		created = append(created, item)
	}
	return created, nil
}

func negativeEvidenceReviewObservationQuery(workspaceID string, req NegativeEvidenceReviewRequest) ObservationQuery {
	return negativeEvidenceObservationQuery(ObservationQuery{
		WorkspaceID: workspaceID,
		ProfileID:   req.ProfileID,
		PeerID:      req.PeerID,
		SessionKey:  req.SessionKey,
	})
}

func negativeEvidenceObservationQuery(q ObservationQuery) ObservationQuery {
	q.Limit = limitutil.Default(q.Limit, 500)
	if len(q.Kinds) == 0 {
		q.Kinds = negativeEvidenceFailureCapableKinds()
	}
	return q
}

func negativeEvidenceFailureCapableKinds() []ObservationKind {
	return []ObservationKind{
		ObservationKindToolError,
		ObservationKindToolResult,
		ObservationKindCustom,
		ObservationKindToolCall,
	}
}

func GenerateNegativeEvidenceCandidates(input NegativeEvidenceCandidateInput) []NegativeEvidenceCandidate {
	minFailures := input.MinFailures
	if minFailures <= 0 {
		minFailures = 2
	}
	type bucket struct {
		candidate      NegativeEvidenceCandidate
		evidenceID     []negativeEvidenceEvidenceRef
		observationIDs map[string]struct{}
	}
	buckets := map[negativeEvidenceCandidateKey]*bucket{}
	for _, obs := range input.Observations {
		if !negativeEvidenceFailureObservation(obs) {
			continue
		}
		key, seed := negativeEvidenceCandidateSeed(input.Projection, obs)
		b := buckets[key]
		if b == nil {
			b = &bucket{candidate: seed, observationIDs: map[string]struct{}{}}
			buckets[key] = b
		}
		if !negativeEvidenceRecordObservation(b.observationIDs, obs.ID) {
			continue
		}
		b.candidate.FailureCount++
		if id := negativeEvidenceObservationID(obs.ID); id != "" {
			b.evidenceID = append(b.evidenceID, negativeEvidenceEvidenceRef{ID: id, ObservedAt: obs.ObservedAt})
		}
		if !obs.ObservedAt.IsZero() && (b.candidate.FirstObservedAt.IsZero() || obs.ObservedAt.Before(b.candidate.FirstObservedAt)) {
			b.candidate.FirstObservedAt = obs.ObservedAt
		}
		if !obs.ObservedAt.IsZero() && obs.ObservedAt.After(b.candidate.LastObservedAt) {
			b.candidate.LastObservedAt = obs.ObservedAt
		}
	}
	out := []NegativeEvidenceCandidate{}
	for _, b := range buckets {
		candidate := b.candidate
		if candidate.FailureCount < minFailures {
			continue
		}
		candidate.EvidenceIDs = negativeEvidenceOrderedEvidenceIDs(b.evidenceID)
		candidate.Recommendation = negativeEvidenceRecommendation(candidate)
		out = append(out, candidate)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FailureCount != out[j].FailureCount {
			return out[i].FailureCount > out[j].FailureCount
		}
		if !out[i].LastObservedAt.Equal(out[j].LastObservedAt) {
			return out[i].LastObservedAt.After(out[j].LastObservedAt)
		}
		return out[i].String() < out[j].String()
	})
	return out
}

type negativeEvidenceCandidateKey struct {
	WorkspaceID string
	ProfileID   string
	PeerID      string
	SessionKey  string
	ToolName    string
}

func negativeEvidenceCandidateSeed(projection SessionEvidenceProjection, obs Observation) (negativeEvidenceCandidateKey, NegativeEvidenceCandidate) {
	toolName := strings.TrimSpace(obs.Metadata["tool_name"])
	if toolName == "" {
		toolName = strings.TrimSpace(obs.Metadata["custom_kind"])
	}
	if toolName == "" {
		toolName = string(obs.Kind)
	}
	workspaceID := strings.TrimSpace(obs.WorkspaceID)
	if workspaceID == "" {
		workspaceID = projection.WorkspaceID
	}
	profileID := strings.TrimSpace(obs.ProfileID)
	peerID := strings.TrimSpace(obs.PeerID)
	sessionKey := strings.TrimSpace(obs.SessionKey)
	key := negativeEvidenceCandidateKey{WorkspaceID: workspaceID, ProfileID: profileID, PeerID: peerID, SessionKey: sessionKey, ToolName: toolName}
	return key, NegativeEvidenceCandidate{
		Kind:        NegativeEvidenceRepeatedToolFailure,
		WorkspaceID: workspaceID,
		ProfileID:   profileID,
		PeerID:      peerID,
		SessionKey:  sessionKey,
		ToolName:    toolName,
		EvidenceIDs: []string{},
	}
}

func negativeEvidenceFailureObservation(obs Observation) bool {
	if obs.Kind == ObservationKindToolError {
		return true
	}
	if obs.Success != nil && !*obs.Success {
		return obs.Kind == ObservationKindToolResult || obs.Kind == ObservationKindCustom || obs.Kind == ObservationKindToolCall
	}
	return false
}

func negativeEvidenceRecordObservation(seen map[string]struct{}, rawID string) bool {
	id := negativeEvidenceObservationID(rawID)
	if id == "" {
		return true
	}
	if _, ok := seen[id]; ok {
		return false
	}
	seen[id] = struct{}{}
	return true
}

func negativeEvidenceObservationID(rawID string) string {
	return strings.TrimSpace(rawID)
}

type negativeEvidenceEvidenceRef struct {
	ID         string
	ObservedAt time.Time
}

func negativeEvidenceOrderedEvidenceIDs(refs []negativeEvidenceEvidenceRef) []string {
	refs = append([]negativeEvidenceEvidenceRef(nil), refs...)
	sort.SliceStable(refs, func(i, j int) bool {
		left, right := refs[i], refs[j]
		if left.ObservedAt.IsZero() != right.ObservedAt.IsZero() {
			return !left.ObservedAt.IsZero()
		}
		if !left.ObservedAt.Equal(right.ObservedAt) {
			return left.ObservedAt.Before(right.ObservedAt)
		}
		return left.ID < right.ID
	})
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.ID != "" {
			out = append(out, ref.ID)
		}
	}
	return out
}

func negativeEvidenceReviewSubjectID(candidate NegativeEvidenceCandidate) string {
	parts := negativeEvidenceReviewSubjectParts(candidate)
	for i, part := range parts {
		parts[i] = negativeEvidenceSubjectToken(part)
	}
	return strings.Join(parts, ":")
}

func negativeEvidenceReviewSubjectParts(candidate NegativeEvidenceCandidate) []string {
	return []string{
		"negative-evidence",
		"kind-" + string(candidate.Kind),
		"workspace-" + candidate.WorkspaceID,
		"profile-" + candidate.ProfileID,
		"peer-" + candidate.PeerID,
		"session-" + candidate.SessionKey,
		"tool-" + candidate.ToolName,
	}
}

func negativeEvidenceSubjectToken(part string) string {
	part = strings.TrimSpace(part)
	if part == "" {
		return "unknown"
	}
	return url.QueryEscape(part)
}

func limitNegativeEvidenceCandidates(candidates []NegativeEvidenceCandidate, limit int) []NegativeEvidenceCandidate {
	if limit <= 0 || len(candidates) <= limit {
		return candidates
	}
	return append([]NegativeEvidenceCandidate(nil), candidates[:limit]...)
}

func negativeEvidenceRecommendation(candidate NegativeEvidenceCandidate) string {
	return fmt.Sprintf("review as negative memory candidate: %d failures for %s; verify live state before repeating this path", candidate.FailureCount, candidate.ToolName)
}

func (c NegativeEvidenceCandidate) String() string {
	return fmt.Sprintf("kind=%s workspace=%s profile=%s peer=%s session=%s tool=%s failures=%d evidence=%s recommendation=%s", c.Kind, c.WorkspaceID, c.ProfileID, c.PeerID, c.SessionKey, c.ToolName, c.FailureCount, strings.Join(c.EvidenceIDs, ","), c.Recommendation)
}
