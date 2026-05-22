package goncho_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/goncho/memory"
)

func ExampleNewService() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "goncho-example-*")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	store, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 0, nil)
	if err != nil {
		panic(err)
	}
	defer func() { _ = store.Close(ctx) }()

	if err := goncho.RunMigrations(store.DB()); err != nil {
		panic(err)
	}

	svc := goncho.NewService(store.DB(), goncho.Config{
		WorkspaceID:    "example-agent",
		ObserverPeerID: "assistant",
	}, nil)

	if err := svc.SetProfile(ctx, "user:juan", []string{
		"Prefers SQLite-backed local memory.",
	}); err != nil {
		panic(err)
	}

	profile, err := svc.Profile(ctx, "user:juan")
	if err != nil {
		panic(err)
	}
	fmt.Println(profile.Card[0])

	// Output:
	// Prefers SQLite-backed local memory.
}
