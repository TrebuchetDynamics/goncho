package delivery

import (
	"context"
	"errors"
	"fmt"
	"time"

	eventcontract "github.com/TrebuchetDynamics/goncho/internal/webhooks/events"
	"github.com/TrebuchetDynamics/goncho/internal/webhooks/signing"
	"github.com/TrebuchetDynamics/goncho/internal/webhooks/urlpolicy"
	"github.com/TrebuchetDynamics/goncho/internal/webhooks/workspace"
)

const (
	DefaultMaxAttempts = 3
	DefaultBackoff     = 30 * time.Second
)

var ErrWorkspaceRequired = workspace.ErrRequired

type Status string

const (
	Delivered        Status = "delivered"
	Retryable        Status = "retryable"
	Failed           Status = "failed"
	EndpointDisabled Status = "endpoint_disabled"
	Skipped          Status = "skipped"
)

type ErrorClass string

const (
	ErrorNone       ErrorClass = ""
	ErrorHTTPStatus ErrorClass = "http_status"
	ErrorNetwork    ErrorClass = "network"
	ErrorSigning    ErrorClass = "signing"
	ErrorStore      ErrorClass = "store"
	ErrorDisabled   ErrorClass = "endpoint_disabled"
)

type Endpoint struct {
	ID             string
	WorkspaceID    string
	URL            string
	Disabled       bool
	DisabledReason string
}

type Store interface {
	ListWebhookDeliveryEndpoints(ctx context.Context, workspaceID string) ([]Endpoint, error)
	RecordWebhookDelivery(ctx context.Context, attempt Attempt) error
	DisableWebhookDeliveryEndpoint(ctx context.Context, endpoint Endpoint, reason string, now time.Time) error
}

type HTTPClient interface {
	PostWebhook(ctx context.Context, req HTTPRequest) (HTTPResponse, error)
}

type Clock interface {
	Now() time.Time
}

type Worker struct {
	Store       Store
	Client      HTTPClient
	Clock       Clock
	Secret      string
	MaxAttempts int
	Backoff     func(attempt int) time.Duration
}

type Request struct {
	WorkspaceID string
	Event       eventcontract.Event
	Attempt     int
}

type HTTPRequest struct {
	URL     string
	Body    string
	Headers map[string]string
}

type HTTPResponse struct {
	StatusCode int
}

type Attempt struct {
	EndpointID  string
	WorkspaceID string
	EventType   eventcontract.Type
	Attempt     int
	Status      Status
	StatusCode  int
	ErrorClass  ErrorClass
	Error       string
	Retry       bool
	NextRetryAt *time.Time
	Evidence    Evidence
	RecordedAt  time.Time
}

type Result struct {
	EndpointID  string
	WorkspaceID string
	EventType   eventcontract.Type
	Attempt     int
	Status      Status
	StatusCode  int
	ErrorClass  ErrorClass
	Error       string
	Retry       bool
	NextRetryAt *time.Time
	Evidence    Evidence
}

type Evidence struct {
	EndpointID  string
	EndpointURL string
	WorkspaceID string
	EventType   eventcontract.Type
	Status      Status
	StatusCode  int
	ErrorClass  ErrorClass
	Attempt     int
	NextRetryAt *time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

func (w Worker) Deliver(ctx context.Context, req Request) ([]Result, error) {
	workspaceID := workspace.Resolve(req.WorkspaceID, req.Event.WorkspaceID)
	if workspaceID == "" {
		return nil, ErrWorkspaceRequired
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
		maxAttempts = DefaultMaxAttempts
	}

	endpoints, err := w.Store.ListWebhookDeliveryEndpoints(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("goncho: list webhook delivery endpoints: %w", err)
	}
	if len(endpoints) == 0 {
		result := w.result(Endpoint{WorkspaceID: workspaceID}, req.Event.Type, attempt, Skipped, 0, ErrorNone, "no webhook endpoints", false, nil, now)
		if err := w.record(ctx, result, now); err != nil {
			return []Result{result}, err
		}
		return []Result{result}, nil
	}

	body, err := Payload(req.Event, now)
	if err != nil {
		return nil, err
	}
	signature, err := signing.SignPayload(body, w.Secret)
	if err != nil {
		return []Result{w.result(Endpoint{WorkspaceID: workspaceID}, req.Event.Type, attempt, Failed, 0, ErrorSigning, err.Error(), false, nil, now)}, nil
	}

	results := make([]Result, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint.WorkspaceID == "" {
			endpoint.WorkspaceID = workspaceID
		}
		if endpoint.Disabled {
			result := w.result(endpoint, req.Event.Type, attempt, EndpointDisabled, 0, ErrorDisabled, endpoint.DisabledReason, false, nil, now)
			results = append(results, result)
			if err := w.record(ctx, result, now); err != nil {
				return results, err
			}
			continue
		}

		httpReq := HTTPRequest{
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
		if result.Status == Failed {
			if disableErr := w.Store.DisableWebhookDeliveryEndpoint(ctx, endpoint, FailureDisableReason(result), now); disableErr != nil {
				return results, fmt.Errorf("goncho: disable webhook endpoint: %w", disableErr)
			}
		}
	}
	return results, nil
}

func (w Worker) classify(endpoint Endpoint, eventType eventcontract.Type, attempt, maxAttempts int, response HTTPResponse, err error, now time.Time) Result {
	if err != nil {
		return w.failureOrRetry(endpoint, eventType, attempt, maxAttempts, 0, ErrorNetwork, err.Error(), now)
	}
	statusCode := response.StatusCode
	if statusCode >= 200 && statusCode < 300 {
		return w.result(endpoint, eventType, attempt, Delivered, statusCode, ErrorNone, "", false, nil, now)
	}
	if RetryableStatus(statusCode) {
		return w.failureOrRetry(endpoint, eventType, attempt, maxAttempts, statusCode, ErrorHTTPStatus, fmt.Sprintf("status %d", statusCode), now)
	}
	return w.result(endpoint, eventType, attempt, Failed, statusCode, ErrorHTTPStatus, fmt.Sprintf("status %d", statusCode), false, nil, now)
}

func (w Worker) failureOrRetry(endpoint Endpoint, eventType eventcontract.Type, attempt, maxAttempts, statusCode int, class ErrorClass, message string, now time.Time) Result {
	if attempt >= maxAttempts {
		return w.result(endpoint, eventType, attempt, Failed, statusCode, class, message, false, nil, now)
	}
	next := now.Add(w.backoff(attempt))
	return w.result(endpoint, eventType, attempt, Retryable, statusCode, class, message, true, &next, now)
}

func (w Worker) result(endpoint Endpoint, eventType eventcontract.Type, attempt int, status Status, statusCode int, class ErrorClass, message string, retry bool, nextRetryAt *time.Time, now time.Time) Result {
	evidence := Evidence{
		EndpointID:  endpoint.ID,
		EndpointURL: urlpolicy.RedactEndpoint(endpoint.URL),
		WorkspaceID: endpoint.WorkspaceID,
		EventType:   eventType,
		Status:      status,
		StatusCode:  statusCode,
		ErrorClass:  class,
		Attempt:     attempt,
		NextRetryAt: nextRetryAt,
	}
	return Result{
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

func (w Worker) record(ctx context.Context, result Result, now time.Time) error {
	if w.Store == nil {
		return nil
	}
	attempt := Attempt{
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

func (w Worker) now() time.Time {
	if w.Clock == nil {
		return systemClock{}.Now()
	}
	return w.Clock.Now().UTC()
}

func (w Worker) backoff(attempt int) time.Duration {
	if w.Backoff != nil {
		return w.Backoff(attempt)
	}
	if attempt <= 0 {
		attempt = 1
	}
	delay := DefaultBackoff
	for i := 1; i < attempt; i++ {
		delay *= 2
	}
	return delay
}

func RetryableStatus(statusCode int) bool {
	return statusCode == 408 || statusCode == 429 || statusCode >= 500
}

func FailureDisableReason(result Result) string {
	if result.Attempt > 0 && result.ErrorClass == ErrorNetwork {
		return "max_attempts_exhausted"
	}
	if RetryableStatus(result.StatusCode) {
		return "max_attempts_exhausted"
	}
	return "permanent_failure"
}

func Payload(event eventcontract.Event, now time.Time) (string, error) {
	return eventcontract.Payload(event, now)
}
