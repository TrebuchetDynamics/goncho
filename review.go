package goncho

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"time"
)

type ReviewKind string

const (
	ReviewKindConflict ReviewKind = "conflict"
	ReviewKindStale    ReviewKind = "stale"
)

type ReviewStatus string

const (
	ReviewStatusOpen     ReviewStatus = "open"
	ReviewStatusResolved ReviewStatus = "resolved"
)

type ReviewResolution string

const (
	ReviewResolutionAccepted   ReviewResolution = "accepted"
	ReviewResolutionRejected   ReviewResolution = "rejected"
	ReviewResolutionSuperseded ReviewResolution = "superseded"
	ReviewResolutionVerified   ReviewResolution = "verified"
)

type ReviewItemCreateParams struct {
	Kind        ReviewKind `json:"kind"`
	WorkspaceID string     `json:"workspace_id,omitempty"`
	PeerID      string     `json:"peer_id,omitempty"`
	SessionKey  string     `json:"session_key,omitempty"`
	SubjectID   string     `json:"subject_id"`
	RelatedID   string     `json:"related_id,omitempty"`
	Reason      string     `json:"reason"`
	EvidenceIDs []string   `json:"evidence_ids,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
}

type ReviewItem struct {
	ID               string           `json:"id"`
	Kind             ReviewKind       `json:"kind"`
	Status           ReviewStatus     `json:"status"`
	WorkspaceID      string           `json:"workspace_id"`
	PeerID           string           `json:"peer_id,omitempty"`
	SessionKey       string           `json:"session_key,omitempty"`
	SubjectID        string           `json:"subject_id"`
	RelatedID        string           `json:"related_id,omitempty"`
	Reason           string           `json:"reason"`
	EvidenceIDs      []string         `json:"evidence_ids,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	Resolution       ReviewResolution `json:"resolution,omitempty"`
	ResolvedBy       string           `json:"resolved_by,omitempty"`
	ResolutionReason string           `json:"resolution_reason,omitempty"`
	ResolvedAt       *time.Time       `json:"resolved_at,omitempty"`
}

type ReviewResolutionParams struct {
	ID               string           `json:"id"`
	Resolution       ReviewResolution `json:"resolution"`
	ResolvedBy       string           `json:"resolved_by"`
	ResolutionReason string           `json:"resolution_reason"`
	ResolvedAt       time.Time        `json:"resolved_at,omitempty"`
}

type ReviewQuery struct {
	WorkspaceID string       `json:"workspace_id,omitempty"`
	PeerID      string       `json:"peer_id,omitempty"`
	SessionKey  string       `json:"session_key,omitempty"`
	SubjectID   string       `json:"subject_id,omitempty"`
	RelatedID   string       `json:"related_id,omitempty"`
	Kind        ReviewKind   `json:"kind,omitempty"`
	Status      ReviewStatus `json:"status,omitempty"`
	Limit       int          `json:"limit,omitempty"`
}

type ReviewList struct {
	Items []ReviewItem `json:"items"`
	Count int          `json:"count"`
}

func (s *Service) CreateReviewItem(ctx context.Context, p ReviewItemCreateParams) (ReviewItem, error) {
	if s == nil {
		return ReviewItem{}, fmt.Errorf("goncho: nil service")
	}
	if strings.TrimSpace(p.WorkspaceID) == "" {
		p.WorkspaceID = s.workspaceID
	}
	return CreateReviewItem(ctx, s.db, p)
}

func CreateReviewItem(ctx context.Context, db *sql.DB, p ReviewItemCreateParams) (ReviewItem, error) {
	if err := ctx.Err(); err != nil {
		return ReviewItem{}, err
	}
	if db == nil {
		return ReviewItem{}, fmt.Errorf("goncho: nil db")
	}
	if err := ensureReviewTable(ctx, db); err != nil {
		return ReviewItem{}, err
	}
	item, evidenceJSON, err := normalizeReviewItem(p)
	if err != nil {
		return ReviewItem{}, err
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO goncho_review_items(
			id, kind, status, workspace_id, peer_id, session_key, subject_id, related_id,
			reason, evidence_ids_json, created_at, resolved_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
	`, item.ID, string(item.Kind), string(item.Status), item.WorkspaceID, item.PeerID, item.SessionKey, item.SubjectID, item.RelatedID, item.Reason, evidenceJSON, item.CreatedAt.UTC().UnixNano())
	if err != nil {
		return ReviewItem{}, fmt.Errorf("goncho: create review item: %w", err)
	}
	return item, nil
}

func (s *Service) ListReviewItems(ctx context.Context, q ReviewQuery) (ReviewList, error) {
	if s == nil {
		return ReviewList{}, fmt.Errorf("goncho: nil service")
	}
	q.WorkspaceID = serviceObservationWorkspace(s.workspaceID, q.WorkspaceID)
	return ListReviewItems(ctx, s.db, q)
}

func (s *Service) ResolveReviewItem(ctx context.Context, p ReviewResolutionParams) (ReviewItem, error) {
	if s == nil {
		return ReviewItem{}, fmt.Errorf("goncho: nil service")
	}
	return ResolveReviewItem(ctx, s.db, p)
}

func ResolveReviewItem(ctx context.Context, db *sql.DB, p ReviewResolutionParams) (ReviewItem, error) {
	if err := ctx.Err(); err != nil {
		return ReviewItem{}, err
	}
	if db == nil {
		return ReviewItem{}, fmt.Errorf("goncho: nil db")
	}
	if err := ensureReviewTable(ctx, db); err != nil {
		return ReviewItem{}, err
	}
	id := strings.TrimSpace(p.ID)
	resolution := ReviewResolution(strings.TrimSpace(string(p.Resolution)))
	resolvedBy := strings.TrimSpace(p.ResolvedBy)
	resolutionReason := strings.TrimSpace(p.ResolutionReason)
	if id == "" {
		return ReviewItem{}, fmt.Errorf("goncho: resolve review item requires id")
	}
	if !validReviewResolution(resolution) {
		return ReviewItem{}, fmt.Errorf("goncho: review resolution must be accepted, rejected, superseded, or verified")
	}
	if resolvedBy == "" {
		return ReviewItem{}, fmt.Errorf("goncho: resolve review item requires resolved_by")
	}
	if resolutionReason == "" {
		return ReviewItem{}, fmt.Errorf("goncho: resolve review item requires resolution_reason")
	}
	resolvedAt := p.ResolvedAt.UTC()
	if resolvedAt.IsZero() {
		resolvedAt = time.Now().UTC()
	}
	result, err := db.ExecContext(ctx, `
		UPDATE goncho_review_items
		SET status = ?, resolution = ?, resolved_by = ?, resolution_reason = ?, resolved_at = ?
		WHERE id = ? AND status = ?
	`, string(ReviewStatusResolved), string(resolution), resolvedBy, resolutionReason, resolvedAt.UnixNano(), id, string(ReviewStatusOpen))
	if err != nil {
		return ReviewItem{}, fmt.Errorf("goncho: resolve review item: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return ReviewItem{}, fmt.Errorf("goncho: resolve review item rows: %w", err)
	}
	if rows == 0 {
		return ReviewItem{}, fmt.Errorf("goncho: review item %q not found or not open", id)
	}
	return getReviewItem(ctx, db, id)
}

func (s *Service) reviewContextUnavailableEvidence(ctx context.Context, peer string) ([]ContextUnavailableEvidence, error) {
	items, err := s.ListReviewItems(ctx, ReviewQuery{PeerID: peer, Status: ReviewStatusOpen})
	if err != nil {
		return nil, err
	}
	return reviewRequiredUnavailableEvidence(items.Items), nil
}

func reviewRequiredUnavailableEvidence(items []ReviewItem) []ContextUnavailableEvidence {
	if len(items) == 0 {
		return nil
	}
	counts := map[ReviewKind]int{}
	for _, item := range items {
		counts[item.Kind]++
	}
	parts := []string{}
	for _, kind := range []ReviewKind{ReviewKindConflict, ReviewKindStale} {
		if counts[kind] > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", kind, counts[kind]))
		}
	}
	reason := fmt.Sprintf("%d open review items require adjudication: %s", len(items), strings.Join(parts, " "))
	if chains := reviewItemChainEvidence(items, 3); len(chains) > 0 {
		reason += "; chains=" + strings.Join(chains, " ")
	}
	return []ContextUnavailableEvidence{{
		Field:      "review",
		Capability: "review_required",
		Reason:     reason,
	}}
}

func reviewItemChainEvidence(items []ReviewItem, limit int) []string {
	if limit <= 0 {
		return nil
	}
	chains := []string{}
	for _, item := range items {
		subjectID := strings.TrimSpace(item.SubjectID)
		if subjectID == "" {
			continue
		}
		chain := subjectID
		if relatedID := strings.TrimSpace(item.RelatedID); relatedID != "" {
			chain += "->" + relatedID
		}
		chains = append(chains, chain)
		if len(chains) >= limit {
			break
		}
	}
	return chains
}

func ListReviewItems(ctx context.Context, db *sql.DB, q ReviewQuery) (ReviewList, error) {
	if err := ctx.Err(); err != nil {
		return ReviewList{}, err
	}
	if db == nil {
		return ReviewList{}, fmt.Errorf("goncho: nil db")
	}
	if err := ensureReviewTable(ctx, db); err != nil {
		return ReviewList{}, err
	}
	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	kind := ReviewKind(strings.TrimSpace(string(q.Kind)))
	if kind != "" && kind != ReviewKindConflict && kind != ReviewKindStale {
		return ReviewList{}, fmt.Errorf("goncho: review kind must be conflict or stale")
	}
	status := ReviewStatus(strings.TrimSpace(string(q.Status)))
	if status != "" && status != ReviewStatusOpen && status != ReviewStatusResolved {
		return ReviewList{}, fmt.Errorf("goncho: review status must be open or resolved")
	}
	args := []any{}
	where := []string{}
	appendFilter := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		where = append(where, column+" = ?")
		args = append(args, value)
	}
	appendFilter("workspace_id", q.WorkspaceID)
	appendFilter("peer_id", q.PeerID)
	appendFilter("session_key", q.SessionKey)
	appendFilter("subject_id", q.SubjectID)
	appendFilter("related_id", q.RelatedID)
	appendFilter("kind", string(kind))
	appendFilter("status", string(status))
	query := `SELECT id, kind, status, workspace_id, peer_id, session_key, subject_id, related_id, reason, evidence_ids_json, created_at, resolution, resolved_by, resolution_reason, resolved_at FROM goncho_review_items`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT ?"
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return ReviewList{}, fmt.Errorf("goncho: list review items: %w", err)
	}
	defer rows.Close()
	out := ReviewList{Items: []ReviewItem{}}
	for rows.Next() {
		item, err := scanReviewItem(rows)
		if err != nil {
			return ReviewList{}, err
		}
		out.Items = append(out.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ReviewList{}, fmt.Errorf("goncho: iterate review items: %w", err)
	}
	out.Count = len(out.Items)
	return out, nil
}

func normalizeReviewItem(p ReviewItemCreateParams) (ReviewItem, string, error) {
	kind := ReviewKind(strings.TrimSpace(string(p.Kind)))
	if kind != ReviewKindConflict && kind != ReviewKindStale {
		return ReviewItem{}, "", fmt.Errorf("goncho: invalid review kind %q", p.Kind)
	}
	workspaceID := strings.TrimSpace(p.WorkspaceID)
	subjectID := strings.TrimSpace(p.SubjectID)
	reason := strings.TrimSpace(p.Reason)
	if workspaceID == "" || subjectID == "" || reason == "" {
		return ReviewItem{}, "", fmt.Errorf("goncho: review item requires workspace_id, subject_id, and reason")
	}
	createdAt := p.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	evidence := make([]string, 0, len(p.EvidenceIDs))
	for _, id := range p.EvidenceIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			evidence = append(evidence, id)
		}
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		return ReviewItem{}, "", fmt.Errorf("goncho: marshal review evidence ids: %w", err)
	}
	peerID := strings.TrimSpace(p.PeerID)
	sessionKey := strings.TrimSpace(p.SessionKey)
	relatedID := strings.TrimSpace(p.RelatedID)
	item := ReviewItem{
		ID:          reviewItemID(createdAt, []string{string(kind), workspaceID, peerID, sessionKey, subjectID, relatedID, reason}, evidence),
		Kind:        kind,
		Status:      ReviewStatusOpen,
		WorkspaceID: workspaceID,
		PeerID:      peerID,
		SessionKey:  sessionKey,
		SubjectID:   subjectID,
		RelatedID:   relatedID,
		Reason:      reason,
		EvidenceIDs: evidence,
		CreatedAt:   createdAt,
	}
	return item, string(raw), nil
}

func reviewItemID(createdAt time.Time, parts []string, evidence []string) string {
	h := fnv.New64a()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	for _, id := range evidence {
		_, _ = h.Write([]byte(id))
		_, _ = h.Write([]byte{0})
	}
	return fmt.Sprintf("review_%d_%016x", createdAt.UnixNano(), h.Sum64())
}

func getReviewItem(ctx context.Context, db *sql.DB, id string) (ReviewItem, error) {
	row := db.QueryRowContext(ctx, `SELECT id, kind, status, workspace_id, peer_id, session_key, subject_id, related_id, reason, evidence_ids_json, created_at, resolution, resolved_by, resolution_reason, resolved_at FROM goncho_review_items WHERE id = ?`, id)
	return scanReviewItem(row)
}

type reviewScanner interface{ Scan(...any) error }

func scanReviewItem(scanner reviewScanner) (ReviewItem, error) {
	var item ReviewItem
	var kind, status, evidenceJSON, resolution string
	var createdAt int64
	var resolvedAt sql.NullInt64
	if err := scanner.Scan(&item.ID, &kind, &status, &item.WorkspaceID, &item.PeerID, &item.SessionKey, &item.SubjectID, &item.RelatedID, &item.Reason, &evidenceJSON, &createdAt, &resolution, &item.ResolvedBy, &item.ResolutionReason, &resolvedAt); err != nil {
		return ReviewItem{}, fmt.Errorf("goncho: scan review item: %w", err)
	}
	var evidence []string
	if err := json.Unmarshal([]byte(evidenceJSON), &evidence); err != nil {
		return ReviewItem{}, fmt.Errorf("goncho: decode review evidence ids: %w", err)
	}
	item.Kind = ReviewKind(kind)
	item.Status = ReviewStatus(status)
	item.Resolution = ReviewResolution(resolution)
	item.EvidenceIDs = evidence
	item.CreatedAt = time.Unix(0, createdAt).UTC()
	if resolvedAt.Valid {
		v := time.Unix(0, resolvedAt.Int64).UTC()
		item.ResolvedAt = &v
	}
	return item, nil
}

func validReviewResolution(resolution ReviewResolution) bool {
	switch resolution {
	case ReviewResolutionAccepted, ReviewResolutionRejected, ReviewResolutionSuperseded, ReviewResolutionVerified:
		return true
	default:
		return false
	}
}

func ensureReviewTable(ctx context.Context, db *sql.DB) error {
	for _, stmt := range gonchoReviewDDL {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
				continue
			}
			return fmt.Errorf("goncho: ensure review table: %w", err)
		}
	}
	return nil
}

var gonchoReviewDDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_review_items (
		id TEXT PRIMARY KEY,
		kind TEXT NOT NULL CHECK(kind IN ('conflict','stale')),
		status TEXT NOT NULL CHECK(status IN ('open','resolved')),
		workspace_id TEXT NOT NULL,
		peer_id TEXT NOT NULL DEFAULT '',
		session_key TEXT NOT NULL DEFAULT '',
		subject_id TEXT NOT NULL,
		related_id TEXT NOT NULL DEFAULT '',
		reason TEXT NOT NULL,
		evidence_ids_json TEXT NOT NULL DEFAULT '[]',
		created_at INTEGER NOT NULL,
		resolution TEXT NOT NULL DEFAULT '',
		resolved_by TEXT NOT NULL DEFAULT '',
		resolution_reason TEXT NOT NULL DEFAULT '',
		resolved_at INTEGER
	)`,
	`ALTER TABLE goncho_review_items ADD COLUMN resolution TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE goncho_review_items ADD COLUMN resolved_by TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE goncho_review_items ADD COLUMN resolution_reason TEXT NOT NULL DEFAULT ''`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_review_items_status ON goncho_review_items(workspace_id, status, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_review_items_scope ON goncho_review_items(workspace_id, peer_id, session_key, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_review_items_subject ON goncho_review_items(subject_id, created_at DESC)`,
}
