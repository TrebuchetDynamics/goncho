package skillproposals

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	workspacepkg "github.com/TrebuchetDynamics/goncho/workspace"
)

type SkillLearningProposalStatus string

const (
	SkillLearningProposalPending  SkillLearningProposalStatus = "pending"
	SkillLearningProposalApproved SkillLearningProposalStatus = "approved"
	SkillLearningProposalRejected SkillLearningProposalStatus = "rejected"
)

type SkillLearningProposalCreateParams struct {
	WorkspaceID string    `json:"workspace_id,omitempty"`
	SkillName   string    `json:"skill_name"`
	SourceTask  string    `json:"source_task"`
	DraftBody   string    `json:"draft_body"`
	CreatedBy   string    `json:"created_by"`
	Evidence    any       `json:"evidence,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type SkillLearningProposalReviewParams struct {
	ProposalID   string    `json:"proposal_id"`
	ReviewedBy   string    `json:"reviewed_by"`
	ReviewReason string    `json:"review_reason"`
	ReviewedAt   time.Time `json:"reviewed_at,omitempty"`
}

type SkillLearningProposalQuery struct {
	WorkspaceID string                      `json:"workspace_id,omitempty"`
	Status      SkillLearningProposalStatus `json:"status,omitempty"`
	Limit       int                         `json:"limit,omitempty"`
}

type SkillLearningProposalRef struct {
	ProposalID  string                      `json:"proposal_id"`
	WorkspaceID string                      `json:"workspace_id"`
	SkillName   string                      `json:"skill_name"`
	Status      SkillLearningProposalStatus `json:"status"`
}

type SkillLearningProposal struct {
	ID           int64                       `json:"id"`
	ProposalID   string                      `json:"proposal_id"`
	WorkspaceID  string                      `json:"workspace_id"`
	SkillName    string                      `json:"skill_name"`
	SourceTask   string                      `json:"source_task"`
	DraftBody    string                      `json:"draft_body"`
	EvidenceJSON json.RawMessage             `json:"evidence_json"`
	Status       SkillLearningProposalStatus `json:"status"`
	CreatedBy    string                      `json:"created_by"`
	ReviewedBy   string                      `json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time                  `json:"reviewed_at,omitempty"`
	ReviewReason string                      `json:"review_reason,omitempty"`
	CreatedAt    time.Time                   `json:"created_at"`
}

type SkillLearningProposalList struct {
	Items []SkillLearningProposal `json:"items"`
	Count int                     `json:"count"`
}

func SubmitSkillLearningProposal(ctx context.Context, db *sql.DB, p SkillLearningProposalCreateParams) (SkillLearningProposalRef, error) {
	if err := ctx.Err(); err != nil {
		return SkillLearningProposalRef{}, err
	}
	if db == nil {
		return SkillLearningProposalRef{}, fmt.Errorf("goncho: nil db")
	}
	if err := ensureSkillLearningProposalTable(ctx, db); err != nil {
		return SkillLearningProposalRef{}, err
	}
	workspaceID := strings.TrimSpace(p.WorkspaceID)
	if workspaceID == "" {
		workspaceID = workspacepkg.DefaultWorkspaceID
	}
	skillName := strings.TrimSpace(p.SkillName)
	sourceTask := strings.TrimSpace(p.SourceTask)
	draftBody := strings.TrimSpace(p.DraftBody)
	createdBy := strings.TrimSpace(p.CreatedBy)
	if skillName == "" || sourceTask == "" || draftBody == "" || createdBy == "" {
		return SkillLearningProposalRef{}, fmt.Errorf("goncho: skill learning proposal requires skill_name, source_task, draft_body, and created_by")
	}
	evidence := p.Evidence
	if evidence == nil {
		evidence = map[string]any{}
	}
	evidenceJSON, err := json.Marshal(evidence)
	if err != nil {
		return SkillLearningProposalRef{}, fmt.Errorf("goncho: marshal skill learning proposal evidence: %w", err)
	}
	createdAt := p.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	proposalID := fmt.Sprintf("skill_prop_%d", createdAt.UnixNano())
	_, err = db.ExecContext(ctx, `
		INSERT INTO goncho_skill_learning_proposals(
			proposal_id, workspace_id, skill_name, source_task, draft_body, evidence_json,
			status, created_by, reviewed_by, reviewed_at, review_reason, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '', NULL, '', ?)
	`, proposalID, workspaceID, skillName, sourceTask, draftBody, string(evidenceJSON), string(SkillLearningProposalPending), createdBy, createdAt.UnixNano())
	if err != nil {
		return SkillLearningProposalRef{}, fmt.Errorf("goncho: submit skill learning proposal: %w", err)
	}
	return SkillLearningProposalRef{ProposalID: proposalID, WorkspaceID: workspaceID, SkillName: skillName, Status: SkillLearningProposalPending}, nil
}

func GetSkillLearningProposal(ctx context.Context, db *sql.DB, proposalID string) (SkillLearningProposal, error) {
	if err := ctx.Err(); err != nil {
		return SkillLearningProposal{}, err
	}
	if db == nil {
		return SkillLearningProposal{}, fmt.Errorf("goncho: nil db")
	}
	if err := ensureSkillLearningProposalTable(ctx, db); err != nil {
		return SkillLearningProposal{}, err
	}
	id := strings.TrimSpace(proposalID)
	if id == "" {
		return SkillLearningProposal{}, fmt.Errorf("goncho: proposal_id is required")
	}
	return getSkillLearningProposal(ctx, db, id)
}

func ListSkillLearningProposals(ctx context.Context, db *sql.DB, q SkillLearningProposalQuery) (SkillLearningProposalList, error) {
	if err := ctx.Err(); err != nil {
		return SkillLearningProposalList{}, err
	}
	if db == nil {
		return SkillLearningProposalList{}, fmt.Errorf("goncho: nil db")
	}
	if err := ensureSkillLearningProposalTable(ctx, db); err != nil {
		return SkillLearningProposalList{}, err
	}
	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	args := []any{}
	where := []string{}
	if workspaceID := strings.TrimSpace(q.WorkspaceID); workspaceID != "" {
		where = append(where, "workspace_id = ?")
		args = append(args, workspaceID)
	}
	if q.Status != "" {
		if !validSkillLearningProposalStatus(q.Status) {
			return SkillLearningProposalList{}, fmt.Errorf("goncho: invalid skill learning proposal status %q", q.Status)
		}
		where = append(where, "status = ?")
		args = append(args, string(q.Status))
	}
	query := `
		SELECT id, proposal_id, workspace_id, skill_name, source_task, draft_body, evidence_json,
		       status, created_by, reviewed_by, reviewed_at, review_reason, created_at
		FROM goncho_skill_learning_proposals`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC, proposal_id DESC LIMIT ?"
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return SkillLearningProposalList{}, fmt.Errorf("goncho: list skill learning proposals: %w", err)
	}
	defer rows.Close()
	items := []SkillLearningProposal{}
	for rows.Next() {
		item, err := scanSkillLearningProposal(rows)
		if err != nil {
			return SkillLearningProposalList{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return SkillLearningProposalList{}, fmt.Errorf("goncho: skill learning proposal rows: %w", err)
	}
	return SkillLearningProposalList{Items: items, Count: len(items)}, nil
}

func ApproveSkillLearningProposal(ctx context.Context, db *sql.DB, p SkillLearningProposalReviewParams) (SkillLearningProposal, error) {
	return reviewSkillLearningProposal(ctx, db, p, SkillLearningProposalApproved)
}

func RejectSkillLearningProposal(ctx context.Context, db *sql.DB, p SkillLearningProposalReviewParams) (SkillLearningProposal, error) {
	return reviewSkillLearningProposal(ctx, db, p, SkillLearningProposalRejected)
}

func reviewSkillLearningProposal(ctx context.Context, db *sql.DB, p SkillLearningProposalReviewParams, status SkillLearningProposalStatus) (SkillLearningProposal, error) {
	if err := ctx.Err(); err != nil {
		return SkillLearningProposal{}, err
	}
	if db == nil {
		return SkillLearningProposal{}, fmt.Errorf("goncho: nil db")
	}
	if err := ensureSkillLearningProposalTable(ctx, db); err != nil {
		return SkillLearningProposal{}, err
	}
	proposalID := strings.TrimSpace(p.ProposalID)
	reviewedBy := strings.TrimSpace(p.ReviewedBy)
	reason := strings.TrimSpace(p.ReviewReason)
	if proposalID == "" || reviewedBy == "" || reason == "" || !validSkillLearningProposalStatus(status) || status == SkillLearningProposalPending {
		return SkillLearningProposal{}, fmt.Errorf("goncho: review skill learning proposal requires proposal_id, reviewed_by, review_reason, and final status")
	}
	reviewedAt := p.ReviewedAt.UTC()
	if reviewedAt.IsZero() {
		reviewedAt = time.Now().UTC()
	}
	result, err := db.ExecContext(ctx, `
		UPDATE goncho_skill_learning_proposals
		SET status = ?, reviewed_by = ?, reviewed_at = ?, review_reason = ?
		WHERE proposal_id = ? AND status = ?
	`, string(status), reviewedBy, reviewedAt.UnixNano(), reason, proposalID, string(SkillLearningProposalPending))
	if err != nil {
		return SkillLearningProposal{}, fmt.Errorf("goncho: review skill learning proposal: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return SkillLearningProposal{}, fmt.Errorf("goncho: review skill learning proposal rows: %w", err)
	}
	if rows == 0 {
		return SkillLearningProposal{}, fmt.Errorf("goncho: skill learning proposal %q not found or not pending", proposalID)
	}
	return getSkillLearningProposal(ctx, db, proposalID)
}

func getSkillLearningProposal(ctx context.Context, db *sql.DB, proposalID string) (SkillLearningProposal, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, proposal_id, workspace_id, skill_name, source_task, draft_body, evidence_json,
		       status, created_by, reviewed_by, reviewed_at, review_reason, created_at
		FROM goncho_skill_learning_proposals
		WHERE proposal_id = ?
	`, proposalID)
	item, err := scanSkillLearningProposal(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return SkillLearningProposal{}, fmt.Errorf("goncho: skill learning proposal %q not found", proposalID)
		}
		return SkillLearningProposal{}, err
	}
	return item, nil
}

type skillLearningProposalScanner interface {
	Scan(dest ...any) error
}

func scanSkillLearningProposal(scanner skillLearningProposalScanner) (SkillLearningProposal, error) {
	var item SkillLearningProposal
	var evidenceRaw string
	var status string
	var reviewedAt sql.NullInt64
	var createdAt int64
	if err := scanner.Scan(
		&item.ID, &item.ProposalID, &item.WorkspaceID, &item.SkillName, &item.SourceTask, &item.DraftBody, &evidenceRaw,
		&status, &item.CreatedBy, &item.ReviewedBy, &reviewedAt, &item.ReviewReason, &createdAt,
	); err != nil {
		return SkillLearningProposal{}, fmt.Errorf("goncho: scan skill learning proposal: %w", err)
	}
	item.EvidenceJSON = json.RawMessage(evidenceRaw)
	item.Status = SkillLearningProposalStatus(status)
	item.CreatedAt = time.Unix(0, createdAt).UTC()
	if reviewedAt.Valid {
		t := time.Unix(0, reviewedAt.Int64).UTC()
		item.ReviewedAt = &t
	}
	return item, nil
}

func validSkillLearningProposalStatus(status SkillLearningProposalStatus) bool {
	switch status {
	case SkillLearningProposalPending, SkillLearningProposalApproved, SkillLearningProposalRejected:
		return true
	default:
		return false
	}
}

func ensureSkillLearningProposalTable(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("goncho: nil db")
	}
	for _, stmt := range DDL {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("goncho: ensure skill learning proposal table: %w", err)
		}
	}
	return nil
}

var DDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_skill_learning_proposals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		proposal_id TEXT NOT NULL UNIQUE,
		workspace_id TEXT NOT NULL DEFAULT '',
		skill_name TEXT NOT NULL,
		source_task TEXT NOT NULL,
		draft_body TEXT NOT NULL,
		evidence_json TEXT NOT NULL DEFAULT '{}',
		status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected')),
		created_by TEXT NOT NULL,
		reviewed_by TEXT NOT NULL DEFAULT '',
		reviewed_at INTEGER,
		review_reason TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_skill_learning_workspace_status ON goncho_skill_learning_proposals(workspace_id, status, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_skill_learning_skill_status ON goncho_skill_learning_proposals(skill_name, status, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_skill_learning_created_at ON goncho_skill_learning_proposals(created_at DESC)`,
}
