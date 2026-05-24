package queuestatus

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/dreamscheduler"
)

// TaskTypes are the only Honcho-style reasoning work units that Goncho
// reports. Delivery, deletion, and vector reconciliation counters are
// deliberately excluded because queue status is observability, not sync.
var TaskTypes = []string{"representation", "summary", "dream"}

// WorkUnitStatus mirrors Honcho's queue status count shape.
type WorkUnitStatus struct {
	CompletedWorkUnits  int                       `json:"completed_work_units"`
	InProgressWorkUnits int                       `json:"in_progress_work_units"`
	PendingWorkUnits    int                       `json:"pending_work_units"`
	TotalWorkUnits      int                       `json:"total_work_units"`
	Sessions            map[string]WorkUnitStatus `json:"sessions,omitempty"`
}

// Status is the local Goncho queue status read model. Until a dedicated
// Goncho task queue exists, it reports deterministic zero-state counts with
// degraded evidence.
type Status struct {
	Status            string                          `json:"status"`
	ObservabilityOnly bool                            `json:"observability_only"`
	WorkUnits         map[string]WorkUnitStatus       `json:"work_units"`
	Dream             dreamscheduler.DreamQueueStatus `json:"dream"`
	Degraded          bool                            `json:"degraded"`
	Message           string                          `json:"message"`
}

// Defaults provides root-package defaults without making this internal Module
// import the public Goncho package.
type Defaults struct {
	WorkspaceID      string
	ObserverPeerID   string
	DreamIdleTimeout time.Duration
}

// Read returns a deterministic local read model. It never waits for the queue
// to drain; dream rows are auditable work intent, not worker output.
func Read(ctx context.Context, db *sql.DB, defaults Defaults, cfgs ...dreamscheduler.QueueStatusConfig) (Status, error) {
	if db == nil {
		return Status{}, errors.New("goncho: nil db")
	}
	if err := ctx.Err(); err != nil {
		return Status{}, err
	}
	cfg := effectiveConfig(defaults, cfgs...)
	status := Zero(defaults)
	dream, counts, err := readDreamQueueStatus(ctx, db, cfg)
	if err != nil {
		return Status{}, err
	}
	status.Dream = dream
	status.WorkUnits["dream"] = counts
	if counts.TotalWorkUnits > 0 {
		status.Message = "no dedicated Goncho representation/summary worker queue exists yet; dream work intent is tracked locally for observability and debugging, do not wait for an empty queue"
	}
	return status, nil
}

// Zero reports that no dedicated Goncho task queue exists yet while preserving
// Honcho-compatible work-unit fields.
func Zero(defaults Defaults) Status {
	cfg := effectiveConfig(defaults)
	workUnits := make(map[string]WorkUnitStatus, len(TaskTypes))
	for _, taskType := range TaskTypes {
		workUnits[taskType] = WorkUnitStatus{}
	}
	return Status{
		Status:            "degraded",
		ObservabilityOnly: true,
		WorkUnits:         workUnits,
		Dream: dreamscheduler.DreamQueueStatus{
			Status:  "dream_disabled",
			Enabled: false,
			Evidence: []dreamscheduler.DreamStatusEvidence{
				dreamscheduler.DisabledEvidence(cfg.WorkspaceID, cfg.ObserverPeerID, ""),
			},
		},
		Degraded: true,
		Message:  "no dedicated Goncho task queue exists yet; zero tracked work units; queue status is for observability and debugging, do not wait for an empty queue",
	}
}

func effectiveConfig(defaults Defaults, cfgs ...dreamscheduler.QueueStatusConfig) dreamscheduler.QueueStatusConfig {
	var cfg dreamscheduler.QueueStatusConfig
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	cfg.WorkspaceID = strings.TrimSpace(cfg.WorkspaceID)
	if cfg.WorkspaceID == "" {
		cfg.WorkspaceID = strings.TrimSpace(defaults.WorkspaceID)
	}
	cfg.ObserverPeerID = strings.TrimSpace(cfg.ObserverPeerID)
	if cfg.ObserverPeerID == "" {
		cfg.ObserverPeerID = strings.TrimSpace(defaults.ObserverPeerID)
	}
	if cfg.Now.IsZero() {
		cfg.Now = time.Now().UTC()
	}
	if cfg.DreamIdleTimeout <= 0 {
		cfg.DreamIdleTimeout = defaults.DreamIdleTimeout
	}
	if cfg.DreamIdleTimeout <= 0 {
		cfg.DreamIdleTimeout = dreamscheduler.DefaultIdleTimeout
	}
	return cfg
}

func readDreamQueueStatus(ctx context.Context, db *sql.DB, cfg dreamscheduler.QueueStatusConfig) (dreamscheduler.DreamQueueStatus, WorkUnitStatus, error) {
	if !cfg.DreamEnabled {
		return dreamscheduler.DreamQueueStatus{
			Status:  "dream_disabled",
			Enabled: false,
			Evidence: []dreamscheduler.DreamStatusEvidence{
				dreamscheduler.DisabledEvidence(cfg.WorkspaceID, cfg.ObserverPeerID, ""),
			},
		}, WorkUnitStatus{}, nil
	}
	present, err := dreamscheduler.TableExists(ctx, db, "goncho_dreams")
	if err != nil {
		return dreamscheduler.DreamQueueStatus{}, WorkUnitStatus{}, err
	}
	if !present {
		return dreamscheduler.DreamQueueStatus{
			Status:       "dream_unavailable",
			Enabled:      true,
			TablePresent: false,
			Evidence: []dreamscheduler.DreamStatusEvidence{
				dreamscheduler.UnavailableEvidence(cfg.WorkspaceID, cfg.ObserverPeerID, ""),
			},
		}, WorkUnitStatus{}, nil
	}

	counts, err := readDreamWorkUnitCounts(ctx, db, cfg.WorkspaceID, cfg.ObserverPeerID)
	if err != nil {
		return dreamscheduler.DreamQueueStatus{}, WorkUnitStatus{}, err
	}
	evidence, err := readDreamStatusEvidence(ctx, db, cfg)
	if err != nil {
		return dreamscheduler.DreamQueueStatus{}, WorkUnitStatus{}, err
	}

	status := "idle"
	if counts.PendingWorkUnits > 0 || counts.InProgressWorkUnits > 0 {
		status = "active"
	} else {
		for _, item := range evidence {
			if item.Code == "dream_cooldown" {
				status = "cooldown"
				break
			}
		}
	}
	return dreamscheduler.DreamQueueStatus{
		Status:       status,
		Enabled:      true,
		TablePresent: true,
		Evidence:     evidence,
	}, counts, nil
}

func readDreamWorkUnitCounts(ctx context.Context, db *sql.DB, workspaceID, observer string) (WorkUnitStatus, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT status, COUNT(*)
		FROM goncho_dreams
		WHERE workspace_id = ?
		  AND observer_peer_id = ?
		GROUP BY status
	`, workspaceID, observer)
	if err != nil {
		return WorkUnitStatus{}, fmt.Errorf("goncho: dream status counts: %w", err)
	}
	defer rows.Close()

	var counts WorkUnitStatus
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return WorkUnitStatus{}, fmt.Errorf("goncho: scan dream status counts: %w", err)
		}
		counts.TotalWorkUnits += count
		switch status {
		case "pending":
			counts.PendingWorkUnits = count
		case "in_progress":
			counts.InProgressWorkUnits = count
		case "completed":
			counts.CompletedWorkUnits = count
		}
	}
	if err := rows.Err(); err != nil {
		return WorkUnitStatus{}, fmt.Errorf("goncho: dream status count rows: %w", err)
	}
	return counts, nil
}

func readDreamStatusEvidence(ctx context.Context, db *sql.DB, cfg dreamscheduler.QueueStatusConfig) ([]dreamscheduler.DreamStatusEvidence, error) {
	var out []dreamscheduler.DreamStatusEvidence
	activeRows, err := db.QueryContext(ctx, `
		SELECT id, workspace_id, observer_peer_id, observed_peer_id, work_unit_key,
		       status, new_conclusions, min_conclusions, last_conclusion_id,
		       COALESCE(last_activity_at, 0), COALESCE(idle_until, 0), COALESCE(cooldown_until, 0)
		FROM goncho_dreams
		WHERE workspace_id = ?
		  AND observer_peer_id = ?
		  AND status IN ('pending', 'in_progress')
		ORDER BY updated_at DESC, id DESC
		LIMIT 10
	`, cfg.WorkspaceID, cfg.ObserverPeerID)
	if err != nil {
		return nil, fmt.Errorf("goncho: dream active evidence: %w", err)
	}
	defer activeRows.Close()
	for activeRows.Next() {
		intent, err := dreamscheduler.ScanIntent(activeRows)
		if err != nil {
			return nil, err
		}
		out = append(out, dreamscheduler.EvidenceFromIntent(intent))
	}
	if err := activeRows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: dream active evidence rows: %w", err)
	}

	now := cfg.Now.UTC().Unix()
	cooldownSeconds := int64(dreamscheduler.DefaultCooldown.Seconds())
	cooldownRows, err := db.QueryContext(ctx, `
		SELECT id, workspace_id, observer_peer_id, observed_peer_id, work_unit_key,
		       status, new_conclusions, min_conclusions, last_conclusion_id,
		       COALESCE(last_activity_at, 0), COALESCE(idle_until, 0),
		       COALESCE(NULLIF(cooldown_until, 0), COALESCE(completed_at, updated_at) + ?)
		FROM goncho_dreams
		WHERE workspace_id = ?
		  AND observer_peer_id = ?
		  AND status = 'completed'
		  AND COALESCE(NULLIF(cooldown_until, 0), COALESCE(completed_at, updated_at) + ?) > ?
		ORDER BY COALESCE(NULLIF(cooldown_until, 0), COALESCE(completed_at, updated_at) + ?) DESC, id DESC
		LIMIT 10
	`, cooldownSeconds, cfg.WorkspaceID, cfg.ObserverPeerID, cooldownSeconds, now, cooldownSeconds)
	if err != nil {
		return nil, fmt.Errorf("goncho: dream cooldown evidence: %w", err)
	}
	defer cooldownRows.Close()
	for cooldownRows.Next() {
		intent, err := dreamscheduler.ScanIntent(cooldownRows)
		if err != nil {
			return nil, err
		}
		item := dreamscheduler.EvidenceFromIntent(intent)
		item.Code = "dream_cooldown"
		item.Reason = "dream cooldown has not elapsed since the last completed dream"
		out = append(out, item)
	}
	if err := cooldownRows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: dream cooldown evidence rows: %w", err)
	}
	return out, nil
}
