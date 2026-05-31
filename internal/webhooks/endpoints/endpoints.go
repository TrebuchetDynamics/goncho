package endpoints

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/webhooks/urlpolicy"
	"github.com/TrebuchetDynamics/goncho/internal/webhooks/workspace"
)

const (
	DefaultWorkspaceLimit = 10
	MaxURLLength          = 2048
)

var (
	ErrWorkspaceRequired = workspace.ErrRequired
	ErrInvalidURL        = errors.New("goncho: invalid webhook url")
	ErrLimitReached      = errors.New("goncho: maximum webhook endpoints reached for workspace")
	ErrNotFound          = errors.New("goncho: webhook endpoint not found")
)

// CreateParams carries endpoint creation inputs at the storage boundary.
type CreateParams struct {
	WorkspaceID string
	URL         string
	Limit       int
	Now         time.Time
}

// CreateResult reports whether endpoint creation inserted a new row or reused an existing row.
type CreateResult struct {
	Endpoint Endpoint `json:"endpoint"`
	Created  bool     `json:"created"`
}

// Endpoint is a registered webhook destination.
type Endpoint struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
}

func GetOrCreate(ctx context.Context, db *sql.DB, defaultWorkspaceID string, params CreateParams) (CreateResult, error) {
	workspaceID := workspace.Resolve(params.WorkspaceID, defaultWorkspaceID)
	endpointURL, err := NormalizeURL(params.URL)
	if err != nil {
		return CreateResult{}, err
	}
	limit := params.Limit
	if limit <= 0 {
		limit = DefaultWorkspaceLimit
	}
	now := params.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := EnsureTable(ctx, db); err != nil {
		return CreateResult{}, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return CreateResult{}, fmt.Errorf("goncho: begin webhook create: %w", err)
	}
	defer tx.Rollback()

	if existing, ok, err := findByURL(ctx, tx, workspaceID, endpointURL); err != nil {
		return CreateResult{}, err
	} else if ok {
		return CreateResult{Endpoint: existing, Created: false}, nil
	}
	count, err := count(ctx, tx, workspaceID)
	if err != nil {
		return CreateResult{}, err
	}
	if count >= limit {
		return CreateResult{}, ErrLimitReached
	}
	endpoint := Endpoint{
		ID:          newID(),
		WorkspaceID: workspaceID,
		URL:         endpointURL,
		CreatedAt:   now,
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO goncho_webhook_endpoints(id, workspace_id, url, created_at)
		VALUES(?, ?, ?, ?)
	`, endpoint.ID, endpoint.WorkspaceID, endpoint.URL, endpoint.CreatedAt.Unix()); err != nil {
		return CreateResult{}, fmt.Errorf("goncho: insert webhook endpoint: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return CreateResult{}, fmt.Errorf("goncho: commit webhook create: %w", err)
	}
	return CreateResult{Endpoint: endpoint, Created: true}, nil
}

func List(ctx context.Context, db *sql.DB, defaultWorkspaceID, workspaceID string) ([]Endpoint, error) {
	workspaceID = workspace.Resolve(workspaceID, defaultWorkspaceID)
	if workspaceID == "" {
		return nil, ErrWorkspaceRequired
	}
	if err := EnsureTable(ctx, db); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, workspace_id, url, created_at
		FROM goncho_webhook_endpoints
		WHERE workspace_id = ?
		ORDER BY created_at ASC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("goncho: list webhook endpoints: %w", err)
	}
	defer rows.Close()

	var out []Endpoint
	for rows.Next() {
		endpoint, err := Scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate webhook endpoints: %w", err)
	}
	return out, nil
}

func Delete(ctx context.Context, db *sql.DB, defaultWorkspaceID, workspaceID, endpointID string) error {
	workspaceID = workspace.Resolve(workspaceID, defaultWorkspaceID)
	endpointID = strings.TrimSpace(endpointID)
	if workspaceID == "" {
		return ErrWorkspaceRequired
	}
	if endpointID == "" {
		return ErrNotFound
	}
	if err := EnsureTable(ctx, db); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `
		DELETE FROM goncho_webhook_endpoints
		WHERE workspace_id = ? AND id = ?
	`, workspaceID, endpointID)
	if err != nil {
		return fmt.Errorf("goncho: delete webhook endpoint: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("goncho: delete webhook endpoint rows: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func EnsureTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS goncho_webhook_endpoints (
			id           TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			url          TEXT NOT NULL CHECK(length(url) <= 2048),
			created_at   INTEGER NOT NULL,
			UNIQUE(workspace_id, url)
		);
		CREATE INDEX IF NOT EXISTS idx_goncho_webhook_endpoints_workspace
			ON goncho_webhook_endpoints(workspace_id, created_at);
	`)
	if err != nil {
		return fmt.Errorf("goncho: ensure webhook endpoint table: %w", err)
	}
	return nil
}

func NormalizeURL(raw string) (string, error) {
	endpointURL, err := urlpolicy.NormalizeEndpoint(raw, MaxURLLength)
	if errors.Is(err, urlpolicy.ErrInvalid) {
		return "", ErrInvalidURL
	}
	return endpointURL, err
}

type Scanner interface {
	Scan(...any) error
}

func Scan(row Scanner) (Endpoint, error) {
	var endpoint Endpoint
	var createdAt int64
	if err := row.Scan(&endpoint.ID, &endpoint.WorkspaceID, &endpoint.URL, &createdAt); err != nil {
		return Endpoint{}, fmt.Errorf("goncho: scan webhook endpoint: %w", err)
	}
	endpoint.CreatedAt = time.Unix(createdAt, 0).UTC()
	return endpoint, nil
}

type sqlRunner interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func findByURL(ctx context.Context, db sqlRunner, workspaceID, endpointURL string) (Endpoint, bool, error) {
	endpoint, err := Scan(db.QueryRowContext(ctx, `
		SELECT id, workspace_id, url, created_at
		FROM goncho_webhook_endpoints
		WHERE workspace_id = ? AND url = ?
	`, workspaceID, endpointURL))
	if err == nil {
		return endpoint, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return Endpoint{}, false, nil
	}
	return Endpoint{}, false, err
}

func count(ctx context.Context, db sqlRunner, workspaceID string) (int, error) {
	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM goncho_webhook_endpoints
		WHERE workspace_id = ?
	`, workspaceID).Scan(&count); err != nil {
		return 0, fmt.Errorf("goncho: count webhook endpoints: %w", err)
	}
	return count, nil
}

func newID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		sum := sha256.Sum256([]byte(time.Now().Format(time.RFC3339Nano)))
		return "we_" + hex.EncodeToString(sum[:12])
	}
	return "we_" + hex.EncodeToString(b[:])
}
