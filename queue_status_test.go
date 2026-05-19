package goncho

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestContractQueueWorkUnitStatusJSONShapeIncludesSessionDetails(t *testing.T) {
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

func TestReadQueueStatusZeroStateIsDeterministicObservabilityOnly(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	first, err := ReadQueueStatus(context.Background(), svc.db)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadQueueStatus(context.Background(), svc.db)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("ReadQueueStatus returned nondeterministic zero-state:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if first.Status != "degraded" || !first.Degraded {
		t.Fatalf("status = %q degraded=%t, want degraded zero-state", first.Status, first.Degraded)
	}
	if !first.ObservabilityOnly {
		t.Fatal("ObservabilityOnly = false, want true")
	}
	if !strings.Contains(first.Message, "zero tracked work units") {
		t.Fatalf("Message = %q, want zero tracked work units evidence", first.Message)
	}
	if !strings.Contains(first.Message, "observability") || !strings.Contains(first.Message, "do not wait") {
		t.Fatalf("Message = %q, want explicit observability-not-synchronization warning", first.Message)
	}

	for _, taskType := range QueueTaskTypes {
		counts, ok := first.WorkUnits[taskType]
		if !ok {
			t.Fatalf("WorkUnits missing task type %q: %#v", taskType, first.WorkUnits)
		}
		if counts.CompletedWorkUnits != 0 || counts.InProgressWorkUnits != 0 || counts.PendingWorkUnits != 0 || counts.TotalWorkUnits != 0 {
			t.Fatalf("%s counts = %+v, want deterministic zero-state", taskType, counts)
		}
		if len(counts.Sessions) != 0 {
			t.Fatalf("%s sessions = %+v, want no per-session details before a Goncho task queue exists", taskType, counts.Sessions)
		}
	}
}

func TestQueueStatusOnlyReportsHonchoReasoningWorkTypes(t *testing.T) {
	want := []string{"representation", "summary", "dream"}
	if !slices.Equal(QueueTaskTypes, want) {
		t.Fatalf("QueueTaskTypes = %#v, want %#v", QueueTaskTypes, want)
	}

	status := ZeroQueueStatus()
	if len(status.WorkUnits) != len(want) {
		t.Fatalf("WorkUnits len = %d, want only %d Honcho reasoning task types: %#v", len(status.WorkUnits), len(want), status.WorkUnits)
	}
	for _, internalTask := range []string{"webhook", "deletion", "vector_reconciliation", "reconciler"} {
		if _, ok := status.WorkUnits[internalTask]; ok {
			t.Fatalf("WorkUnits included internal infrastructure task %q: %#v", internalTask, status.WorkUnits)
		}
	}
}
