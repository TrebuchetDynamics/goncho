package goncho

import (
	"context"
	"testing"
)

func TestImageMemoryStoresRefsChecksumsAndSearchesWithoutEmbeddings(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	ctx := context.Background()
	stored, err := svc.StoreImageMemory(ctx, ImageMemoryParams{
		Peer:       "peer-images",
		ImageRef:   "file://screenshots/login-error.png",
		Checksum:   "sha256:abc123",
		AltText:    "Login page showing invalid token error",
		SessionKey: "sess-images",
		Metadata:   map[string]string{"media_type": "image/png"},
	})
	if err != nil {
		t.Fatalf("StoreImageMemory: %v", err)
	}
	if stored.ID == 0 || stored.EmbeddingStatus != ImageEmbeddingDeferred || stored.Checksum != "sha256:abc123" {
		t.Fatalf("stored image = %+v", stored)
	}

	replayed, err := svc.StoreImageMemory(ctx, ImageMemoryParams{Peer: "peer-images", ImageRef: "file://screenshots/login-error.png", Checksum: "sha256:abc123", AltText: "duplicate", SessionKey: "sess-images"})
	if err != nil {
		t.Fatalf("StoreImageMemory duplicate: %v", err)
	}
	if replayed.ID != stored.ID || !replayed.Replayed {
		t.Fatalf("replayed = %+v, want idempotent replay of %d", replayed, stored.ID)
	}

	byChecksum, err := svc.SearchImageMemories(ctx, ImageMemoryQuery{Peer: "peer-images", Query: "sha256:abc123", Limit: 5})
	if err != nil {
		t.Fatalf("SearchImageMemories checksum: %v", err)
	}
	if len(byChecksum.Images) != 1 || byChecksum.Images[0].ImageRef != "file://screenshots/login-error.png" {
		t.Fatalf("checksum search = %+v", byChecksum.Images)
	}
	byText, err := svc.SearchImageMemories(ctx, ImageMemoryQuery{Peer: "peer-images", Query: "invalid token", Limit: 5})
	if err != nil {
		t.Fatalf("SearchImageMemories text: %v", err)
	}
	if len(byText.Images) != 1 || byText.Images[0].EmbeddingStatus != ImageEmbeddingDeferred {
		t.Fatalf("text search = %+v, want image ref with deferred embedding status", byText.Images)
	}
}
