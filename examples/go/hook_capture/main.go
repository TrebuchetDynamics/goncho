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
	store, err := memory.OpenSqlite(filepath.Join(".", "goncho-hooks.db"), 0, nil)
	if err != nil {
		panic(err)
	}
	defer store.Close(ctx)
	if err := goncho.RunMigrations(store.DB()); err != nil {
		panic(err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "example", ObserverPeerID: "host"}, nil)
	captured, err := svc.CaptureHostHook(ctx, goncho.HostHookEvent{Event: goncho.HostHookPrompt, Host: "example-host", PeerID: "user", SessionKey: "session-1", Content: "Remember: run tests before release."})
	if err != nil {
		panic(err)
	}
	fmt.Printf("observations=%d messages=%d\n", len(captured.Observations), len(captured.Messages))
}
