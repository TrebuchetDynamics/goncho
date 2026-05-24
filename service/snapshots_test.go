package goncho

import (
	"context"
	"slices"
	"testing"
)

func TestSnapshotManifestIsDeterministicAndGitOperationsAreAdapterOwned(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	ctx := context.Background()
	if err := svc.SetProfile(ctx, "peer-snap", []string{"Snapshot profile fact."}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-snap", SessionKey: "sess-snap", Conclusion: "Snapshot captures conclusions."}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateMemorySlot(ctx, MemorySlotParams{Peer: "peer-snap", Scope: MemoryScopeWorkspace, Name: "style", Value: "snapshot-safe"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpsertAction(ctx, ActionParams{Peer: "peer-snap", ActionID: "snapshot", Title: "Export deterministic manifest"}); err != nil {
		t.Fatal(err)
	}

	first, err := svc.ExportSnapshotManifest(ctx, SnapshotParams{Peer: "peer-snap"})
	if err != nil {
		t.Fatalf("ExportSnapshotManifest first: %v", err)
	}
	second, err := svc.ExportSnapshotManifest(ctx, SnapshotParams{Peer: "peer-snap"})
	if err != nil {
		t.Fatalf("ExportSnapshotManifest second: %v", err)
	}
	if first.SnapshotID == "" || first.SnapshotID != second.SnapshotID {
		t.Fatalf("snapshot IDs = %q/%q, want deterministic stable id", first.SnapshotID, second.SnapshotID)
	}
	if !slices.Equal(snapshotEntryKeys(first.Entries), snapshotEntryKeys(second.Entries)) {
		t.Fatalf("entry keys differ: %v vs %v", snapshotEntryKeys(first.Entries), snapshotEntryKeys(second.Entries))
	}
	if first.Git.AdapterOwned != true || first.Git.Operation != "none" {
		t.Fatalf("git metadata = %+v, want adapter-owned no-op", first.Git)
	}
	if !slices.Contains(snapshotEntryKinds(first.Entries), "conclusion") || !slices.Contains(snapshotEntryKinds(first.Entries), "slot") || !slices.Contains(snapshotEntryKinds(first.Entries), "action") {
		t.Fatalf("snapshot entries = %+v, want conclusion/slot/action", first.Entries)
	}

	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-snap", SessionKey: "sess-snap", Conclusion: "Snapshot diff sees added memory."}); err != nil {
		t.Fatal(err)
	}
	third, err := svc.ExportSnapshotManifest(ctx, SnapshotParams{Peer: "peer-snap"})
	if err != nil {
		t.Fatalf("ExportSnapshotManifest third: %v", err)
	}
	diff := DiffSnapshotManifests(first, third)
	if diff.FromSnapshotID != first.SnapshotID || diff.ToSnapshotID != third.SnapshotID || len(diff.Added) != 1 || len(diff.Removed) != 0 {
		t.Fatalf("snapshot diff = %+v, want one added entry", diff)
	}

	rollback := BuildSnapshotRollbackMetadata(third, first)
	if rollback.AdapterOwned != true || rollback.Applied || rollback.FromSnapshotID != third.SnapshotID || rollback.TargetSnapshotID != first.SnapshotID {
		t.Fatalf("rollback metadata = %+v, want unapplied adapter-owned rollback plan", rollback)
	}
}

func snapshotEntryKeys(entries []SnapshotEntry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Key)
	}
	return out
}

func snapshotEntryKinds(entries []SnapshotEntry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Kind)
	}
	return out
}
