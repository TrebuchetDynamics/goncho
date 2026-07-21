package goncho

import (
	"context"
	"testing"
)

func TestConsistencyReportGroupsDuplicatesAndConflictsWithinScope(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	first, err := svc.Conclude(ctx, ConcludeParams{ProfileID: "profile-a", Peer: "team", SessionKey: "session-1", Scope: MemoryScopeWorkspace, Conclusion: "Build database uses SQLite."})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Conclude(ctx, ConcludeParams{ProfileID: "profile-a", Peer: "team", SessionKey: "session-2", Scope: MemoryScopeWorkspace, Conclusion: "Build database uses PostgreSQL."})
	if err != nil {
		t.Fatal(err)
	}
	duplicateA, err := svc.Conclude(ctx, ConcludeParams{ProfileID: "profile-a", Peer: "team", SessionKey: "session-1", Scope: MemoryScopeWorkspace, Conclusion: "Backups run before every release."})
	if err != nil {
		t.Fatal(err)
	}
	duplicateB, err := svc.Conclude(ctx, ConcludeParams{ProfileID: "profile-a", Peer: "team", SessionKey: "session-2", Scope: MemoryScopeWorkspace, Conclusion: "Backups run before every release."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{ProfileID: "profile-a", Peer: "team", SessionKey: "session-3", Scope: MemoryScopeProfile, Conclusion: "Build database uses MySQL."}); err != nil {
		t.Fatal(err)
	}

	report, err := svc.ConsistencyReport(ctx, ConsistencyReportParams{ProfileID: "profile-a", Peer: "team"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Mutates {
		t.Fatal("consistency report must be read-only")
	}
	if report.Counts[ConsistencyDuplicate] != 1 || report.Counts[ConsistencyConflict] != 1 {
		t.Fatalf("counts = %+v, want one duplicate and one conflict group", report.Counts)
	}
	assertConsistencyGroupIDs(t, report.Groups, ConsistencyConflict, first.ID, second.ID)
	assertConsistencyGroupIDs(t, report.Groups, ConsistencyDuplicate, duplicateA.ID, duplicateB.ID)
}

func assertConsistencyGroupIDs(t *testing.T, groups []ConsistencyGroup, kind string, want ...int64) {
	t.Helper()
	for _, group := range groups {
		if group.Kind != kind {
			continue
		}
		got := map[int64]bool{}
		for _, member := range group.Members {
			got[member.ID] = true
		}
		if len(got) != len(want) {
			continue
		}
		all := true
		for _, id := range want {
			all = all && got[id]
		}
		if all {
			return
		}
	}
	t.Fatalf("groups = %+v, want %s group containing IDs %v", groups, kind, want)
}
