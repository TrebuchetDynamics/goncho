package goncho

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestMemoryResourceRegistryExposesStatusProfileLatestGraphAndRecallPrompt(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	ctx := context.Background()
	if err := svc.SetProfile(ctx, "peer-resources", []string{"Prefers resources before tool registration."}); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-resources", SessionKey: "sess-resources", Conclusion: "Mira owns the authentication service."}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-resources", SessionKey: "sess-resources", Conclusion: "The authentication service uses SQLite for local recall."}); err != nil {
		t.Fatal(err)
	}

	registry := NewMemoryResourceRegistry(svc)
	descriptors := registry.Descriptors()
	if got, want := resourceDescriptorNames(descriptors), []string{"goncho://graph/stats", "goncho://latest", "goncho://profile", "goncho://recall/prompt", "goncho://status"}; !slices.Equal(got, want) {
		t.Fatalf("resource descriptors = %v, want %v", got, want)
	}
	for _, descriptor := range descriptors {
		if descriptor.Kind != MemoryResourceKindResource && descriptor.Kind != MemoryResourceKindPrompt {
			t.Fatalf("descriptor %s kind = %q", descriptor.URI, descriptor.Kind)
		}
		if descriptor.Description == "" || descriptor.MimeType == "" {
			t.Fatalf("descriptor %s missing description/mime type: %+v", descriptor.URI, descriptor)
		}
	}

	status, err := registry.Read(ctx, MemoryResourceRequest{URI: "goncho://status", Peer: "peer-resources"})
	if err != nil {
		t.Fatalf("status resource: %v", err)
	}
	if status.URI != "goncho://status" || status.Payload["workspace_id"] != "default" || status.Payload["observer_peer_id"] != "gormes" {
		t.Fatalf("status payload = %+v", status.Payload)
	}
	if caps, ok := status.Payload["capabilities"].([]string); !ok || !slices.Contains(caps, "recall") || !slices.Contains(caps, "hook_capture") {
		t.Fatalf("status capabilities = %#v, want recall and hook_capture", status.Payload["capabilities"])
	}

	profile, err := registry.Read(ctx, MemoryResourceRequest{URI: "goncho://profile", Peer: "peer-resources"})
	if err != nil {
		t.Fatalf("profile resource: %v", err)
	}
	if profile.Payload["peer"] != "peer-resources" {
		t.Fatalf("profile payload = %+v", profile.Payload)
	}
	if card, ok := profile.Payload["card"].([]string); !ok || !slices.Contains(card, "Prefers resources before tool registration.") {
		t.Fatalf("profile card = %#v", profile.Payload["card"])
	}

	latest, err := registry.Read(ctx, MemoryResourceRequest{URI: "goncho://latest", Peer: "peer-resources", SessionKey: "sess-resources", Limit: 2})
	if err != nil {
		t.Fatalf("latest resource: %v", err)
	}
	if latest.Payload["count"] != 2 {
		t.Fatalf("latest payload = %+v, want count 2", latest.Payload)
	}
	if hits, ok := latest.Payload["results"].([]SearchHit); !ok || len(hits) != 2 {
		t.Fatalf("latest results = %#v", latest.Payload["results"])
	}

	graph, err := registry.Read(ctx, MemoryResourceRequest{URI: "goncho://graph/stats", Peer: "peer-resources"})
	if err != nil {
		t.Fatalf("graph stats resource: %v", err)
	}
	if graph.Payload["annotation_count"] == nil || graph.Payload["relation_count"] == nil {
		t.Fatalf("graph payload = %+v, want annotation/relation stats", graph.Payload)
	}

	prompt, err := registry.Read(ctx, MemoryResourceRequest{URI: "goncho://recall/prompt", Peer: "peer-resources", Query: "who owns auth?", Limit: 3})
	if err != nil {
		t.Fatalf("recall prompt resource: %v", err)
	}
	text, ok := prompt.Payload["prompt"].(string)
	if !ok || !strings.Contains(text, "who owns auth?") || !strings.Contains(text, "Use goncho_recall") || !strings.Contains(text, "Require provenance") {
		t.Fatalf("recall prompt = %#v", prompt.Payload["prompt"])
	}
}

func resourceDescriptorNames(descriptors []MemoryResourceDescriptor) []string {
	names := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		names = append(names, descriptor.URI)
	}
	return names
}
