package goncho

import (
	"context"
	"testing"
)

func TestActionSignalReadReceiptAuthorizedSameWorkspaceProfile(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.UpsertAction(ctx, ActionParams{ProfileID: "profile-a", Peer: "peer-signal", ActionID: "review", Title: "Review handoff"}); err != nil {
		t.Fatalf("UpsertAction: %v", err)
	}
	signal, err := svc.SignalAction(ctx, ActionSignalParams{ProfileID: "profile-a", Peer: "peer-signal", ActionID: "review", Signal: "ready", Message: "handoff ready", Actor: "agent:a"})
	if err != nil {
		t.Fatalf("SignalAction: %v", err)
	}

	receipt, err := svc.RecordActionSignalReceipt(ctx, ActionSignalReceiptParams{ProfileID: "profile-a", Peer: "peer-signal", ActionID: "review", SignalID: signal.ID, Actor: "agent:b", ActorProfileID: "profile-a"})
	if err != nil {
		t.Fatalf("RecordActionSignalReceipt: %v", err)
	}
	if !receipt.Authorized || receipt.Decision != ActionSignalReceiptDecisionAllowed || receipt.Receipt.Actor != "agent:b" {
		t.Fatalf("receipt = %+v, want authorized receipt for agent:b", receipt)
	}

	graph, err := svc.ReadActionGraph(ctx, ActionGraphQuery{ProfileID: "profile-a", Peer: "peer-signal"})
	if err != nil {
		t.Fatalf("ReadActionGraph: %v", err)
	}
	node := actionNodeByID(graph.Nodes, "review")
	if node == nil || len(node.Signals) != 1 || len(node.Signals[0].Receipts) != 1 || node.Signals[0].Receipts[0].Actor != "agent:b" {
		t.Fatalf("node = %+v, want signal read receipt attached", node)
	}
}

func TestActionSignalReadReceiptUpsertReturnsStableReceiptID(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.UpsertAction(ctx, ActionParams{Peer: "peer-signal-id", ActionID: "review", Title: "Review receipt identity"}); err != nil {
		t.Fatalf("UpsertAction: %v", err)
	}
	signal, err := svc.SignalAction(ctx, ActionSignalParams{Peer: "peer-signal-id", ActionID: "review", Signal: "ready", Actor: "agent:a"})
	if err != nil {
		t.Fatalf("SignalAction: %v", err)
	}

	first, err := svc.RecordActionSignalReceipt(ctx, ActionSignalReceiptParams{Peer: "peer-signal-id", ActionID: "review", SignalID: signal.ID, Actor: "agent:b"})
	if err != nil {
		t.Fatalf("RecordActionSignalReceipt first: %v", err)
	}
	other, err := svc.RecordActionSignalReceipt(ctx, ActionSignalReceiptParams{Peer: "peer-signal-id", ActionID: "review", SignalID: signal.ID, Actor: "agent:c"})
	if err != nil {
		t.Fatalf("RecordActionSignalReceipt other: %v", err)
	}
	if first.Receipt.ID == 0 || other.Receipt.ID == 0 || first.Receipt.ID == other.Receipt.ID {
		t.Fatalf("receipt ids first=%d other=%d, want distinct non-zero ids", first.Receipt.ID, other.Receipt.ID)
	}

	again, err := svc.RecordActionSignalReceipt(ctx, ActionSignalReceiptParams{Peer: "peer-signal-id", ActionID: "review", SignalID: signal.ID, Actor: "agent:b"})
	if err != nil {
		t.Fatalf("RecordActionSignalReceipt again: %v", err)
	}
	if again.Receipt.ID != first.Receipt.ID {
		t.Fatalf("updated receipt id = %d, want original id %d (not last inserted id %d)", again.Receipt.ID, first.Receipt.ID, other.Receipt.ID)
	}
}

func TestActionSignalReadReceiptDeniesCrossProfileAndAudits(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.UpsertAction(ctx, ActionParams{ProfileID: "profile-a", Peer: "peer-signal", ActionID: "private", Title: "Private profile signal"}); err != nil {
		t.Fatalf("UpsertAction: %v", err)
	}
	signal, err := svc.SignalAction(ctx, ActionSignalParams{ProfileID: "profile-a", Peer: "peer-signal", ActionID: "private", Signal: "blocked", Actor: "agent:a"})
	if err != nil {
		t.Fatalf("SignalAction: %v", err)
	}

	denied, err := svc.RecordActionSignalReceipt(ctx, ActionSignalReceiptParams{ProfileID: "profile-a", Peer: "peer-signal", ActionID: "private", SignalID: signal.ID, Actor: "agent:b", ActorProfileID: "profile-b"})
	if err != nil {
		t.Fatalf("RecordActionSignalReceipt denied: %v", err)
	}
	if denied.Authorized || denied.Decision != ActionSignalReceiptDecisionDenied || denied.Reason == "" {
		t.Fatalf("denied = %+v, want observable authorization denial", denied)
	}

	receipts, err := svc.ListActionSignalReceipts(ctx, ActionSignalReceiptQuery{ProfileID: "profile-a", Peer: "peer-signal", ActionID: "private", SignalID: signal.ID})
	if err != nil {
		t.Fatalf("ListActionSignalReceipts: %v", err)
	}
	if receipts.Count != 0 {
		t.Fatalf("receipts = %+v, want denied cross-profile read not recorded", receipts)
	}
	audit, err := svc.ListActionSignalReceiptAudit(ctx, ActionSignalReceiptAuditQuery{ProfileID: "profile-a", Peer: "peer-signal", ActionID: "private"})
	if err != nil {
		t.Fatalf("ListActionSignalReceiptAudit: %v", err)
	}
	if audit.Count != 1 || audit.Events[0].Decision != ActionSignalReceiptDecisionDenied || audit.Events[0].Actor != "agent:b" {
		t.Fatalf("audit = %+v, want denied actor evidence", audit)
	}
}
