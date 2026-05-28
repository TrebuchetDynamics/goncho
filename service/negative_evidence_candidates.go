package goncho

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
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
	if q.Limit <= 0 {
		q.Limit = 500
	}
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
	candidates, err := s.NegativeEvidenceCandidates(ctx, ObservationQuery{WorkspaceID: workspaceID, ProfileID: req.ProfileID, PeerID: req.PeerID, SessionKey: req.SessionKey, Limit: req.Limit})
	if err != nil {
		return nil, err
	}
	created := []ReviewItem{}
	for _, candidate := range candidates {
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

func GenerateNegativeEvidenceCandidates(input NegativeEvidenceCandidateInput) []NegativeEvidenceCandidate {
	minFailures := input.MinFailures
	if minFailures <= 0 {
		minFailures = 2
	}
	type bucket struct {
		candidate NegativeEvidenceCandidate
	}
	buckets := map[string]*bucket{}
	for _, obs := range input.Observations {
		if !negativeEvidenceFailureObservation(obs) {
			continue
		}
		toolName := strings.TrimSpace(obs.Metadata["tool_name"])
		if toolName == "" {
			toolName = strings.TrimSpace(obs.Metadata["custom_kind"])
		}
		if toolName == "" {
			toolName = string(obs.Kind)
		}
		workspaceID := strings.TrimSpace(obs.WorkspaceID)
		if workspaceID == "" {
			workspaceID = input.Projection.WorkspaceID
		}
		key := strings.Join([]string{workspaceID, strings.TrimSpace(obs.ProfileID), strings.TrimSpace(obs.PeerID), strings.TrimSpace(obs.SessionKey), toolName}, "\x00")
		b := buckets[key]
		if b == nil {
			b = &bucket{candidate: NegativeEvidenceCandidate{
				Kind:        NegativeEvidenceRepeatedToolFailure,
				WorkspaceID: workspaceID,
				ProfileID:   strings.TrimSpace(obs.ProfileID),
				PeerID:      strings.TrimSpace(obs.PeerID),
				SessionKey:  strings.TrimSpace(obs.SessionKey),
				ToolName:    toolName,
				EvidenceIDs: []string{},
			}}
			buckets[key] = b
		}
		b.candidate.FailureCount++
		if strings.TrimSpace(obs.ID) != "" {
			b.candidate.EvidenceIDs = append(b.candidate.EvidenceIDs, strings.TrimSpace(obs.ID))
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
		sort.Strings(candidate.EvidenceIDs)
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

func negativeEvidenceFailureObservation(obs Observation) bool {
	if obs.Kind == ObservationKindToolError {
		return true
	}
	if obs.Success != nil && !*obs.Success {
		return obs.Kind == ObservationKindToolResult || obs.Kind == ObservationKindCustom || obs.Kind == ObservationKindToolCall
	}
	return false
}

func negativeEvidenceReviewSubjectID(candidate NegativeEvidenceCandidate) string {
	parts := []string{"negative-evidence", string(candidate.Kind), candidate.ToolName, candidate.SessionKey}
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(strings.TrimSpace(part), " ", "-")
	}
	return strings.Join(parts, ":")
}

func negativeEvidenceRecommendation(candidate NegativeEvidenceCandidate) string {
	return fmt.Sprintf("review as negative memory candidate: %d failures for %s; verify live state before repeating this path", candidate.FailureCount, candidate.ToolName)
}

func (c NegativeEvidenceCandidate) String() string {
	return fmt.Sprintf("kind=%s workspace=%s profile=%s peer=%s session=%s tool=%s failures=%d evidence=%s recommendation=%s", c.Kind, c.WorkspaceID, c.ProfileID, c.PeerID, c.SessionKey, c.ToolName, c.FailureCount, strings.Join(c.EvidenceIDs, ","), c.Recommendation)
}
