package goncho

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ProposalKind enumerates the types of memory a child agent can propose.
type ProposalKind string

const (
	KindFact        ProposalKind = "fact"
	KindPreference  ProposalKind = "preference"
	KindDecision    ProposalKind = "decision"
	KindObservation ProposalKind = "observation"
	KindReport      ProposalKind = "report"
	KindArtifact    ProposalKind = "artifact"
)

// ValidProposalKind returns true if k is a recognized proposal kind.
func ValidProposalKind(k string) bool {
	switch ProposalKind(k) {
	case KindFact, KindPreference, KindDecision, KindObservation, KindReport, KindArtifact:
		return true
	default:
		return false
	}
}

// ProposalStatus tracks the lifecycle state of a memory proposal.
type ProposalStatus string

const (
	StatusPending   ProposalStatus = "pending"
	StatusAccepted  ProposalStatus = "accepted"
	StatusRejected  ProposalStatus = "rejected"
)

// MemoryProposal represents a child agent's tentative write to durable memory.
// Children propose; parents review and commit.
type MemoryProposal struct {
	ID              int64          `json:"id"`
	ProposalID      string         `json:"proposal_id"`
	SubtaskID       string         `json:"subtask_id"`
	ChildAgentID    string         `json:"child_agent_id"`
	ParentAgentID   string         `json:"parent_agent_id"`
	ProposedTier    MemoryTier     `json:"proposed_tier"`
	Kind            ProposalKind   `json:"kind"`
	Content         string         `json:"content"`
	EvidenceJSON    json.RawMessage `json:"evidence_json"`
	Status          ProposalStatus `json:"status"`
	ReviewedBy      *string        `json:"reviewed_by,omitempty"`
	ReviewedAt      *int64         `json:"reviewed_at,omitempty"`
	CommittedMemoryID *string      `json:"committed_memory_id,omitempty"`
	CreatedAt       int64          `json:"created_at"`
}

// ProposalRef is a lightweight handle returned when submitting a proposal.
type ProposalRef struct {
	ProposalID   string         `json:"proposal_id"`
	ChildAgentID string         `json:"child_agent_id"`
	ParentAgentID string        `json:"parent_agent_id"`
	Status       ProposalStatus `json:"status"`
}

// SubmitProposal inserts a child agent's memory proposal into the database.
// The proposal enters "pending" status awaiting parent review.
// Returns a ProposalRef for the caller to track the submission.
func SubmitProposal(ctx context.Context, db *sql.DB, subtaskID, childAgentID, parentAgentID, content string, tier MemoryTier, kind ProposalKind, evidence any) (*ProposalRef, error) {
	if !ValidTier(string(tier)) {
		return nil, fmt.Errorf("invalid proposed tier %q", tier)
	}
	if !ValidProposalKind(string(kind)) {
		return nil, fmt.Errorf("invalid proposal kind %q", kind)
	}

	evidenceBytes, err := json.Marshal(evidence)
	if err != nil {
		return nil, fmt.Errorf("marshal evidence: %w", err)
	}

	proposalID := fmt.Sprintf("prop_%s_%d", childAgentID, time.Now().UnixNano())
	now := time.Now().Unix()

	_, err = db.ExecContext(ctx, `
		INSERT INTO memory_proposals(
			proposal_id, subtask_id, child_agent_id, parent_agent_id,
			proposed_tier, kind, content, evidence_json, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)
	`, proposalID, subtaskID, childAgentID, parentAgentID, string(tier), string(kind), content, evidenceBytes, now)
	if err != nil {
		return nil, fmt.Errorf("insert memory proposal: %w", err)
	}

	return &ProposalRef{
		ProposalID:   proposalID,
		ChildAgentID: childAgentID,
		ParentAgentID: parentAgentID,
		Status:       StatusPending,
	}, nil
}

// ListPendingProposals returns all pending proposals for a given parent agent,
// ordered by creation time (newest first).
func ListPendingProposals(ctx context.Context, db *sql.DB, parentAgentID string) ([]MemoryProposal, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, proposal_id, subtask_id, child_agent_id, parent_agent_id,
		       proposed_tier, kind, content, evidence_json, status,
		       reviewed_by, reviewed_at, committed_memory_id, created_at
		FROM memory_proposals
		WHERE parent_agent_id = ? AND status = 'pending'
		ORDER BY created_at DESC
	`, parentAgentID)
	if err != nil {
		return nil, fmt.Errorf("query pending proposals: %w", err)
	}
	defer rows.Close()

	var proposals []MemoryProposal
	for rows.Next() {
		var p MemoryProposal
		var evidenceRaw []byte
		if err := rows.Scan(
			&p.ID, &p.ProposalID, &p.SubtaskID, &p.ChildAgentID, &p.ParentAgentID,
			&p.ProposedTier, &p.Kind, &p.Content, &evidenceRaw, &p.Status,
			&p.ReviewedBy, &p.ReviewedAt, &p.CommittedMemoryID, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan proposal: %w", err)
		}
		p.EvidenceJSON = json.RawMessage(evidenceRaw)
		proposals = append(proposals, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proposals: %w", err)
	}
	return proposals, nil
}

// GetProposal fetches a single proposal by its ID.
func GetProposal(ctx context.Context, db *sql.DB, proposalID string) (*MemoryProposal, error) {
	var p MemoryProposal
	var evidenceRaw []byte
	err := db.QueryRowContext(ctx, `
		SELECT id, proposal_id, subtask_id, child_agent_id, parent_agent_id,
		       proposed_tier, kind, content, evidence_json, status,
		       reviewed_by, reviewed_at, committed_memory_id, created_at
		FROM memory_proposals
		WHERE proposal_id = ?
	`, proposalID).Scan(
		&p.ID, &p.ProposalID, &p.SubtaskID, &p.ChildAgentID, &p.ParentAgentID,
		&p.ProposedTier, &p.Kind, &p.Content, &evidenceRaw, &p.Status,
		&p.ReviewedBy, &p.ReviewedAt, &p.CommittedMemoryID, &p.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("proposal %q not found", proposalID)
		}
		return nil, fmt.Errorf("query proposal: %w", err)
	}
	p.EvidenceJSON = json.RawMessage(evidenceRaw)
	return &p, nil
}

// AcceptProposal marks a proposal as accepted and optionally records the
// committed memory ID. The reviewedBy field identifies the parent agent
// that approved the proposal.
func AcceptProposal(ctx context.Context, db *sql.DB, proposalID, reviewedBy string, committedMemoryID *string) error {
	now := time.Now().Unix()
	var result sql.Result
	var err error

	if committedMemoryID != nil {
		result, err = db.ExecContext(ctx, `
			UPDATE memory_proposals
			SET status = 'accepted', reviewed_by = ?, reviewed_at = ?, committed_memory_id = ?
			WHERE proposal_id = ? AND status = 'pending'
		`, reviewedBy, now, *committedMemoryID, proposalID)
	} else {
		result, err = db.ExecContext(ctx, `
			UPDATE memory_proposals
			SET status = 'accepted', reviewed_by = ?, reviewed_at = ?
			WHERE proposal_id = ? AND status = 'pending'
		`, reviewedBy, now, proposalID)
	}
	if err != nil {
		return fmt.Errorf("accept proposal: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("proposal %q not found or not pending", proposalID)
	}
	return nil
}

// RejectProposal marks a proposal as rejected. The reviewedBy field identifies
// the parent agent that rejected the proposal.
func RejectProposal(ctx context.Context, db *sql.DB, proposalID, reviewedBy string) error {
	now := time.Now().Unix()
	result, err := db.ExecContext(ctx, `
		UPDATE memory_proposals
		SET status = 'rejected', reviewed_by = ?, reviewed_at = ?
		WHERE proposal_id = ? AND status = 'pending'
	`, reviewedBy, now, proposalID)
	if err != nil {
		return fmt.Errorf("reject proposal: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("proposal %q not found or not pending", proposalID)
	}
	return nil
}
