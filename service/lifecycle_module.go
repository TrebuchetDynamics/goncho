package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/sqlutil"
)

type lifecycleModule struct {
	db             *sql.DB
	workspaceID    string
	maxMessageSize int
}

func (s *Service) lifecycle() lifecycleModule {
	return lifecycleModule{
		db:             s.db,
		workspaceID:    s.workspaceID,
		maxMessageSize: s.maxMessageSize,
	}
}

func (l lifecycleModule) CreateMessages(ctx context.Context, params CreateMessagesParams) (CreateMessagesResult, error) {
	sessionKey := strings.TrimSpace(params.SessionKey)
	if sessionKey == "" {
		return CreateMessagesResult{}, fmt.Errorf("goncho: session_key is required")
	}

	var lastErr error
	for attempt := 0; attempt < createMessagesLockRetryAttempts; attempt++ {
		result, err := l.createMessagesOnce(ctx, sessionKey, params.Messages)
		if err == nil {
			return result, nil
		}
		if !isTransientSQLiteLockError(err) {
			return CreateMessagesResult{}, err
		}
		lastErr = err
		if attempt == createMessagesLockRetryAttempts-1 {
			break
		}
		if err := waitCreateMessagesLockRetry(ctx, attempt); err != nil {
			return CreateMessagesResult{}, fmt.Errorf("goncho: create messages retry canceled: %w; last error: %v", err, lastErr)
		}
	}
	return CreateMessagesResult{}, lastErr
}

func (l lifecycleModule) createMessagesOnce(ctx context.Context, sessionKey string, inputs []CreateMessage) (CreateMessagesResult, error) {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return CreateMessagesResult{}, fmt.Errorf("goncho: begin create messages: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	messages, err := createLifecycleMessages(ctx, tx, l.workspaceID, sessionKey, l.maxMessageSize, inputs)
	if err != nil {
		return CreateMessagesResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return CreateMessagesResult{}, fmt.Errorf("goncho: commit create messages: %w", err)
	}
	committed = true
	return CreateMessagesResult{
		WorkspaceID: l.workspaceID,
		SessionKey:  sessionKey,
		Messages:    messages,
	}, nil
}

func (l lifecycleModule) DeleteSession(ctx context.Context, sessionKey string) (SessionDeletionResult, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return SessionDeletionResult{}, fmt.Errorf("goncho: session_key is required")
	}
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return SessionDeletionResult{}, fmt.Errorf("goncho: begin delete session: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	result, err := deleteLifecycleSession(ctx, tx, l.workspaceID, sessionKey)
	if err != nil {
		return SessionDeletionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return SessionDeletionResult{}, fmt.Errorf("goncho: commit delete session: %w", err)
	}
	committed = true
	return result, nil
}

func (l lifecycleModule) DeleteWorkspace(ctx context.Context) (WorkspaceDeletionResult, error) {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return WorkspaceDeletionResult{}, fmt.Errorf("goncho: begin delete workspace: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	result, err := deleteLifecycleWorkspace(ctx, tx, l.workspaceID)
	if err != nil {
		return WorkspaceDeletionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return WorkspaceDeletionResult{}, fmt.Errorf("goncho: commit delete workspace: %w", err)
	}
	committed = true
	return result, nil
}

func waitCreateMessagesLockRetry(ctx context.Context, attempt int) error {
	timer := time.NewTimer(createMessagesLockRetryDelay(attempt))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func createMessagesLockRetryDelay(attempt int) time.Duration {
	delay := createMessagesLockRetryMin * time.Duration(attempt+1)
	if delay > createMessagesLockRetryMax {
		return createMessagesLockRetryMax
	}
	return delay
}

func isTransientSQLiteLockError(err error) bool {
	return sqlutil.IsSQLiteTransientLockError(err)
}
