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
	failed := false
	for _, id := range []string{"resource-fail-1", "resource-fail-2"} {
		if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, PeerID: "peer-resources", SessionKey: "sess-resources", Success: &failed, Input: "private command", Output: "private stack", Metadata: map[string]string{"tool_name": "bash"}}); err != nil {
			t.Fatalf("Observe failure %s: %v", id, err)
		}
	}

	registry := NewMemoryResourceRegistry(svc)
	descriptors := registry.Descriptors()
	if got, want := resourceDescriptorNames(descriptors), []string{"goncho://graph/stats", "goncho://handoff/prompt", "goncho://latest", "goncho://negative-evidence/candidates", "goncho://profile", "goncho://recall/prompt", "goncho://review/prompt", "goncho://status", "goncho://verify/prompt"}; !slices.Equal(got, want) {
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

	negative, err := registry.Read(ctx, MemoryResourceRequest{URI: "goncho://negative-evidence/candidates", Peer: "peer-resources", SessionKey: "sess-resources"})
	if err != nil {
		t.Fatalf("negative evidence resource: %v", err)
	}
	if negative.Payload["count"] != 1 {
		t.Fatalf("negative payload = %+v, want one candidate", negative.Payload)
	}
	candidates, ok := negative.Payload["candidates"].([]NegativeEvidenceCandidate)
	if !ok || len(candidates) != 1 || candidates[0].FailureCount != 2 || candidates[0].ToolName != "bash" {
		t.Fatalf("negative candidates = %#v", negative.Payload["candidates"])
	}
	if strings.Contains(candidates[0].String(), "private command") || strings.Contains(candidates[0].String(), "private stack") {
		t.Fatalf("negative candidate leaked raw observation content: %s", candidates[0].String())
	}

	prompt, err := registry.Read(ctx, MemoryResourceRequest{URI: "goncho://recall/prompt", Peer: "peer-resources", Query: "who owns auth?", Limit: 3})
	if err != nil {
		t.Fatalf("recall prompt resource: %v", err)
	}
	text, ok := prompt.Payload["prompt"].(string)
	if !ok || !strings.Contains(text, "who owns auth?") || !strings.Contains(text, "Use goncho_recall") || !strings.Contains(text, "Require provenance") {
		t.Fatalf("recall prompt = %#v", prompt.Payload["prompt"])
	}
	for _, tc := range []struct {
		uri  string
		want string
	}{
		{"goncho://handoff/prompt", "session handoff"},
		{"goncho://review/prompt", "Review open Goncho memory items"},
		{"goncho://verify/prompt", "Before taking consequential action"},
	} {
		content, err := registry.Read(ctx, MemoryResourceRequest{URI: tc.uri, Peer: "peer-resources", Query: "deploy?"})
		if err != nil {
			t.Fatalf("read %s: %v", tc.uri, err)
		}
		text, ok := content.Payload["prompt"].(string)
		if !ok || !strings.Contains(text, tc.want) {
			t.Fatalf("prompt %s = %#v, want %q", tc.uri, content.Payload["prompt"], tc.want)
		}
	}
}

func resourceDescriptorNames(descriptors []MemoryResourceDescriptor) []string {
	names := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		names = append(names, descriptor.URI)
	}
	return names
}
