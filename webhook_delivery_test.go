package goncho

import (
	"context"
	"testing"
	"time"
)

func TestWebhookDeliveryPublicFacadeDeliversSignedPayload(t *testing.T) {
	store := &publicFacadeWebhookDeliveryStore{
		endpoints: []WebhookDeliveryEndpoint{{
			ID:          "we_public",
			WorkspaceID: "workspace-a",
			URL:         "https://hooks.example/public?token=secret",
		}},
	}
	client := &publicFacadeWebhookHTTPClient{}
	worker := WebhookDeliveryWorker{
		Store:       store,
		Client:      client,
		Clock:       publicFacadeWebhookClock{now: time.Date(2026, 4, 28, 16, 0, 0, 0, time.UTC)},
		Secret:      "delivery-secret",
		MaxAttempts: 3,
	}

	event, err := NewTestWebhookEvent("workspace-a")
	if err != nil {
		t.Fatal(err)
	}
	results, err := worker.Deliver(context.Background(), WebhookDeliveryRequest{WorkspaceID: "workspace-a", Event: event})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != WebhookDeliveryDelivered || results[0].Evidence.EndpointURL != "https://hooks.example/public?<redacted>" {
		t.Fatalf("results = %+v, want delivered public facade result with redacted evidence", results)
	}
	if len(client.calls) != 1 || client.calls[0].Headers["X-Honcho-Signature"] == "" {
		t.Fatalf("client calls = %+v, want signed public facade request", client.calls)
	}
	if len(store.attempts) != 1 || store.attempts[0].Status != WebhookDeliveryDelivered {
		t.Fatalf("attempts = %+v, want delivered attempt recorded", store.attempts)
	}
}

type publicFacadeWebhookClock struct {
	now time.Time
}

func (c publicFacadeWebhookClock) Now() time.Time { return c.now }

type publicFacadeWebhookDeliveryStore struct {
	endpoints []WebhookDeliveryEndpoint
	attempts  []WebhookDeliveryAttempt
}

func (s *publicFacadeWebhookDeliveryStore) ListWebhookDeliveryEndpoints(context.Context, string) ([]WebhookDeliveryEndpoint, error) {
	out := make([]WebhookDeliveryEndpoint, len(s.endpoints))
	copy(out, s.endpoints)
	return out, nil
}

func (s *publicFacadeWebhookDeliveryStore) RecordWebhookDelivery(_ context.Context, attempt WebhookDeliveryAttempt) error {
	s.attempts = append(s.attempts, attempt)
	return nil
}

func (s *publicFacadeWebhookDeliveryStore) DisableWebhookDeliveryEndpoint(context.Context, WebhookDeliveryEndpoint, string, time.Time) error {
	return nil
}

type publicFacadeWebhookHTTPClient struct {
	calls []WebhookHTTPRequest
}

func (c *publicFacadeWebhookHTTPClient) PostWebhook(_ context.Context, req WebhookHTTPRequest) (WebhookHTTPResponse, error) {
	c.calls = append(c.calls, req)
	return WebhookHTTPResponse{StatusCode: 202}, nil
}
