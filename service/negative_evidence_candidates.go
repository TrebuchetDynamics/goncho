package goncho

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type NegativeEvidenceCandidateKind string

const (
	NegativeEvidenceRepeatedToolFailure NegativeEvidenceCandidateKind = "repeated_tool_failure"

	// negativeEvidenceObservationScanLimit is a pre-candidate scan guard. It must be
	// larger than the review-item limit because candidate grouping happens after
	// observations are fetched. The scan is intentionally newest-first so a full
	// history cannot hide the failures most likely to be repeated now.
	negativeEvidenceObservationScanLimit = 5000
)

func negativeEvidenceObservationScanOrderSQL() string {
	return " ORDER BY observed_at DESC, id DESC LIMIT ?"
}

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
	observations, err := s.listNegativeEvidenceObservations(ctx, q)
	if err != nil {
		return nil, err
	}
	projection := ProjectSessionEvidence(SessionEvidenceInput{WorkspaceID: serviceObservationWorkspace(s.workspaceID, q.WorkspaceID)})
	return GenerateNegativeEvidenceCandidates(NegativeEvidenceCandidateInput{Projection: projection, Observations: observations, MinFailures: 2}), nil
}

func (s *Service) CreateNegativeEvidenceReviewItems(ctx context.Context, req NegativeEvidenceReviewRequest) ([]ReviewItem, error) {
	if s == nil {
		return nil, ErrObservationInvalid
	}
	requestWorkspaceID := serviceObservationWorkspace(s.workspaceID, req.WorkspaceID)
	candidates, err := s.NegativeEvidenceCandidates(ctx, negativeEvidenceReviewObservationQuery(req))
	if err != nil {
		return nil, err
	}
	created := []ReviewItem{}
	for _, candidate := range candidates {
		reviewWorkspaceID := negativeEvidenceReviewWorkspaceID(requestWorkspaceID, candidate)
		subjectID := negativeEvidenceReviewSubjectID(candidate)
		existing, err := s.ListReviewItems(ctx, ReviewQuery{WorkspaceID: reviewWorkspaceID, PeerID: candidate.PeerID, SessionKey: candidate.SessionKey, SubjectID: subjectID, Status: ReviewStatusOpen, Limit: 1})
		if err != nil {
			return nil, err
		}
		if len(existing.Items) > 0 {
			continue
		}
		kind := ReviewKindStale
		item, err := s.CreateReviewItem(ctx, ReviewItemCreateParams{
			Kind:        kind,
			WorkspaceID: reviewWorkspaceID,
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
		if negativeEvidenceReviewCreationLimitReached(len(created), req.Limit) {
			break
		}
	}
	return created, nil
}

func negativeEvidenceReviewObservationQuery(req NegativeEvidenceReviewRequest) ObservationQuery {
	return negativeEvidenceObservationQuery(ObservationQuery{
		WorkspaceID: req.WorkspaceID,
		ProfileID:   req.ProfileID,
		PeerID:      req.PeerID,
		SessionKey:  req.SessionKey,
	})
}

func negativeEvidenceReviewWorkspaceID(requestWorkspaceID string, candidate NegativeEvidenceCandidate) string {
	if workspaceID := strings.TrimSpace(candidate.WorkspaceID); workspaceID != "" {
		return workspaceID
	}
	return requestWorkspaceID
}

func negativeEvidenceObservationQuery(q ObservationQuery) ObservationQuery {
	q.Limit = negativeEvidenceObservationLimit(q.Limit)
	if len(q.Kinds) == 0 {
		q.Kinds = negativeEvidenceFailureCapableKinds()
	}
	return q
}

func negativeEvidenceObservationLimit(limit int) int {
	if limit <= 0 || limit > negativeEvidenceObservationScanLimit {
		return negativeEvidenceObservationScanLimit
	}
	return limit
}

func negativeEvidenceFailureCapableKinds() []ObservationKind {
	return []ObservationKind{
		ObservationKindToolError,
		ObservationKindToolResult,
		ObservationKindCustom,
		ObservationKindToolCall,
	}
}

func (s *Service) listNegativeEvidenceObservations(ctx context.Context, q ObservationQuery) ([]Observation, error) {
	query := "SELECT id, kind, workspace_id, profile_id, peer_id, session_key, context_id, input, output, success, metadata_json, input_truncated, output_truncated, input_original_bytes, output_original_bytes, redacted, redaction_count, checksum, observed_at FROM goncho_observations WHERE 1=1"
	args := []any{}
	if workspaceID := strings.TrimSpace(q.WorkspaceID); workspaceID != "" && workspaceID != "*" {
		query += " AND workspace_id = ?"
		args = append(args, workspaceID)
	}
	if profileID := strings.TrimSpace(q.ProfileID); profileID != "" {
		query += " AND profile_id = ?"
		args = append(args, profileID)
	}
	if peerID := strings.TrimSpace(q.PeerID); peerID != "" {
		query += " AND peer_id = ?"
		args = append(args, peerID)
	}
	if sessionKey := strings.TrimSpace(q.SessionKey); sessionKey != "" {
		query += " AND session_key = ?"
		args = append(args, sessionKey)
	}
	if len(q.Kinds) > 0 {
		placeholders := make([]string, 0, len(q.Kinds))
		for _, kind := range q.Kinds {
			placeholders = append(placeholders, "?")
			args = append(args, string(kind))
		}
		query += " AND kind IN (" + strings.Join(placeholders, ",") + ")"
	}
	query += negativeEvidenceObservationScanOrderSQL()
	args = append(args, q.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Observation{}
	for rows.Next() {
		obs, err := scanNegativeEvidenceObservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, obs)
	}
	return out, rows.Err()
}

func scanNegativeEvidenceObservation(rows *sql.Rows) (Observation, error) {
	var obs Observation
	var kind string
	var success sql.NullInt64
	var metadata string
	var observedAt int64
	if err := rows.Scan(&obs.ID, &kind, &obs.WorkspaceID, &obs.ProfileID, &obs.PeerID, &obs.SessionKey, &obs.ContextID, &obs.Input, &obs.Output, &success, &metadata, &obs.InputTruncated, &obs.OutputTruncated, &obs.InputOriginalBytes, &obs.OutputOriginalBytes, &obs.Redacted, &obs.RedactionCount, &obs.Checksum, &observedAt); err != nil {
		return Observation{}, err
	}
	obs.Kind = ObservationKind(kind)
	if success.Valid {
		value := success.Int64 != 0
		obs.Success = &value
	}
	obs.Metadata = map[string]string{}
	if strings.TrimSpace(metadata) != "" {
		_ = json.Unmarshal([]byte(metadata), &obs.Metadata)
	}
	obs.ObservedAt = time.Unix(0, observedAt).UTC()
	return obs, nil
}

func GenerateNegativeEvidenceCandidates(input NegativeEvidenceCandidateInput) []NegativeEvidenceCandidate {
	minFailures := input.MinFailures
	if minFailures <= 0 {
		minFailures = 2
	}
	buckets := map[negativeEvidenceCandidateKey]*negativeEvidenceBucket{}
	for _, obs := range input.Observations {
		negativeEvidenceBucketObservation(buckets, input.Projection, obs)
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

type negativeEvidenceBucket struct {
	candidate      NegativeEvidenceCandidate
	evidenceID     []negativeEvidenceEvidenceRef
	observationIDs map[string]struct{}
}

func negativeEvidenceBucketObservation(buckets map[negativeEvidenceCandidateKey]*negativeEvidenceBucket, projection SessionEvidenceProjection, obs Observation) {
	signal, ok := negativeEvidenceFailureSignalFrom(projection, obs)
	if !ok {
		return
	}
	key := signal.Scope.key()
	b := buckets[key]
	if b == nil {
		b = &negativeEvidenceBucket{candidate: signal.Scope.candidate(), observationIDs: map[string]struct{}{}}
		buckets[key] = b
	}
	if !negativeEvidenceRecordObservation(b.observationIDs, signal.EvidenceID) {
		return
	}
	b.candidate.FailureCount++
	b.evidenceID = append(b.evidenceID, negativeEvidenceEvidenceRef{ID: signal.EvidenceID, ObservedAt: signal.ObservedAt})
	if !signal.ObservedAt.IsZero() && (b.candidate.FirstObservedAt.IsZero() || signal.ObservedAt.Before(b.candidate.FirstObservedAt)) {
		b.candidate.FirstObservedAt = signal.ObservedAt
	}
	if !signal.ObservedAt.IsZero() && signal.ObservedAt.After(b.candidate.LastObservedAt) {
		b.candidate.LastObservedAt = signal.ObservedAt
	}
}

type negativeEvidenceFailureSignal struct {
	EvidenceID string
	Scope      negativeEvidenceObservationScope
	ObservedAt time.Time
}

func negativeEvidenceFailureSignalFrom(projection SessionEvidenceProjection, obs Observation) (negativeEvidenceFailureSignal, bool) {
	evidenceID, ok := negativeEvidenceReplayableFailureObservation(obs)
	if !ok {
		return negativeEvidenceFailureSignal{}, false
	}
	return negativeEvidenceFailureSignal{
		EvidenceID: evidenceID,
		Scope:      negativeEvidenceObservationScopeFrom(projection, obs),
		ObservedAt: obs.ObservedAt,
	}, true
}

type negativeEvidenceObservationScope struct {
	WorkspaceID string
	ProfileID   string
	PeerID      string
	SessionKey  string
	ToolName    string
}

func negativeEvidenceObservationScopeFrom(projection SessionEvidenceProjection, obs Observation) negativeEvidenceObservationScope {
	workspaceID := strings.TrimSpace(obs.WorkspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(projection.WorkspaceID)
	}
	return negativeEvidenceObservationScope{
		WorkspaceID: workspaceID,
		ProfileID:   strings.TrimSpace(obs.ProfileID),
		PeerID:      strings.TrimSpace(obs.PeerID),
		SessionKey:  strings.TrimSpace(obs.SessionKey),
		ToolName:    negativeEvidenceToolName(obs),
	}
}

func (scope negativeEvidenceObservationScope) key() negativeEvidenceCandidateKey {
	return negativeEvidenceCandidateKey{WorkspaceID: scope.WorkspaceID, ProfileID: scope.ProfileID, PeerID: scope.PeerID, SessionKey: scope.SessionKey, ToolName: scope.ToolName}
}

func (scope negativeEvidenceObservationScope) candidate() NegativeEvidenceCandidate {
	return NegativeEvidenceCandidate{
		Kind:        NegativeEvidenceRepeatedToolFailure,
		WorkspaceID: scope.WorkspaceID,
		ProfileID:   scope.ProfileID,
		PeerID:      scope.PeerID,
		SessionKey:  scope.SessionKey,
		ToolName:    scope.ToolName,
		EvidenceIDs: []string{},
	}
}

func negativeEvidenceToolName(obs Observation) string {
	for _, key := range []string{"tool_name", "custom_kind"} {
		if value := negativeEvidenceNormalizeToolName(obs.Metadata[key]); value != "" {
			return value
		}
	}
	return string(obs.Kind)
}

func negativeEvidenceNormalizeToolName(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func negativeEvidenceReplayableFailureObservation(obs Observation) (string, bool) {
	if !negativeEvidenceFailureObservation(obs) {
		return "", false
	}
	id := negativeEvidenceObservationID(obs.ID)
	if id == "" {
		return "", false
	}
	return id, true
}

func negativeEvidenceFailureObservation(obs Observation) bool {
	if obs.Success != nil && *obs.Success {
		return false
	}
	if obs.Kind == ObservationKindToolError {
		return true
	}
	if obs.Success != nil && !*obs.Success {
		return negativeEvidenceFailureCapableKind(obs.Kind)
	}
	return false
}

func negativeEvidenceFailureCapableKind(kind ObservationKind) bool {
	switch kind {
	case ObservationKindToolResult, ObservationKindCustom, ObservationKindToolCall:
		return true
	default:
		return false
	}
}

func negativeEvidenceRecordObservation(seen map[string]struct{}, rawID string) bool {
	id := negativeEvidenceObservationID(rawID)
	if id == "" {
		return false
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

func negativeEvidenceReviewCreationLimitReached(created, limit int) bool {
	return limit > 0 && created >= limit
}

func negativeEvidenceRecommendation(candidate NegativeEvidenceCandidate) string {
	return fmt.Sprintf("review as negative memory candidate: %d failures for %s; verify live state before repeating this path", candidate.FailureCount, candidate.ToolName)
}

func (c NegativeEvidenceCandidate) String() string {
	return fmt.Sprintf("kind=%s workspace=%s profile=%s peer=%s session=%s tool=%s failures=%d evidence=%s recommendation=%s", c.Kind, c.WorkspaceID, c.ProfileID, c.PeerID, c.SessionKey, c.ToolName, c.FailureCount, strings.Join(c.EvidenceIDs, ","), c.Recommendation)
}
