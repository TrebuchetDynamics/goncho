package webhooks_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/webhooks"
)

func TestWebhookDeliveryPostsSignedHonchoPayload(t *testing.T) {
	clock := fixedWebhookClock{now: time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC)}
	store := &recordingWebhookDeliveryStore{
		endpoints: []webhooks.WebhookDeliveryEndpoint{
			{
				ID:          "we_a",
				WorkspaceID: "workspace-a",
				URL:         "https://hooks.example/a?token=secret",
			},
		},
	}
	client := &recordingWebhookHTTPClient{
		responses: map[string]webhooks.WebhookHTTPResponse{
			"https://hooks.example/a?token=secret": {StatusCode: 202},
		},
	}
	worker := webhooks.WebhookDeliveryWorker{
		Store:       store,
		Client:      client,
		Clock:       clock,
		Secret:      "delivery-secret",
		MaxAttempts: 3,
	}

	event, err := webhooks.NewTestWebhookEvent("workspace-a")
	if err != nil {
		t.Fatal(err)
	}
	results, err := worker.Deliver(context.Background(), webhooks.WebhookDeliveryRequest{
		WorkspaceID: "workspace-a",
		Event:       event,
		Attempt:     1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	result := results[0]
	if result.Status != webhooks.WebhookDeliveryDelivered || result.StatusCode != 202 || result.Retry {
		t.Fatalf("result = %+v, want delivered 202 without retry", result)
	}
	if result.Evidence.EndpointURL != "https://hooks.example/a?<redacted>" {
		t.Fatalf("evidence url = %q, want query redacted", result.Evidence.EndpointURL)
	}
	if strings.Contains(result.Evidence.EndpointURL, "secret") {
		t.Fatalf("evidence leaked secret-bearing query: %+v", result.Evidence)
	}
	if len(client.calls) != 1 {
		t.Fatalf("client calls = %d, want 1", len(client.calls))
	}

	call := client.calls[0]
	wantPayload := `{"data":{"workspace_id":"workspace-a"},"timestamp":"2026-02-13T00:00:00Z","type":"test.event"}`
	if call.Body != wantPayload {
		t.Fatalf("payload = %q, want %q", call.Body, wantPayload)
	}
	if call.Headers["Content-Type"] != "application/json" {
		t.Fatalf("content-type = %q, want application/json", call.Headers["Content-Type"])
	}
	if call.Headers["X-Honcho-Signature"] != hmacSHA256Hex("delivery-secret", wantPayload) {
		t.Fatalf("signature = %q, want Honcho HMAC", call.Headers["X-Honcho-Signature"])
	}
	if strings.Contains(call.Headers["X-Honcho-Signature"], "delivery-secret") {
		t.Fatalf("signature header leaked secret")
	}
	if len(store.attempts) != 1 || store.attempts[0].Status != webhooks.WebhookDeliveryDelivered {
		t.Fatalf("recorded attempts = %+v, want delivered audit", store.attempts)
	}
}

func TestWebhookDeliveryQueueEmptyPayloadAndRetryBackoff(t *testing.T) {
	clock := fixedWebhookClock{now: time.Date(2026, 4, 28, 15, 30, 0, 0, time.UTC)}
	store := &recordingWebhookDeliveryStore{
		endpoints: []webhooks.WebhookDeliveryEndpoint{
			{
				ID:          "we_retry",
				WorkspaceID: "workspace-a",
				URL:         "https://hooks.example/retry",
			},
		},
	}
	client := &recordingWebhookHTTPClient{
		responses: map[string]webhooks.WebhookHTTPResponse{
			"https://hooks.example/retry": {StatusCode: 503},
		},
	}
	worker := webhooks.WebhookDeliveryWorker{
		Store:       store,
		Client:      client,
		Clock:       clock,
		Secret:      "delivery-secret",
		MaxAttempts: 3,
		Backoff: func(attempt int) time.Duration {
			if attempt != 2 {
				t.Fatalf("backoff attempt = %d, want 2", attempt)
			}
			return 2 * time.Minute
		},
	}

	event, err := webhooks.NewQueueEmptyWebhookEvent(webhooks.QueueEmptyWebhookEventParams{
		WorkspaceID: "workspace-a",
		QueueType:   "summary",
		SessionID:   "sess-1",
		Observer:    "alice",
		Observed:    "bob",
	})
	if err != nil {
		t.Fatal(err)
	}
	results, err := worker.Deliver(context.Background(), webhooks.WebhookDeliveryRequest{
		WorkspaceID: "workspace-a",
		Event:       event,
		Attempt:     2,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	result := results[0]
	if result.Status != webhooks.WebhookDeliveryRetryable || !result.Retry || result.ErrorClass != webhooks.WebhookDeliveryErrorHTTPStatus {
		t.Fatalf("result = %+v, want retryable HTTP status", result)
	}
	wantRetry := clock.now.Add(2 * time.Minute)
	if result.NextRetryAt == nil || !result.NextRetryAt.Equal(wantRetry) {
		t.Fatalf("next retry = %v, want %v", result.NextRetryAt, wantRetry)
	}
	wantPayload := `{"data":{"observed":"bob","observer":"alice","queue_type":"summary","session_id":"sess-1","workspace_id":"workspace-a"},"timestamp":"2026-04-28T15:30:00Z","type":"queue.empty"}`
	if client.calls[0].Body != wantPayload {
		t.Fatalf("payload = %q, want %q", client.calls[0].Body, wantPayload)
	}
	if len(store.disabled) != 0 {
		t.Fatalf("disabled = %+v, want no endpoint disabled while attempts remain", store.disabled)
	}
	if len(store.attempts) != 1 || store.attempts[0].NextRetryAt == nil || !store.attempts[0].NextRetryAt.Equal(wantRetry) {
		t.Fatalf("recorded attempts = %+v, want retry audit with next retry", store.attempts)
	}
}

func TestWebhookDeliveryPermanentFailureAndMaxAttemptsDisableEndpoint(t *testing.T) {
	t.Run("permanent_status_disables_endpoint", func(t *testing.T) {
		store := &recordingWebhookDeliveryStore{
			endpoints: []webhooks.WebhookDeliveryEndpoint{{
				ID:          "we_perm",
				WorkspaceID: "workspace-a",
				URL:         "https://hooks.example/permanent",
			}},
		}
		client := &recordingWebhookHTTPClient{
			responses: map[string]webhooks.WebhookHTTPResponse{
				"https://hooks.example/permanent": {StatusCode: 404},
			},
		}
		worker := testWebhookDeliveryWorker(store, client)

		results, err := worker.Deliver(context.Background(), webhooks.WebhookDeliveryRequest{
			WorkspaceID: "workspace-a",
			Event:       mustTestWebhookEvent(t, "workspace-a"),
			Attempt:     1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 || results[0].Status != webhooks.WebhookDeliveryFailed || results[0].Retry {
			t.Fatalf("results = %+v, want permanent failure", results)
		}
		if !slices.Contains(store.disabled, "we_perm") {
			t.Fatalf("disabled endpoints = %+v, want we_perm", store.disabled)
		}
	})

	t.Run("max_attempts_disable_retryable_failure", func(t *testing.T) {
		store := &recordingWebhookDeliveryStore{
			endpoints: []webhooks.WebhookDeliveryEndpoint{{
				ID:          "we_exhausted",
				WorkspaceID: "workspace-a",
				URL:         "https://hooks.example/exhausted",
			}},
		}
		client := &recordingWebhookHTTPClient{
			errs: map[string]error{
				"https://hooks.example/exhausted": errors.New("connection reset"),
			},
		}
		worker := testWebhookDeliveryWorker(store, client)

		results, err := worker.Deliver(context.Background(), webhooks.WebhookDeliveryRequest{
			WorkspaceID: "workspace-a",
			Event:       mustTestWebhookEvent(t, "workspace-a"),
			Attempt:     3,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 || results[0].Status != webhooks.WebhookDeliveryFailed || results[0].Retry || results[0].ErrorClass != webhooks.WebhookDeliveryErrorNetwork {
			t.Fatalf("results = %+v, want exhausted network failure", results)
		}
		if !slices.Contains(store.disabled, "we_exhausted") {
			t.Fatalf("disabled endpoints = %+v, want we_exhausted", store.disabled)
		}
	})
}

func TestWebhookDeliverySkipsDisabledEndpoints(t *testing.T) {
	store := &recordingWebhookDeliveryStore{
		endpoints: []webhooks.WebhookDeliveryEndpoint{{
			ID:             "we_disabled",
			WorkspaceID:    "workspace-a",
			URL:            "https://hooks.example/disabled?token=secret",
			Disabled:       true,
			DisabledReason: "max_attempts_exhausted",
		}},
	}
	client := &recordingWebhookHTTPClient{}
	worker := testWebhookDeliveryWorker(store, client)

	results, err := worker.Deliver(context.Background(), webhooks.WebhookDeliveryRequest{
		WorkspaceID: "workspace-a",
		Event:       mustTestWebhookEvent(t, "workspace-a"),
		Attempt:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	result := results[0]
	if result.Status != webhooks.WebhookDeliveryEndpointDisabled || result.Retry {
		t.Fatalf("result = %+v, want endpoint disabled without retry", result)
	}
	if result.Evidence.EndpointURL != "https://hooks.example/disabled?<redacted>" || strings.Contains(result.Evidence.EndpointURL, "secret") {
		t.Fatalf("evidence url = %q, want redacted disabled endpoint", result.Evidence.EndpointURL)
	}
	if len(client.calls) != 0 {
		t.Fatalf("client calls = %+v, want disabled endpoint skipped", client.calls)
	}
}

func TestWebhookDeliveryRecordsSkippedWhenNoEndpoints(t *testing.T) {
	store := &recordingWebhookDeliveryStore{}
	client := &recordingWebhookHTTPClient{}
	worker := testWebhookDeliveryWorker(store, client)

	results, err := worker.Deliver(context.Background(), webhooks.WebhookDeliveryRequest{
		WorkspaceID: "workspace-a",
		Event:       mustTestWebhookEvent(t, "workspace-a"),
		Attempt:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != webhooks.WebhookDeliverySkipped || results[0].Retry {
		t.Fatalf("results = %+v, want skipped delivery without retry", results)
	}
	if results[0].Evidence.WorkspaceID != "workspace-a" || results[0].Evidence.EndpointURL != "" {
		t.Fatalf("evidence = %+v, want workspace-only skipped evidence", results[0].Evidence)
	}
	if len(store.attempts) != 1 || store.attempts[0].Status != webhooks.WebhookDeliverySkipped {
		t.Fatalf("attempts = %+v, want skipped audit record", store.attempts)
	}
	if len(client.calls) != 0 {
		t.Fatalf("client calls = %+v, want no HTTP call without endpoints", client.calls)
	}
}

func testWebhookDeliveryWorker(store webhooks.WebhookDeliveryStore, client webhooks.WebhookHTTPClient) webhooks.WebhookDeliveryWorker {
	return webhooks.WebhookDeliveryWorker{
		Store:       store,
		Client:      client,
		Clock:       fixedWebhookClock{now: time.Date(2026, 4, 28, 16, 0, 0, 0, time.UTC)},
		Secret:      "delivery-secret",
		MaxAttempts: 3,
		Backoff: func(int) time.Duration {
			return time.Minute
		},
	}
}

func mustTestWebhookEvent(t *testing.T, workspaceID string) webhooks.WebhookEvent {
	t.Helper()
	event, err := webhooks.NewTestWebhookEvent(workspaceID)
	if err != nil {
		t.Fatal(err)
	}
	return event
}

type fixedWebhookClock struct {
	now time.Time
}

func (c fixedWebhookClock) Now() time.Time {
	return c.now
}

type recordingWebhookDeliveryStore struct {
	endpoints []webhooks.WebhookDeliveryEndpoint
	attempts  []webhooks.WebhookDeliveryAttempt
	disabled  []string
}

func (s *recordingWebhookDeliveryStore) ListWebhookDeliveryEndpoints(context.Context, string) ([]webhooks.WebhookDeliveryEndpoint, error) {
	out := make([]webhooks.WebhookDeliveryEndpoint, len(s.endpoints))
	copy(out, s.endpoints)
	return out, nil
}

func (s *recordingWebhookDeliveryStore) RecordWebhookDelivery(_ context.Context, attempt webhooks.WebhookDeliveryAttempt) error {
	s.attempts = append(s.attempts, attempt)
	return nil
}

func (s *recordingWebhookDeliveryStore) DisableWebhookDeliveryEndpoint(_ context.Context, endpoint webhooks.WebhookDeliveryEndpoint, _ string, _ time.Time) error {
	s.disabled = append(s.disabled, endpoint.ID)
	return nil
}

type recordingWebhookHTTPClient struct {
	responses map[string]webhooks.WebhookHTTPResponse
	errs      map[string]error
	calls     []webhooks.WebhookHTTPRequest
}

func (c *recordingWebhookHTTPClient) PostWebhook(_ context.Context, req webhooks.WebhookHTTPRequest) (webhooks.WebhookHTTPResponse, error) {
	c.calls = append(c.calls, req)
	if c.errs != nil {
		if err := c.errs[req.URL]; err != nil {
			return webhooks.WebhookHTTPResponse{}, err
		}
	}
	if c.responses != nil {
		if response, ok := c.responses[req.URL]; ok {
			return response, nil
		}
	}
	return webhooks.WebhookHTTPResponse{StatusCode: 200}, nil
}

func hmacSHA256Hex(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
