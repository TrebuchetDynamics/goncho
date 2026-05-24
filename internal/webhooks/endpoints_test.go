package webhooks_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/webhooks"
	memory "github.com/TrebuchetDynamics/goncho/memory"
)

func newTestWebhookDB(t *testing.T) *memory.SqliteStore {
	t.Helper()
	store, err := memory.OpenSqlite(t.TempDir()+"/webhooks.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	return store
}

func TestWebhooks_GetOrCreateIsIdempotentAndWorkspaceScoped(t *testing.T) {
	store := newTestWebhookDB(t)
	ctx := context.Background()
	now := time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC)
	first, err := webhooks.GetOrCreateEndpoint(ctx, store.DB(), "default", webhooks.WebhookEndpointCreateParams{
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
	second, err := webhooks.GetOrCreateEndpoint(ctx, store.DB(), "default", webhooks.WebhookEndpointCreateParams{
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

	other, err := webhooks.GetOrCreateEndpoint(ctx, store.DB(), "default", webhooks.WebhookEndpointCreateParams{
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

	defaultList, err := webhooks.ListEndpoints(ctx, store.DB(), "default", "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(defaultList) != 1 || defaultList[0].URL != "https://example.com/webhook" {
		t.Fatalf("default endpoints = %+v, want one endpoint", defaultList)
	}
}

func TestWebhooks_RejectsInvalidURLAndWorkspaceLimit(t *testing.T) {
	store := newTestWebhookDB(t)
	ctx := context.Background()
	for _, rawURL := range []string{
		"192.168.1.1/webhook",
		"ftp://example.com/webhook",
		"http://127.0.0.1/webhook",
		"http://10.0.0.1/webhook",
	} {
		_, err := webhooks.GetOrCreateEndpoint(ctx, store.DB(), "default", webhooks.WebhookEndpointCreateParams{
			WorkspaceID: "default",
			URL:         rawURL,
		})
		if !errors.Is(err, webhooks.ErrWebhookInvalidURL) {
			t.Fatalf("url %q err = %v, want ErrWebhookInvalidURL", rawURL, err)
		}
	}

	if _, err := webhooks.GetOrCreateEndpoint(ctx, store.DB(), "default", webhooks.WebhookEndpointCreateParams{
		WorkspaceID: "default",
		URL:         "https://one.example/webhook",
		Limit:       1,
	}); err != nil {
		t.Fatal(err)
	}
	_, err := webhooks.GetOrCreateEndpoint(ctx, store.DB(), "default", webhooks.WebhookEndpointCreateParams{
		WorkspaceID: "default",
		URL:         "https://two.example/webhook",
		Limit:       1,
	})
	if !errors.Is(err, webhooks.ErrWebhookLimitReached) {
		t.Fatalf("limit err = %v, want ErrWebhookLimitReached", err)
	}
}

func TestWebhooks_DeleteEndpointIsWorkspaceScoped(t *testing.T) {
	store := newTestWebhookDB(t)
	ctx := context.Background()
	defaultEndpoint, err := webhooks.GetOrCreateEndpoint(ctx, store.DB(), "default", webhooks.WebhookEndpointCreateParams{
		WorkspaceID: "default",
		URL:         "https://delete.example/webhook",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := webhooks.GetOrCreateEndpoint(ctx, store.DB(), "default", webhooks.WebhookEndpointCreateParams{
		WorkspaceID: "other",
		URL:         "https://delete.example/webhook",
	}); err != nil {
		t.Fatal(err)
	}
	if err := webhooks.DeleteEndpoint(ctx, store.DB(), "default", "other", defaultEndpoint.Endpoint.ID); !errors.Is(err, webhooks.ErrWebhookNotFound) {
		t.Fatalf("cross-workspace delete err = %v, want ErrWebhookNotFound", err)
	}
	if err := webhooks.DeleteEndpoint(ctx, store.DB(), "default", "default", defaultEndpoint.Endpoint.ID); err != nil {
		t.Fatal(err)
	}
	got, err := webhooks.ListEndpoints(ctx, store.DB(), "default", "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("default endpoints = %+v, want deleted", got)
	}
}

func TestWebhooks_TestEventAndSignatureMatchHonchoContract(t *testing.T) {
	event, err := webhooks.NewTestWebhookEvent("default")
	if err != nil {
		t.Fatal(err)
	}
	if event.Type != webhooks.WebhookEventTest || event.WorkspaceID != "default" || event.Data["workspace_id"] != "default" {
		t.Fatalf("event = %+v, want test.event for workspace", event)
	}
	payload := `{"data":{"workspace_id":"default"},"timestamp":"2026-04-28T15:00:00Z","type":"test.event"}`
	got, err := webhooks.SignWebhookPayload(payload, "webhook-secret")
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte("webhook-secret"))
	mac.Write([]byte(payload))
	want := hex.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}
	if _, err := webhooks.SignWebhookPayload(payload, ""); !errors.Is(err, webhooks.ErrWebhookSecretMissing) {
		t.Fatalf("missing secret err = %v, want ErrWebhookSecretMissing", err)
	}
}
