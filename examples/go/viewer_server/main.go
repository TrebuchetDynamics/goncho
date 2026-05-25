package main

import (
	"context"
	"fmt"
	"net/http/httptest"
	"path/filepath"

	gonchohttp "github.com/TrebuchetDynamics/goncho/http"
	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func main() {
	ctx := context.Background()
	store, err := memory.OpenSqlite(filepath.Join(".", "goncho-viewer.db"), 0, nil)
	if err != nil {
		panic(err)
	}
	defer store.Close(ctx)
	if err := goncho.RunMigrations(store.DB()); err != nil {
		panic(err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "example", ObserverPeerID: "agent"}, nil)
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: "user", SessionKey: "viewer", Conclusion: "Viewer example memory.", Scope: goncho.MemoryScopeWorkspace}); err != nil {
		panic(err)
	}
	snapshot, err := svc.ViewerSnapshot(ctx)
	if err != nil {
		panic(err)
	}
	server := httptest.NewServer(gonchohttp.NewServiceHandler(svc))
	defer server.Close()
	fmt.Printf("viewer=%s conclusions=%d url=%s/v3/workspaces/example/viewer\n", snapshot.Status, snapshot.Counts.Conclusions, server.URL)
}
