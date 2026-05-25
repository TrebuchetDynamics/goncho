package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func main() {
	ctx := context.Background()
	store, err := memory.OpenSqlite(filepath.Join(".", "goncho-example.db"), 0, nil)
	if err != nil {
		panic(err)
	}
	defer store.Close(ctx)
	if err := goncho.RunMigrations(store.DB()); err != nil {
		panic(err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "example", ObserverPeerID: "agent"}, nil)
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: "user", SessionKey: "demo", Conclusion: "Project docs live in docs/.", Scope: goncho.MemoryScopeWorkspace}); err != nil {
		panic(err)
	}
	result, err := svc.Search(ctx, goncho.SearchParams{Peer: "user", Query: "Where do project docs live?"})
	if err != nil {
		panic(err)
	}
	fmt.Printf("results=%d\n", len(result.Results))
}
