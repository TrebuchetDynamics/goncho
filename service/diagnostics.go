package goncho

import (
	"context"
	"database/sql"

	"github.com/TrebuchetDynamics/goncho/internal/queuestatus"
)

// QueueTaskTypes are the only Honcho-style reasoning work units that Goncho
// reports. Delivery, deletion, and vector reconciliation counters are
// deliberately excluded because queue status is observability, not sync.
var QueueTaskTypes = queuestatus.TaskTypes

// QueueWorkUnitStatus mirrors Honcho's queue status count shape.
type QueueWorkUnitStatus = queuestatus.WorkUnitStatus

// QueueStatus is the local Goncho queue status read model. Until a dedicated
// Goncho task queue exists, it reports deterministic zero-state counts with
// degraded evidence.
type QueueStatus = queuestatus.Status

// ReadQueueStatus returns a deterministic local read model. It never waits for
// the queue to drain; dream rows are auditable work intent, not worker output.
func ReadQueueStatus(ctx context.Context, db *sql.DB, cfgs ...QueueStatusConfig) (QueueStatus, error) {
	return queuestatus.Read(ctx, db, queueStatusDefaults(), cfgs...)
}

// ZeroQueueStatus reports that no dedicated Goncho task queue exists yet while
// preserving Honcho-compatible work-unit fields.
func ZeroQueueStatus() QueueStatus {
	return queuestatus.Zero(queueStatusDefaults())
}

func queueStatusDefaults() queuestatus.Defaults {
	return queuestatus.Defaults{
		WorkspaceID:      DefaultWorkspaceID,
		ObserverPeerID:   DefaultObserverPeerID,
		DreamIdleTimeout: DefaultDreamIdleTimeout,
	}
}
