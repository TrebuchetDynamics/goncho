package goncho

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func CreateGoal(ctx context.Context, db *sql.DB, p GoalParams) (GoalResult, error) {
	if p.Name == "" {
		return GoalResult{}, errors.New("goncho: goal name is required")
	}

	now := time.Now().UTC()
	id := fmt.Sprintf("goal_%d", now.UnixNano())

	_, err := db.ExecContext(ctx, `
		INSERT INTO goals (id, name, status, parent_id, created_at, updated_at)
		VALUES (?, ?, 'active', ?, ?, ?)
	`, id, p.Name, nullString(p.ParentID), now.Unix(), now.Unix())
	if err != nil {
		return GoalResult{}, fmt.Errorf("goncho: create goal: %w", err)
	}

	return GoalResult{Goal: Goal{
		ID: id, Name: p.Name, Status: GoalActive, ParentID: p.ParentID,
		CreatedAt: now, UpdatedAt: now,
	}}, nil
}

func CompleteGoal(ctx context.Context, db *sql.DB, id string) error {
	return setGoalStatus(ctx, db, id, GoalCompleted)
}

func ArchiveGoal(ctx context.Context, db *sql.DB, id string) error {
	return setGoalStatus(ctx, db, id, GoalArchived)
}

func setGoalStatus(ctx context.Context, db *sql.DB, id string, status GoalStatus) error {
	now := time.Now().UTC()
	res, err := db.ExecContext(ctx, `
		UPDATE goals SET status = ?, updated_at = ? WHERE id = ? AND status = 'active'
	`, status, now.Unix(), id)
	if err != nil {
		return fmt.Errorf("goncho: update goal: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("goncho: goal rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("goncho: goal %s not found or not active", id)
	}
	return nil
}

func AssembleContext(ctx context.Context, db *sql.DB, p ContextParams) (ContextResult, error) {
	if p.MaxTokens <= 0 {
		p.MaxTokens = 4000
	}

	result, err := Retrieve(ctx, db, RetrieveParams{
		Query:       "",
		PeerID:      p.PeerID,
		WorkspaceID: p.WorkspaceID,
		ContextID:   p.ContextID,
		Limit:       50,
	})
	if err != nil {
		return ContextResult{}, fmt.Errorf("goncho: retrieve for context: %w", err)
	}

	var memories []Memory
	tokenCount := 0
	for _, m := range result.Memories {
		est := estimateTokens(m.Content)
		if tokenCount+est > p.MaxTokens {
			break
		}
		memories = append(memories, m)
		tokenCount += est
	}

	return ContextResult{Memories: memories, TokenEst: tokenCount}, nil
}

func estimateTokens(content string) int {
	words := 0
	inWord := false
	for _, c := range content {
		if c == ' ' || c == '\n' || c == '\t' {
			if inWord {
				words++
				inWord = false
			}
		} else {
			inWord = true
		}
	}
	if inWord {
		words++
	}
	return words + words/4
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
