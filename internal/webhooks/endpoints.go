package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultWebhookWorkspaceLimit = 10
	MaxWebhookURLLength          = 2048
)

var (
	ErrWebhookWorkspaceRequired = errors.New("goncho: workspace_id is required")
	ErrWebhookInvalidURL        = errors.New("goncho: invalid webhook url")
	ErrWebhookLimitReached      = errors.New("goncho: maximum webhook endpoints reached for workspace")
	ErrWebhookNotFound          = errors.New("goncho: webhook endpoint not found")
	ErrWebhookSecretMissing     = errors.New("goncho: webhook secret is required")
)

type WebhookEndpointCreateParams struct {
	WorkspaceID string
	URL         string
	Limit       int
	Now         time.Time
}

type WebhookEndpointCreateResult struct {
	Endpoint WebhookEndpoint `json:"endpoint"`
	Created  bool            `json:"created"`
}

type WebhookEndpoint struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
}

type WebhookEventType string

const (
	WebhookEventQueueEmpty WebhookEventType = "queue.empty"
	WebhookEventTest       WebhookEventType = "test.event"
)

type WebhookEvent struct {
	Type        WebhookEventType `json:"type"`
	WorkspaceID string           `json:"workspace_id"`
	Data        map[string]any   `json:"data,omitempty"`
}

type QueueEmptyWebhookEventParams struct {
	WorkspaceID string
	QueueType   string
	SessionID   string
	Observer    string
	Observed    string
}

func GetOrCreateEndpoint(ctx context.Context, db *sql.DB, defaultWorkspaceID string, params WebhookEndpointCreateParams) (WebhookEndpointCreateResult, error) {
	workspaceID := strings.TrimSpace(params.WorkspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(defaultWorkspaceID)
	}
	endpointURL, err := normalizeWebhookURL(params.URL)
	if err != nil {
		return WebhookEndpointCreateResult{}, err
	}
	limit := params.Limit
	if limit <= 0 {
		limit = DefaultWebhookWorkspaceLimit
	}
	now := params.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := ensureEndpointTable(ctx, db); err != nil {
		return WebhookEndpointCreateResult{}, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return WebhookEndpointCreateResult{}, fmt.Errorf("goncho: begin webhook create: %w", err)
	}
	defer tx.Rollback()

	if existing, ok, err := findEndpointByURL(ctx, tx, workspaceID, endpointURL); err != nil {
		return WebhookEndpointCreateResult{}, err
	} else if ok {
		return WebhookEndpointCreateResult{Endpoint: existing, Created: false}, nil
	}
	count, err := countEndpoints(ctx, tx, workspaceID)
	if err != nil {
		return WebhookEndpointCreateResult{}, err
	}
	if count >= limit {
		return WebhookEndpointCreateResult{}, ErrWebhookLimitReached
	}
	endpoint := WebhookEndpoint{
		ID:          newWebhookEndpointID(),
		WorkspaceID: workspaceID,
		URL:         endpointURL,
		CreatedAt:   now,
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO goncho_webhook_endpoints(id, workspace_id, url, created_at)
		VALUES(?, ?, ?, ?)
	`, endpoint.ID, endpoint.WorkspaceID, endpoint.URL, endpoint.CreatedAt.Unix()); err != nil {
		return WebhookEndpointCreateResult{}, fmt.Errorf("goncho: insert webhook endpoint: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return WebhookEndpointCreateResult{}, fmt.Errorf("goncho: commit webhook create: %w", err)
	}
	return WebhookEndpointCreateResult{Endpoint: endpoint, Created: true}, nil
}

func ListEndpoints(ctx context.Context, db *sql.DB, defaultWorkspaceID, workspaceID string) ([]WebhookEndpoint, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(defaultWorkspaceID)
	}
	if workspaceID == "" {
		return nil, ErrWebhookWorkspaceRequired
	}
	if err := ensureEndpointTable(ctx, db); err != nil {
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

	var out []WebhookEndpoint
	for rows.Next() {
		endpoint, err := scanWebhookEndpoint(rows)
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

func DeleteEndpoint(ctx context.Context, db *sql.DB, defaultWorkspaceID, workspaceID, endpointID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(defaultWorkspaceID)
	}
	endpointID = strings.TrimSpace(endpointID)
	if workspaceID == "" {
		return ErrWebhookWorkspaceRequired
	}
	if endpointID == "" {
		return ErrWebhookNotFound
	}
	if err := ensureEndpointTable(ctx, db); err != nil {
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
		return ErrWebhookNotFound
	}
	return nil
}

func NewTestWebhookEvent(workspaceID string) (WebhookEvent, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return WebhookEvent{}, ErrWebhookWorkspaceRequired
	}
	return WebhookEvent{
		Type:        WebhookEventTest,
		WorkspaceID: workspaceID,
		Data:        map[string]any{"workspace_id": workspaceID},
	}, nil
}

func NewQueueEmptyWebhookEvent(params QueueEmptyWebhookEventParams) (WebhookEvent, error) {
	workspaceID := strings.TrimSpace(params.WorkspaceID)
	if workspaceID == "" {
		return WebhookEvent{}, ErrWebhookWorkspaceRequired
	}
	queueType := strings.TrimSpace(params.QueueType)
	if queueType == "" {
		queueType = "default"
	}
	data := map[string]any{
		"workspace_id": workspaceID,
		"queue_type":   queueType,
	}
	if sessionID := strings.TrimSpace(params.SessionID); sessionID != "" {
		data["session_id"] = sessionID
	}
	if observer := strings.TrimSpace(params.Observer); observer != "" {
		data["observer"] = observer
	}
	if observed := strings.TrimSpace(params.Observed); observed != "" {
		data["observed"] = observed
	}
	return WebhookEvent{
		Type:        WebhookEventQueueEmpty,
		WorkspaceID: workspaceID,
		Data:        data,
	}, nil
}

func SignWebhookPayload(payload, secret string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", ErrWebhookSecretMissing
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func ensureEndpointTable(ctx context.Context, db *sql.DB) error {
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

func normalizeWebhookURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > MaxWebhookURLLength {
		return "", ErrWebhookInvalidURL
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrWebhookInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", ErrWebhookInvalidURL
	}
	host := parsed.Hostname()
	if host == "" || privateWebhookHost(host) {
		return "", ErrWebhookInvalidURL
	}
	return parsed.String(), nil
}

func privateWebhookHost(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

type webhookEndpointScanner interface {
	Scan(...any) error
}

func scanWebhookEndpoint(row webhookEndpointScanner) (WebhookEndpoint, error) {
	var endpoint WebhookEndpoint
	var createdAt int64
	if err := row.Scan(&endpoint.ID, &endpoint.WorkspaceID, &endpoint.URL, &createdAt); err != nil {
		return WebhookEndpoint{}, fmt.Errorf("goncho: scan webhook endpoint: %w", err)
	}
	endpoint.CreatedAt = time.Unix(createdAt, 0).UTC()
	return endpoint, nil
}

type endpointSQL interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func findEndpointByURL(ctx context.Context, db endpointSQL, workspaceID, endpointURL string) (WebhookEndpoint, bool, error) {
	endpoint, err := scanWebhookEndpoint(db.QueryRowContext(ctx, `
		SELECT id, workspace_id, url, created_at
		FROM goncho_webhook_endpoints
		WHERE workspace_id = ? AND url = ?
	`, workspaceID, endpointURL))
	if err == nil {
		return endpoint, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return WebhookEndpoint{}, false, nil
	}
	return WebhookEndpoint{}, false, err
}

func countEndpoints(ctx context.Context, db endpointSQL, workspaceID string) (int, error) {
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

func newWebhookEndpointID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		sum := sha256.Sum256([]byte(time.Now().Format(time.RFC3339Nano)))
		return "we_" + hex.EncodeToString(sum[:12])
	}
	return "we_" + hex.EncodeToString(b[:])
}
