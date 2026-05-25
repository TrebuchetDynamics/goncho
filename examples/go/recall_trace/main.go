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
	store, err := memory.OpenSqlite(filepath.Join(".", "goncho-recall.db"), 0, nil)
	if err != nil {
		panic(err)
	}
	defer store.Close(ctx)
	if err := goncho.RunMigrations(store.DB()); err != nil {
		panic(err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "example", ObserverPeerID: "agent"}, nil)
	_, _ = svc.Conclude(ctx, goncho.ConcludeParams{Peer: "user", SessionKey: "release", Conclusion: "Release checklist: run go test ./... before tagging.", Scope: goncho.MemoryScopeWorkspace})
	trace, err := svc.Recall(ctx, goncho.RecallQuery{Peer: "user", SessionKey: "release", Query: "What should I run before tagging?", Limit: 5})
	if err != nil {
		panic(err)
	}
	fmt.Printf("trace=%s selected=%d warnings=%d\n", trace.TraceID, len(trace.Selected), len(trace.Warnings))
}
