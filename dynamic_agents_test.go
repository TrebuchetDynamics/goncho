package goncho

import (
	"context"
	"errors"
	"testing"

	memory "github.com/TrebuchetDynamics/goncho/memory"
)

func newTestDynamicAgentRegistry(t *testing.T) (*DynamicAgentRegistry, *memory.SqliteStore, string) {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/dynamic-agents.db"
	store, err := memory.OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	reg, err := NewDynamicAgentRegistry(store.DB())
	if err != nil {
		t.Fatalf("NewDynamicAgentRegistry: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close(context.Background())
	})
	return reg, store, path
}

// TestDynamicAgentRegistry_CreateRoundTrips proves Create + Get returns the
// same AgentRecord with a stable ID and persona seed across a registry
// reopen on the same SQLite database.
func TestDynamicAgentRegistry_CreateRoundTrips(t *testing.T) {
	reg, store, path := newTestDynamicAgentRegistry(t)
	ctx := context.Background()

	created, err := reg.Create(ctx, CreateAgentOptions{
		Name:    "Research Bot",
		Persona: "You answer literature-review questions.",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID != "research-bot" {
		t.Errorf("ID = %q, want %q", created.ID, "research-bot")
	}
	if created.Name != "Research Bot" {
		t.Errorf("Name = %q, want %q", created.Name, "Research Bot")
	}
	if created.Persona != "You answer literature-review questions." {
		t.Errorf("Persona = %q, want literature-review persona", created.Persona)
	}
	if created.CreatedAt.IsZero() {
		t.Errorf("CreatedAt = zero, want non-zero timestamp")
	}

	// Close, then reopen on the same path — state must survive.
	if err := store.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, err := memory.OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatalf("reopen OpenSqlite: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close(ctx) })
	reg2, err := NewDynamicAgentRegistry(reopened.DB())
	if err != nil {
		t.Fatalf("reopen NewDynamicAgentRegistry: %v", err)
	}

	got, found, err := reg2.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if !found {
		t.Fatalf("Get(%q) found=false after reopen, want stored record", created.ID)
	}
	if got.ID != created.ID || got.Name != created.Name || got.Persona != created.Persona {
		t.Errorf("Get after reopen = %+v, want %+v", got, created)
	}
}

// TestDynamicAgentRegistry_BindResolvesByPeer proves Bind + Resolve returns
// the dynamic AgentID for a (channel, chat_id, thread_id) match. This is
// the user-facing contract the gateway overlay reads when an inbound event
// arrives in a thread that runtime-spawned an agent.
func TestDynamicAgentRegistry_BindResolvesByPeer(t *testing.T) {
	reg, _, _ := newTestDynamicAgentRegistry(t)
	ctx := context.Background()

	created, err := reg.Create(ctx, CreateAgentOptions{
		Name:    "Research",
		Persona: "literature review",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	match := BindingMatch{
		Channel:  "telegram",
		PeerKind: "group",
		PeerID:   "-100123",
		ThreadID: "7",
	}
	if err := reg.Bind(ctx, created.ID, match); err != nil {
		t.Fatalf("Bind: %v", err)
	}

	got, found, err := reg.Resolve(ctx, match)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !found {
		t.Fatalf("Resolve(%+v) found=false, want %q", match, created.ID)
	}
	if got != created.ID {
		t.Errorf("Resolve = %q, want %q", got, created.ID)
	}

	// A different thread must not resolve to the same agent.
	other := match
	other.ThreadID = "8"
	if _, found, err := reg.Resolve(ctx, other); err != nil {
		t.Fatalf("Resolve other thread: %v", err)
	} else if found {
		t.Errorf("Resolve(other thread) found=true, want false")
	}
}

// TestDynamicAgentRegistry_StaticConfigWinsOnIDConflict proves Create
// rejects a name that normalizes to an AgentID already claimed by static
// config (passed via CreateAgentOptions.ReservedIDs). Operator-defined
// identity in config.toml is the source of truth; the dynamic registry must
// not silently shadow it.
func TestDynamicAgentRegistry_StaticConfigWinsOnIDConflict(t *testing.T) {
	reg, _, _ := newTestDynamicAgentRegistry(t)
	ctx := context.Background()

	reserved := map[string]struct{}{"research": {}}
	_, err := reg.Create(ctx, CreateAgentOptions{
		Name:        "Research",
		Persona:     "literature review",
		ReservedIDs: reserved,
	})
	if !errors.Is(err, ErrAgentIDReserved) {
		t.Fatalf("Create with reserved id err = %v, want %v", err, ErrAgentIDReserved)
	}

	// And no record landed in the dynamic store.
	if _, found, err := reg.Get(ctx, "research"); err != nil {
		t.Fatalf("Get after rejected create: %v", err)
	} else if found {
		t.Errorf("Get(research) found=true after rejected create, want false")
	}

	// A different (non-reserved) name on the same registry still works.
	other, err := reg.Create(ctx, CreateAgentOptions{
		Name:        "Triage",
		Persona:     "categorize bugs",
		ReservedIDs: reserved,
	})
	if err != nil {
		t.Fatalf("Create non-reserved: %v", err)
	}
	if other.ID != "triage" {
		t.Errorf("Create non-reserved ID = %q, want %q", other.ID, "triage")
	}
}

// TestDynamicAgentRegistry_UnbindRemovesMatch proves Unbind removes the
// persisted binding for a (channel, peer, thread) tuple. Subsequent Resolve
// calls return not-found so the gateway falls back to static config (or
// reports no-agent) — a runtime spawn must be reversible.
func TestDynamicAgentRegistry_UnbindRemovesMatch(t *testing.T) {
	reg, _, _ := newTestDynamicAgentRegistry(t)
	ctx := context.Background()

	created, err := reg.Create(ctx, CreateAgentOptions{Name: "Research"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	match := BindingMatch{
		Channel:  "telegram",
		PeerKind: "group",
		PeerID:   "-100123",
		ThreadID: "7",
	}
	if err := reg.Bind(ctx, created.ID, match); err != nil {
		t.Fatalf("Bind: %v", err)
	}

	if err := reg.Unbind(ctx, match); err != nil {
		t.Fatalf("Unbind: %v", err)
	}

	if _, found, err := reg.Resolve(ctx, match); err != nil {
		t.Fatalf("Resolve after Unbind: %v", err)
	} else if found {
		t.Errorf("Resolve after Unbind found=true, want false")
	}

	// Unbinding an already-removed match is a no-op (not an error).
	if err := reg.Unbind(ctx, match); err != nil {
		t.Errorf("idempotent Unbind err = %v, want nil", err)
	}

	// The agent record itself is not deleted by Unbind — only its binding.
	if _, found, err := reg.Get(ctx, created.ID); err != nil {
		t.Fatalf("Get after Unbind: %v", err)
	} else if !found {
		t.Errorf("Get(%q) found=false after Unbind, want agent record to remain", created.ID)
	}
}
