package goncho

import (
	"context"
	"testing"
)

func TestLocalActionGraphTracksDependenciesFrontierNextActionAndSignals(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	ctx := context.Background()
	if _, err := svc.UpsertAction(ctx, ActionParams{Peer: "peer-actions", ActionID: "design", Title: "Design slot memory UX"}); err != nil {
		t.Fatalf("upsert design: %v", err)
	}
	if _, err := svc.UpsertAction(ctx, ActionParams{Peer: "peer-actions", ActionID: "test", Title: "Write slot memory tests", DependsOn: []string{"design"}}); err != nil {
		t.Fatalf("upsert test: %v", err)
	}
	if _, err := svc.UpsertAction(ctx, ActionParams{Peer: "peer-actions", ActionID: "ship", Title: "Ship slot memory", DependsOn: []string{"test"}}); err != nil {
		t.Fatalf("upsert ship: %v", err)
	}

	graph, err := svc.ReadActionGraph(ctx, ActionGraphQuery{Peer: "peer-actions"})
	if err != nil {
		t.Fatalf("ReadActionGraph: %v", err)
	}
	if len(graph.Nodes) != 3 || len(graph.Frontier) != 1 || graph.Frontier[0].ActionID != "design" {
		t.Fatalf("initial graph = %+v, want only design in frontier", graph)
	}
	if graph.NextAction == nil || graph.NextAction.ActionID != "design" {
		t.Fatalf("next action = %+v, want design", graph.NextAction)
	}

	if _, err := svc.CompleteAction(ctx, ActionGraphQuery{Peer: "peer-actions", ActionID: "design"}); err != nil {
		t.Fatalf("CompleteAction: %v", err)
	}
	graph, err = svc.ReadActionGraph(ctx, ActionGraphQuery{Peer: "peer-actions"})
	if err != nil {
		t.Fatalf("ReadActionGraph after complete: %v", err)
	}
	if len(graph.Frontier) != 1 || graph.Frontier[0].ActionID != "test" || graph.NextAction.ActionID != "test" {
		t.Fatalf("frontier after design done = %+v, want test", graph.Frontier)
	}

	signal, err := svc.SignalAction(ctx, ActionSignalParams{Peer: "peer-actions", ActionID: "ship", Signal: "blocked", Message: "waiting for docs review", Actor: "agent:test"})
	if err != nil {
		t.Fatalf("SignalAction: %v", err)
	}
	if signal.Signal != "blocked" || signal.Actor != "agent:test" {
		t.Fatalf("signal = %+v", signal)
	}
	graph, err = svc.ReadActionGraph(ctx, ActionGraphQuery{Peer: "peer-actions"})
	if err != nil {
		t.Fatalf("ReadActionGraph after signal: %v", err)
	}
	ship := actionNodeByID(graph.Nodes, "ship")
	if ship == nil || len(ship.Signals) != 1 || ship.Signals[0].Message != "waiting for docs review" {
		t.Fatalf("ship node = %+v, want signal attached", ship)
	}
	if ship.Status != ActionStatusPending {
		t.Fatalf("ship status = %s, want local-only signals not leases/status mutation", ship.Status)
	}
}

func actionNodeByID(nodes []ActionNode, id string) *ActionNode {
	for i := range nodes {
		if nodes[i].ActionID == id {
			return &nodes[i]
		}
	}
	return nil
}
