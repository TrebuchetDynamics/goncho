package goncho

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestQueueStatusPublicFacadeWorkUnitStatusJSONShapeIncludesSessionDetails(t *testing.T) {
	raw, err := json.Marshal(QueueWorkUnitStatus{
		CompletedWorkUnits:  2,
		InProgressWorkUnits: 1,
		PendingWorkUnits:    3,
		TotalWorkUnits:      6,
		Sessions: map[string]QueueWorkUnitStatus{
			"sess-a": {
				PendingWorkUnits: 1,
				TotalWorkUnits:   1,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	text := string(raw)
	for _, want := range []string{
		`"completed_work_units":2`,
		`"in_progress_work_units":1`,
		`"pending_work_units":3`,
		`"total_work_units":6`,
		`"sessions":{"sess-a"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("QueueWorkUnitStatus JSON missing %s in %s", want, raw)
		}
	}
}

func TestQueueStatusPublicFacadeReadQueueStatusUsesGonchoDefaults(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	got, err := ReadQueueStatus(context.Background(), svc.db)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "degraded" || !got.Degraded || !got.ObservabilityOnly {
		t.Fatalf("ReadQueueStatus = %+v, want degraded observability-only zero state", got)
	}
	if got.Dream.Status != "dream_disabled" || len(got.Dream.Evidence) != 1 {
		t.Fatalf("Dream status = %+v, want disabled evidence", got.Dream)
	}
	if got.Dream.Evidence[0].WorkspaceID != DefaultWorkspaceID || got.Dream.Evidence[0].ObserverPeerID != DefaultObserverPeerID {
		t.Fatalf("Dream evidence defaults = %+v, want workspace/observer defaults", got.Dream.Evidence[0])
	}
}
