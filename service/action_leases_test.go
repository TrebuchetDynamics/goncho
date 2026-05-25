package goncho

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestActionLeaseAcquireRenewExpireAndAudit(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.UpsertAction(ctx, ActionParams{Peer: "peer-leases", ActionID: "draft", Title: "Draft server mode docs"}); err != nil {
		t.Fatalf("upsert action: %v", err)
	}

	first, err := svc.AcquireActionLease(ctx, ActionLeaseParams{Peer: "peer-leases", ActionID: "draft", Owner: "agent:a", TTL: time.Hour})
	if err != nil {
		t.Fatalf("AcquireActionLease first: %v", err)
	}
	if !first.Acquired || first.Decision != ActionLeaseDecisionAcquired || first.Lease.Owner != "agent:a" || first.Lease.ExpiresAt <= first.Lease.AcquiredAt {
		t.Fatalf("first lease = %+v, want acquired by agent:a with expiry", first)
	}

	blocked, err := svc.AcquireActionLease(ctx, ActionLeaseParams{Peer: "peer-leases", ActionID: "draft", Owner: "agent:b", TTL: time.Hour})
	if err != nil {
		t.Fatalf("AcquireActionLease second: %v", err)
	}
	if blocked.Acquired || blocked.Decision != ActionLeaseDecisionHeldByOther || blocked.Lease.Owner != "agent:a" {
		t.Fatalf("blocked lease = %+v, want held-by-other evidence", blocked)
	}

	renewed, err := svc.RenewActionLease(ctx, ActionLeaseParams{Peer: "peer-leases", ActionID: "draft", Owner: "agent:a", TTL: 2 * time.Hour})
	if err != nil {
		t.Fatalf("RenewActionLease: %v", err)
	}
	if !renewed.Acquired || renewed.Decision != ActionLeaseDecisionRenewed || renewed.Lease.ExpiresAt <= first.Lease.ExpiresAt {
		t.Fatalf("renewed lease = %+v first=%+v, want extended expiry", renewed, first)
	}

	audit, err := svc.ListActionLeaseAudit(ctx, ActionLeaseAuditQuery{Peer: "peer-leases", ActionID: "draft"})
	if err != nil {
		t.Fatalf("ListActionLeaseAudit: %v", err)
	}
	if len(audit.Events) != 3 || audit.Events[0].Decision != ActionLeaseDecisionRenewed || audit.Events[1].Decision != ActionLeaseDecisionHeldByOther || audit.Events[2].Decision != ActionLeaseDecisionAcquired {
		t.Fatalf("audit events = %+v, want renew/deny/acquire newest-first", audit.Events)
	}
	if audit.Events[1].Actor != "agent:b" || audit.Events[1].Reason == "" {
		t.Fatalf("deny audit = %+v, want actor and reason", audit.Events[1])
	}
}

func TestActionLeaseConcurrentAcquireAllowsOnlyOneOwner(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.UpsertAction(ctx, ActionParams{Peer: "peer-concurrent", ActionID: "lease", Title: "Lease exactly once"}); err != nil {
		t.Fatalf("upsert action: %v", err)
	}

	start := make(chan struct{})
	results := make(chan ActionLeaseResult, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, owner := range []string{"agent:a", "agent:b"} {
		owner := owner
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := svc.AcquireActionLease(ctx, ActionLeaseParams{Peer: "peer-concurrent", ActionID: "lease", Owner: owner, TTL: time.Hour})
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatalf("AcquireActionLease concurrent: %v", err)
	}
	acquired := 0
	held := 0
	owners := map[string]bool{}
	for result := range results {
		if result.Decision == ActionLeaseDecisionAcquired {
			acquired++
			owners[result.Lease.Owner] = true
		}
		if result.Decision == ActionLeaseDecisionHeldByOther {
			held++
		}
	}
	if acquired != 1 || held != 1 || len(owners) != 1 {
		t.Fatalf("concurrent lease results acquired=%d held=%d owners=%v, want exactly one owner and one denial", acquired, held, owners)
	}
}

func TestActionLeaseExpirationAllowsAnotherOwner(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.UpsertAction(ctx, ActionParams{Peer: "peer-expire", ActionID: "smoke", Title: "Run compose smoke"}); err != nil {
		t.Fatalf("upsert action: %v", err)
	}
	if _, err := svc.AcquireActionLease(ctx, ActionLeaseParams{Peer: "peer-expire", ActionID: "smoke", Owner: "agent:a", TTL: time.Millisecond}); err != nil {
		t.Fatalf("AcquireActionLease: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	expired, err := svc.ExpireActionLeases(ctx, ActionLeaseExpireParams{Peer: "peer-expire"})
	if err != nil {
		t.Fatalf("ExpireActionLeases: %v", err)
	}
	if expired.ExpiredCount != 1 || len(expired.Events) != 1 || expired.Events[0].Decision != ActionLeaseDecisionExpired || expired.Events[0].Actor != "agent:a" {
		t.Fatalf("expired = %+v, want one expired audit event for agent:a", expired)
	}

	next, err := svc.AcquireActionLease(ctx, ActionLeaseParams{Peer: "peer-expire", ActionID: "smoke", Owner: "agent:b", TTL: time.Hour})
	if err != nil {
		t.Fatalf("AcquireActionLease after expiry: %v", err)
	}
	if !next.Acquired || next.Lease.Owner != "agent:b" {
		t.Fatalf("next lease = %+v, want agent:b acquired after expiry", next)
	}
}
