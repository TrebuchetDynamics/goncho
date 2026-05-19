package goncho

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

func TestWebhooks_GetOrCreateIsIdempotentAndWorkspaceScoped(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC)
	first, err := svc.GetOrCreateWebhookEndpoint(ctx, WebhookEndpointCreateParams{
		WorkspaceID: "default",
		URL:         "https://example.com/webhook",
		Now:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Created {
		t.Fatalf("Created = false, want first create")
	}
	second, err := svc.GetOrCreateWebhookEndpoint(ctx, WebhookEndpointCreateParams{
		WorkspaceID: "default",
		URL:         "https://example.com/webhook",
		Now:         now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Created {
		t.Fatalf("Created = true, want duplicate to return existing endpoint")
	}
	if first.Endpoint.ID != second.Endpoint.ID || !first.Endpoint.CreatedAt.Equal(second.Endpoint.CreatedAt) {
		t.Fatalf("duplicate endpoint = %+v, want original %+v", second.Endpoint, first.Endpoint)
	}

	other, err := svc.GetOrCreateWebhookEndpoint(ctx, WebhookEndpointCreateParams{
		WorkspaceID: "other",
		URL:         "https://example.com/webhook",
		Now:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if other.Endpoint.ID == first.Endpoint.ID || other.Endpoint.WorkspaceID != "other" {
		t.Fatalf("other workspace endpoint = %+v, want separate resource", other.Endpoint)
	}

	defaultList, err := svc.ListWebhookEndpoints(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(defaultList) != 1 || defaultList[0].URL != "https://example.com/webhook" {
		t.Fatalf("default endpoints = %+v, want one endpoint", defaultList)
	}
}

func TestWebhooks_RejectsInvalidURLAndWorkspaceLimit(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	for _, rawURL := range []string{
		"192.168.1.1/webhook",
		"ftp://example.com/webhook",
		"http://127.0.0.1/webhook",
		"http://10.0.0.1/webhook",
	} {
		_, err := svc.GetOrCreateWebhookEndpoint(ctx, WebhookEndpointCreateParams{
			WorkspaceID: "default",
			URL:         rawURL,
		})
		if !errors.Is(err, ErrWebhookInvalidURL) {
			t.Fatalf("url %q err = %v, want ErrWebhookInvalidURL", rawURL, err)
		}
	}

	if _, err := svc.GetOrCreateWebhookEndpoint(ctx, WebhookEndpointCreateParams{
		WorkspaceID: "default",
		URL:         "https://one.example/webhook",
		Limit:       1,
	}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.GetOrCreateWebhookEndpoint(ctx, WebhookEndpointCreateParams{
		WorkspaceID: "default",
		URL:         "https://two.example/webhook",
		Limit:       1,
	})
	if !errors.Is(err, ErrWebhookLimitReached) {
		t.Fatalf("limit err = %v, want ErrWebhookLimitReached", err)
	}
}

func TestWebhooks_DeleteEndpointIsWorkspaceScoped(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	defaultEndpoint, err := svc.GetOrCreateWebhookEndpoint(ctx, WebhookEndpointCreateParams{
		WorkspaceID: "default",
		URL:         "https://delete.example/webhook",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetOrCreateWebhookEndpoint(ctx, WebhookEndpointCreateParams{
		WorkspaceID: "other",
		URL:         "https://delete.example/webhook",
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteWebhookEndpoint(ctx, "other", defaultEndpoint.Endpoint.ID); !errors.Is(err, ErrWebhookNotFound) {
		t.Fatalf("cross-workspace delete err = %v, want ErrWebhookNotFound", err)
	}
	if err := svc.DeleteWebhookEndpoint(ctx, "default", defaultEndpoint.Endpoint.ID); err != nil {
		t.Fatal(err)
	}
	got, err := svc.ListWebhookEndpoints(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("default endpoints = %+v, want deleted", got)
	}
}

func TestWebhooks_TestEventAndSignatureMatchHonchoContract(t *testing.T) {
	event, err := NewTestWebhookEvent("default")
	if err != nil {
		t.Fatal(err)
	}
	if event.Type != WebhookEventTest || event.WorkspaceID != "default" || event.Data["workspace_id"] != "default" {
		t.Fatalf("event = %+v, want test.event for workspace", event)
	}
	payload := `{"data":{"workspace_id":"default"},"timestamp":"2026-04-28T15:00:00Z","type":"test.event"}`
	got, err := SignWebhookPayload(payload, "webhook-secret")
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte("webhook-secret"))
	mac.Write([]byte(payload))
	want := hex.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}
	if _, err := SignWebhookPayload(payload, ""); !errors.Is(err, ErrWebhookSecretMissing) {
		t.Fatalf("missing secret err = %v, want ErrWebhookSecretMissing", err)
	}
}
