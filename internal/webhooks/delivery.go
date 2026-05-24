package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	defaultWebhookDeliveryMaxAttempts = 3
	defaultWebhookDeliveryBackoff     = 30 * time.Second
)

type WebhookDeliveryStatus string

const (
	WebhookDeliveryDelivered        WebhookDeliveryStatus = "delivered"
	WebhookDeliveryRetryable        WebhookDeliveryStatus = "retryable"
	WebhookDeliveryFailed           WebhookDeliveryStatus = "failed"
	WebhookDeliveryEndpointDisabled WebhookDeliveryStatus = "endpoint_disabled"
	WebhookDeliverySkipped          WebhookDeliveryStatus = "skipped"
)

type WebhookDeliveryErrorClass string

const (
	WebhookDeliveryErrorNone       WebhookDeliveryErrorClass = ""
	WebhookDeliveryErrorHTTPStatus WebhookDeliveryErrorClass = "http_status"
	WebhookDeliveryErrorNetwork    WebhookDeliveryErrorClass = "network"
	WebhookDeliveryErrorSigning    WebhookDeliveryErrorClass = "signing"
	WebhookDeliveryErrorStore      WebhookDeliveryErrorClass = "store"
	WebhookDeliveryErrorDisabled   WebhookDeliveryErrorClass = "endpoint_disabled"
)

type WebhookDeliveryEndpoint struct {
	ID             string
	WorkspaceID    string
	URL            string
	Disabled       bool
	DisabledReason string
}

type WebhookDeliveryStore interface {
	ListWebhookDeliveryEndpoints(ctx context.Context, workspaceID string) ([]WebhookDeliveryEndpoint, error)
	RecordWebhookDelivery(ctx context.Context, attempt WebhookDeliveryAttempt) error
	DisableWebhookDeliveryEndpoint(ctx context.Context, endpoint WebhookDeliveryEndpoint, reason string, now time.Time) error
}

type WebhookHTTPClient interface {
	PostWebhook(ctx context.Context, req WebhookHTTPRequest) (WebhookHTTPResponse, error)
}

type WebhookClock interface {
	Now() time.Time
}

type WebhookDeliveryWorker struct {
	Store       WebhookDeliveryStore
	Client      WebhookHTTPClient
	Clock       WebhookClock
	Secret      string
	MaxAttempts int
	Backoff     func(attempt int) time.Duration
}

type WebhookDeliveryRequest struct {
	WorkspaceID string
	Event       WebhookEvent
	Attempt     int
}

type WebhookHTTPRequest struct {
	URL     string
	Body    string
	Headers map[string]string
}

type WebhookHTTPResponse struct {
	StatusCode int
}

type WebhookDeliveryAttempt struct {
	EndpointID  string
	WorkspaceID string
	EventType   WebhookEventType
	Attempt     int
	Status      WebhookDeliveryStatus
	StatusCode  int
	ErrorClass  WebhookDeliveryErrorClass
	Error       string
	Retry       bool
	NextRetryAt *time.Time
	Evidence    WebhookDeliveryEvidence
	RecordedAt  time.Time
}

type WebhookDeliveryResult struct {
	EndpointID  string
	WorkspaceID string
	EventType   WebhookEventType
	Attempt     int
	Status      WebhookDeliveryStatus
	StatusCode  int
	ErrorClass  WebhookDeliveryErrorClass
	Error       string
	Retry       bool
	NextRetryAt *time.Time
	Evidence    WebhookDeliveryEvidence
}

type WebhookDeliveryEvidence struct {
	EndpointID  string
	EndpointURL string
	WorkspaceID string
	EventType   WebhookEventType
	Status      WebhookDeliveryStatus
	StatusCode  int
	ErrorClass  WebhookDeliveryErrorClass
	Attempt     int
	NextRetryAt *time.Time
}

type systemWebhookClock struct{}

func (systemWebhookClock) Now() time.Time {
	return time.Now().UTC()
}

func (w WebhookDeliveryWorker) Deliver(ctx context.Context, req WebhookDeliveryRequest) ([]WebhookDeliveryResult, error) {
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(req.Event.WorkspaceID)
	}
	if workspaceID == "" {
		return nil, ErrWebhookWorkspaceRequired
	}
	if w.Store == nil {
		return nil, errors.New("goncho: webhook delivery store is required")
	}
	if w.Client == nil {
		return nil, errors.New("goncho: webhook http client is required")
	}
	now := w.now()
	attempt := req.Attempt
	if attempt <= 0 {
		attempt = 1
	}
	maxAttempts := w.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultWebhookDeliveryMaxAttempts
	}

	endpoints, err := w.Store.ListWebhookDeliveryEndpoints(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("goncho: list webhook delivery endpoints: %w", err)
	}
	if len(endpoints) == 0 {
		result := w.result(WebhookDeliveryEndpoint{
			WorkspaceID: workspaceID,
		}, req.Event.Type, attempt, WebhookDeliverySkipped, 0, WebhookDeliveryErrorNone, "no webhook endpoints", false, nil, now)
		if err := w.record(ctx, result, now); err != nil {
			return []WebhookDeliveryResult{result}, err
		}
		return []WebhookDeliveryResult{result}, nil
	}

	body, err := buildWebhookDeliveryPayload(req.Event, now)
	if err != nil {
		return nil, err
	}
	signature, err := SignWebhookPayload(body, w.Secret)
	if err != nil {
		return []WebhookDeliveryResult{w.result(WebhookDeliveryEndpoint{
			WorkspaceID: workspaceID,
		}, req.Event.Type, attempt, WebhookDeliveryFailed, 0, WebhookDeliveryErrorSigning, err.Error(), false, nil, now)}, nil
	}

	results := make([]WebhookDeliveryResult, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint.WorkspaceID == "" {
			endpoint.WorkspaceID = workspaceID
		}
		if endpoint.Disabled {
			result := w.result(endpoint, req.Event.Type, attempt, WebhookDeliveryEndpointDisabled, 0, WebhookDeliveryErrorDisabled, endpoint.DisabledReason, false, nil, now)
			results = append(results, result)
			if err := w.record(ctx, result, now); err != nil {
				return results, err
			}
			continue
		}

		httpReq := WebhookHTTPRequest{
			URL:  endpoint.URL,
			Body: body,
			Headers: map[string]string{
				"Content-Type":       "application/json",
				"X-Honcho-Signature": signature,
			},
		}
		httpResp, err := w.Client.PostWebhook(ctx, httpReq)
		result := w.classify(endpoint, req.Event.Type, attempt, maxAttempts, httpResp, err, now)
		results = append(results, result)
		if err := w.record(ctx, result, now); err != nil {
			return results, err
		}
		if result.Status == WebhookDeliveryFailed {
			if disableErr := w.Store.DisableWebhookDeliveryEndpoint(ctx, endpoint, failureDisableReason(result), now); disableErr != nil {
				return results, fmt.Errorf("goncho: disable webhook endpoint: %w", disableErr)
			}
		}
	}
	return results, nil
}

func (w WebhookDeliveryWorker) classify(endpoint WebhookDeliveryEndpoint, eventType WebhookEventType, attempt, maxAttempts int, response WebhookHTTPResponse, err error, now time.Time) WebhookDeliveryResult {
	if err != nil {
		return w.failureOrRetry(endpoint, eventType, attempt, maxAttempts, 0, WebhookDeliveryErrorNetwork, err.Error(), now)
	}
	statusCode := response.StatusCode
	if statusCode >= 200 && statusCode < 300 {
		return w.result(endpoint, eventType, attempt, WebhookDeliveryDelivered, statusCode, WebhookDeliveryErrorNone, "", false, nil, now)
	}
	if retryableWebhookStatus(statusCode) {
		return w.failureOrRetry(endpoint, eventType, attempt, maxAttempts, statusCode, WebhookDeliveryErrorHTTPStatus, fmt.Sprintf("status %d", statusCode), now)
	}
	return w.result(endpoint, eventType, attempt, WebhookDeliveryFailed, statusCode, WebhookDeliveryErrorHTTPStatus, fmt.Sprintf("status %d", statusCode), false, nil, now)
}

func (w WebhookDeliveryWorker) failureOrRetry(endpoint WebhookDeliveryEndpoint, eventType WebhookEventType, attempt, maxAttempts, statusCode int, class WebhookDeliveryErrorClass, message string, now time.Time) WebhookDeliveryResult {
	if attempt >= maxAttempts {
		return w.result(endpoint, eventType, attempt, WebhookDeliveryFailed, statusCode, class, message, false, nil, now)
	}
	next := now.Add(w.backoff(attempt))
	return w.result(endpoint, eventType, attempt, WebhookDeliveryRetryable, statusCode, class, message, true, &next, now)
}

func (w WebhookDeliveryWorker) result(endpoint WebhookDeliveryEndpoint, eventType WebhookEventType, attempt int, status WebhookDeliveryStatus, statusCode int, class WebhookDeliveryErrorClass, message string, retry bool, nextRetryAt *time.Time, now time.Time) WebhookDeliveryResult {
	evidence := WebhookDeliveryEvidence{
		EndpointID:  endpoint.ID,
		EndpointURL: redactWebhookEndpointURL(endpoint.URL),
		WorkspaceID: endpoint.WorkspaceID,
		EventType:   eventType,
		Status:      status,
		StatusCode:  statusCode,
		ErrorClass:  class,
		Attempt:     attempt,
		NextRetryAt: nextRetryAt,
	}
	return WebhookDeliveryResult{
		EndpointID:  endpoint.ID,
		WorkspaceID: endpoint.WorkspaceID,
		EventType:   eventType,
		Attempt:     attempt,
		Status:      status,
		StatusCode:  statusCode,
		ErrorClass:  class,
		Error:       message,
		Retry:       retry,
		NextRetryAt: nextRetryAt,
		Evidence:    evidence,
	}
}

func (w WebhookDeliveryWorker) record(ctx context.Context, result WebhookDeliveryResult, now time.Time) error {
	if w.Store == nil {
		return nil
	}
	attempt := WebhookDeliveryAttempt{
		EndpointID:  result.EndpointID,
		WorkspaceID: result.WorkspaceID,
		EventType:   result.EventType,
		Attempt:     result.Attempt,
		Status:      result.Status,
		StatusCode:  result.StatusCode,
		ErrorClass:  result.ErrorClass,
		Error:       result.Error,
		Retry:       result.Retry,
		NextRetryAt: result.NextRetryAt,
		Evidence:    result.Evidence,
		RecordedAt:  now,
	}
	if err := w.Store.RecordWebhookDelivery(ctx, attempt); err != nil {
		return fmt.Errorf("goncho: record webhook delivery: %w", err)
	}
	return nil
}

func (w WebhookDeliveryWorker) now() time.Time {
	if w.Clock == nil {
		return systemWebhookClock{}.Now()
	}
	return w.Clock.Now().UTC()
}

func (w WebhookDeliveryWorker) backoff(attempt int) time.Duration {
	if w.Backoff != nil {
		return w.Backoff(attempt)
	}
	if attempt <= 0 {
		attempt = 1
	}
	delay := defaultWebhookDeliveryBackoff
	for i := 1; i < attempt; i++ {
		delay *= 2
	}
	return delay
}

func retryableWebhookStatus(statusCode int) bool {
	return statusCode == 408 || statusCode == 429 || statusCode >= 500
}

func failureDisableReason(result WebhookDeliveryResult) string {
	if result.Attempt > 0 && result.ErrorClass == WebhookDeliveryErrorNetwork {
		return "max_attempts_exhausted"
	}
	if retryableWebhookStatus(result.StatusCode) {
		return "max_attempts_exhausted"
	}
	return "permanent_failure"
}

func buildWebhookDeliveryPayload(event WebhookEvent, now time.Time) (string, error) {
	if strings.TrimSpace(event.WorkspaceID) == "" {
		return "", ErrWebhookWorkspaceRequired
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

func redactWebhookEndpointURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "<redacted>"
	}
	parsed.User = nil
	if parsed.RawQuery != "" || parsed.ForceQuery {
		parsed.RawQuery = "<redacted>"
	}
	parsed.Fragment = ""
	return parsed.String()
}
