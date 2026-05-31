package events

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrWorkspaceRequired reports a missing webhook event workspace.
var ErrWorkspaceRequired = errors.New("goncho: workspace_id is required")

// Type identifies a webhook event contract.
type Type string

const (
	QueueEmpty Type = "queue.empty"
	Test       Type = "test.event"
)

// Event is the shared webhook event envelope delivered to endpoints.
type Event struct {
	Type        Type           `json:"type"`
	WorkspaceID string         `json:"workspace_id"`
	Data        map[string]any `json:"data,omitempty"`
}

// QueueEmptyParams carries queue-empty webhook event details.
type QueueEmptyParams struct {
	WorkspaceID string
	QueueType   string
	SessionID   string
	Observer    string
	Observed    string
}

// NewTest returns a test webhook event for a workspace.
func NewTest(workspaceID string) (Event, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return Event{}, ErrWorkspaceRequired
	}
	return Event{
		Type:        Test,
		WorkspaceID: workspaceID,
		Data:        map[string]any{"workspace_id": workspaceID},
	}, nil
}

// NewQueueEmpty returns a queue-empty webhook event for a workspace.
func NewQueueEmpty(params QueueEmptyParams) (Event, error) {
	workspaceID := strings.TrimSpace(params.WorkspaceID)
	if workspaceID == "" {
		return Event{}, ErrWorkspaceRequired
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
	return Event{
		Type:        QueueEmpty,
		WorkspaceID: workspaceID,
		Data:        data,
	}, nil
}

// Payload renders the event into the signed webhook delivery payload.
func Payload(event Event, now time.Time) (string, error) {
	if strings.TrimSpace(event.WorkspaceID) == "" {
		return "", ErrWorkspaceRequired
	}
	if event.Type == "" {
		return "", errors.New("goncho: webhook event type is required")
	}
	payload := map[string]any{
		"type":      string(event.Type),
		"data":      event.Data,
		"timestamp": now.UTC().Format(time.RFC3339),
	}
	if payload["data"] == nil {
		payload["data"] = map[string]any{"workspace_id": event.WorkspaceID}
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		return "", fmt.Errorf("goncho: encode webhook payload: %w", err)
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}
