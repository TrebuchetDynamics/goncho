package goncho

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDistributedLeaseAPIsRejectWithoutServerMode(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if _, err := svc.UpsertAction(ctx, ActionParams{Peer: "peer-local", ActionID: "lease", Title: "Local action"}); err != nil {
		t.Fatalf("UpsertAction: %v", err)
	}
	_, err := svc.AcquireActionLease(ctx, ActionLeaseParams{Peer: "peer-local", ActionID: "lease", Owner: "agent:a", TTL: time.Hour})
	if err == nil || !strings.Contains(err.Error(), "requires server_mode=team-enabled") {
		t.Fatalf("AcquireActionLease err = %v, want server-mode gate", err)
	}
}
