package goncho

import (
	"context"
	"errors"
	"testing"
)

func TestMemorySlotsSupportScopedCRUDAppendAuditAndIsolation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	ctx := context.Background()
	created, err := svc.CreateMemorySlot(ctx, MemorySlotParams{
		ProfileID: "mineru",
		Peer:      "peer-slots",
		Scope:     MemoryScopeProfile,
		Name:      "reply_style",
		Value:     "concise",
		Kind:      "preference",
	})
	if err != nil {
		t.Fatalf("CreateMemorySlot: %v", err)
	}
	if created.Revision != 1 || created.Deleted {
		t.Fatalf("created slot = %+v, want revision 1 active", created)
	}

	appended, err := svc.AppendMemorySlot(ctx, MemorySlotParams{ProfileID: "mineru", Peer: "peer-slots", Scope: MemoryScopeProfile, Name: "reply_style", Value: "no filler"})
	if err != nil {
		t.Fatalf("AppendMemorySlot: %v", err)
	}
	if appended.Value != "concise\nno filler" || appended.Revision != 2 {
		t.Fatalf("appended slot = %+v, want newline append revision 2", appended)
	}

	replaced, err := svc.ReplaceMemorySlot(ctx, MemorySlotParams{ProfileID: "mineru", Peer: "peer-slots", Scope: MemoryScopeProfile, Name: "reply_style", Value: "direct and terse", Kind: "preference"})
	if err != nil {
		t.Fatalf("ReplaceMemorySlot: %v", err)
	}
	if replaced.Value != "direct and terse" || replaced.Revision != 3 {
		t.Fatalf("replaced slot = %+v, want replacement revision 3", replaced)
	}

	got, err := svc.GetMemorySlot(ctx, MemorySlotQuery{ProfileID: "mineru", Peer: "peer-slots", Scope: MemoryScopeProfile, Name: "reply_style"})
	if err != nil {
		t.Fatalf("GetMemorySlot: %v", err)
	}
	if got.Value != "direct and terse" || got.Kind != "preference" {
		t.Fatalf("got slot = %+v", got)
	}

	mineru, err := svc.ListMemorySlots(ctx, MemorySlotQuery{ProfileID: "mineru", Peer: "peer-slots", Scope: MemoryScopeProfile})
	if err != nil {
		t.Fatalf("ListMemorySlots mineru: %v", err)
	}
	if len(mineru.Slots) != 1 || mineru.Slots[0].Name != "reply_style" {
		t.Fatalf("mineru slots = %+v, want one reply_style", mineru.Slots)
	}
	yunobo, err := svc.ListMemorySlots(ctx, MemorySlotQuery{ProfileID: "yunobo", Peer: "peer-slots", Scope: MemoryScopeProfile})
	if err != nil {
		t.Fatalf("ListMemorySlots yunobo: %v", err)
	}
	if len(yunobo.Slots) != 0 {
		t.Fatalf("yunobo slots = %+v, want profile isolation", yunobo.Slots)
	}

	deleted, err := svc.DeleteMemorySlot(ctx, MemorySlotQuery{ProfileID: "mineru", Peer: "peer-slots", Scope: MemoryScopeProfile, Name: "reply_style"})
	if err != nil {
		t.Fatalf("DeleteMemorySlot: %v", err)
	}
	if !deleted.Deleted || deleted.Revision != 4 {
		t.Fatalf("deleted slot = %+v, want tombstone revision 4", deleted)
	}
	_, err = svc.GetMemorySlot(ctx, MemorySlotQuery{ProfileID: "mineru", Peer: "peer-slots", Scope: MemoryScopeProfile, Name: "reply_style"})
	if !errors.Is(err, ErrMemorySlotNotFound) {
		t.Fatalf("GetMemorySlot after delete err = %v, want ErrMemorySlotNotFound", err)
	}

	audit, err := svc.ListObservations(ctx, ObservationQuery{PeerID: "peer-slots", Kinds: []ObservationKind{ObservationKindCustom}, Limit: 10})
	if err != nil {
		t.Fatalf("ListObservations audit: %v", err)
	}
	if audit.Count != 4 {
		t.Fatalf("audit count = %d, want four mutating slot audit events", audit.Count)
	}
	for _, obs := range audit.Observations {
		if obs.Metadata["custom_kind"] != "memory_slot" || obs.Metadata["slot_name"] != "reply_style" || obs.Metadata["profile_id"] != "mineru" {
			t.Fatalf("slot audit metadata = %+v", obs.Metadata)
		}
	}
}
