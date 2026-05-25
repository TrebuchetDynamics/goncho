package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func main() {
	ctx := context.Background()
	store, err := memory.OpenSqlite(filepath.Join(".", "goncho-example.db"), 0, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close(ctx)
	if err := goncho.RunMigrations(store.DB()); err != nil {
		log.Fatal(err)
	}

	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "example", ObserverPeerID: "agent:demo"}, nil)
	mem := goncho.NewMemoryFacade(svc)

	item, err := mem.Add(ctx, goncho.MemoryAddParams{
		ID:       "demo-memory-1",
		UserID:   "user:demo",
		AgentID:  "agent:demo",
		RunID:    "quickstart",
		Content:  "The demo user prefers local-first memory with evidence.",
		Metadata: map[string]string{"kind": "preference"},
	})
	if err != nil {
		log.Fatal(err)
	}

	results, err := mem.Search(ctx, goncho.MemorySearchParams{UserID: "user:demo", Query: "local-first evidence", Metadata: map[string]string{"kind": "preference"}})
	if err != nil {
		log.Fatal(err)
	}

	history, err := mem.History(ctx, goncho.MemoryHistoryParams{ID: item.ID, UserID: "user:demo"})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("saved %s; search=%d; history=%d\n", item.ID, results.Count, history.Count)
}
