package goncho

import (
	"context"
	"testing"
	"time"
)

func TestWebhooksPublicFacadeCreatesEndpointAndSignsEvent(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	created, err := svc.GetOrCreateWebhookEndpoint(ctx, WebhookEndpointCreateParams{
		WorkspaceID: "default",
		URL:         "https://example.com/webhook",
		Now:         time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.Created || created.Endpoint.WorkspaceID != "default" || created.Endpoint.URL != "https://example.com/webhook" {
		t.Fatalf("created endpoint = %+v, want public facade endpoint", created)
	}
	listed, err := svc.ListWebhookEndpoints(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != created.Endpoint.ID {
		t.Fatalf("listed endpoints = %+v, want created endpoint", listed)
	}

	event, err := NewTestWebhookEvent("default")
	if err != nil {
		t.Fatal(err)
	}
	if event.Type != WebhookEventTest || event.Data["workspace_id"] != "default" {
		t.Fatalf("event = %+v, want test.event workspace payload", event)
	}
	if signature, err := SignWebhookPayload(`{"type":"test.event"}`, "secret"); err != nil || signature == "" {
		t.Fatalf("SignWebhookPayload = %q/%v, want signature", signature, err)
	}
}
