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
	store, err := memory.OpenSqlite(filepath.Join(".", "goncho-slots.db"), 0, nil)
	if err != nil {
		panic(err)
	}
	defer store.Close(ctx)
	if err := goncho.RunMigrations(store.DB()); err != nil {
		panic(err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "example", ObserverPeerID: "agent"}, nil)
	if _, err := svc.CreateMemorySlot(ctx, goncho.MemorySlotParams{Peer: "user", Scope: goncho.MemoryScopeWorkspace, Name: "release_checklist", Kind: "text", Value: "run tests before release"}); err != nil {
		panic(err)
	}
	slots, err := svc.ListMemorySlots(ctx, goncho.MemorySlotQuery{Peer: "user", Scope: goncho.MemoryScopeWorkspace})
	if err != nil {
		panic(err)
	}
	fmt.Printf("slots=%d\n", len(slots.Slots))
}
