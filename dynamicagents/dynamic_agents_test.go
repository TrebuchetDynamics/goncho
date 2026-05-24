package dynamicagents

import (
	"context"
	"testing"

	memory "github.com/TrebuchetDynamics/goncho/memory"
)

func TestDynamicAgentRegistryPublicFacadeCreatesAndResolvesBinding(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(t.TempDir()+"/dynamic-agents.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() { _ = store.Close(ctx) })

	reg, err := NewDynamicAgentRegistry(store.DB())
	if err != nil {
		t.Fatalf("NewDynamicAgentRegistry: %v", err)
	}
	created, err := reg.Create(ctx, CreateAgentOptions{Name: "Research Bot", Persona: "literature review"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID != "research-bot" || created.Persona != "literature review" {
		t.Fatalf("created = %+v, want public facade dynamic agent record", created)
	}

	match := BindingMatch{Channel: "telegram", PeerKind: "group", PeerID: "-100123", ThreadID: "7"}
	if err := reg.Bind(ctx, created.ID, match); err != nil {
		t.Fatalf("Bind: %v", err)
	}
	got, found, err := reg.Resolve(ctx, match)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !found || got != created.ID {
		t.Fatalf("Resolve = %q/%v, want %q/true", got, found, created.ID)
	}
}
