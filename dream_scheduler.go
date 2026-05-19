package goncho

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const defaultDreamType = "consolidation"

// DreamScheduleParams controls a local dream work-intent scheduling attempt.
// Manual requests bypass threshold/cooldown/idle gates but still dedupe active
// work and respect disabled/unavailable degraded modes.
type DreamScheduleParams struct {
	Peer   string
	Now    time.Time
	Manual bool
	Reason string
}

// DreamScheduleResult is auditable scheduler evidence. It never contains LLM
// output; the first dream slice only records local work intent.
type DreamScheduleResult struct {
	Action           string              `json:"action"`
	ID               int64               `json:"id,omitempty"`
	Status           string              `json:"status,omitempty"`
	WorkspaceID      string              `json:"workspace_id"`
	ObserverPeerID   string              `json:"observer_peer_id"`
	ObservedPeerID   string              `json:"observed_peer_id"`
	WorkUnitKey      string              `json:"work_unit_key,omitempty"`
	NewConclusions   int                 `json:"new_conclusions"`
	MinConclusions   int                 `json:"min_conclusions"`
	LastConclusionID int64               `json:"last_conclusion_id,omitempty"`
	Evidence         DreamStatusEvidence `json:"evidence"`
}

// DreamStatusEvidence is the shared doctor/status/context evidence shape for
// local dream scheduler state.
type DreamStatusEvidence struct {
	Code             string `json:"code"`
	Reason           string `json:"reason"`
	DreamID          int64  `json:"dream_id,omitempty"`
	WorkspaceID      string `json:"workspace_id,omitempty"`
	ObserverPeerID   string `json:"observer_peer_id,omitempty"`
	ObservedPeerID   string `json:"observed_peer_id,omitempty"`
	Status           string `json:"status,omitempty"`
	WorkUnitKey      string `json:"work_unit_key,omitempty"`
	NewConclusions   int    `json:"new_conclusions,omitempty"`
	MinConclusions   int    `json:"min_conclusions,omitempty"`
	LastConclusionID int64  `json:"last_conclusion_id,omitempty"`
	LastActivityAt   int64  `json:"last_activity_at,omitempty"`
	IdleUntil        int64  `json:"idle_until,omitempty"`
	CooldownUntil    int64  `json:"cooldown_until,omitempty"`
}

// DreamQueueStatus is embedded in queue/doctor output so operators can see
// whether dreaming is disabled, unavailable, cooling down, pending, or running.
type DreamQueueStatus struct {
	Status       string                `json:"status"`
	Enabled      bool                  `json:"enabled"`
	TablePresent bool                  `json:"table_present"`
	Evidence     []DreamStatusEvidence `json:"evidence,omitempty"`
}

// QueueStatusConfig supplies config-dependent dream status to ReadQueueStatus.
type QueueStatusConfig struct {
	DreamEnabled     bool
	WorkspaceID      string
	ObserverPeerID   string
	Now              time.Time
	DreamIdleTimeout time.Duration
}

type dreamEligibility struct {
	newConclusions   int
	maxConclusionID  int64
	lastConclusionID int64
	lastActivityAt   int64
	idleUntil        int64
	cooldownUntil    int64
	lastDreamID      int64
}

type dreamIntent struct {
	ID               int64
	WorkspaceID      string
	ObserverPeerID   string
	ObservedPeerID   string
	WorkUnitKey      string
	Status           string
	NewConclusions   int
	MinConclusions   int
	LastConclusionID int64
	LastActivityAt   int64
	IdleUntil        int64
	CooldownUntil    int64
}

func (s *Service) ScheduleDream(ctx context.Context, params DreamScheduleParams) (DreamScheduleResult, error) {
	peer := strings.TrimSpace(params.Peer)
	if peer == "" {
		return DreamScheduleResult{}, fmt.Errorf("goncho: peer is required")
	}
	now := effectiveDreamNow(params.Now)
	base := DreamScheduleResult{
		Action:         "rejected",
		WorkspaceID:    s.workspaceID,
		ObserverPeerID: s.observer,
		ObservedPeerID: peer,
		MinConclusions: DefaultDreamMinConclusions,
	}
	if !s.dreamEnabled {
		base.Evidence = dreamDisabledEvidence(s.workspaceID, s.observer, peer)
		return base, nil
	}
	present, err := sqliteTableExists(ctx, s.db, "goncho_dreams")
	if err != nil {
		return DreamScheduleResult{}, err
	}
	if !present {
		base.Evidence = dreamUnavailableEvidence(s.workspaceID, s.observer, peer)
		return base, nil
	}

	active, err := findActiveDreamIntent(ctx, s.db, s.workspaceID, s.observer, peer)
	if err != nil {
		return DreamScheduleResult{}, err
	}
	if active != nil {
		return dreamResultFromIntent("reused", *active), nil
	}

	eligibility, err := readDreamEligibility(ctx, s.db, s.workspaceID, s.observer, peer, now, s.dreamIdle)
	if err != nil {
		return DreamScheduleResult{}, err
	}
	base.NewConclusions = eligibility.newConclusions
	base.LastConclusionID = eligibility.maxConclusionID
	if !params.Manual {
		if eligibility.newConclusions < DefaultDreamMinConclusions {
			base.Evidence = DreamStatusEvidence{
				Code:             "dream_threshold",
				Reason:           "dream requires at least 50 new conclusions since the last completed dream",
				WorkspaceID:      s.workspaceID,
				ObserverPeerID:   s.observer,
				ObservedPeerID:   peer,
				NewConclusions:   eligibility.newConclusions,
				MinConclusions:   DefaultDreamMinConclusions,
				LastConclusionID: eligibility.maxConclusionID,
			}
			return base, nil
		}
		if eligibility.cooldownUntil > now {
			base.Evidence = DreamStatusEvidence{
				Code:             "dream_cooldown",
				Reason:           "dream cooldown has not elapsed since the last completed dream",
				WorkspaceID:      s.workspaceID,
				ObserverPeerID:   s.observer,
				ObservedPeerID:   peer,
				NewConclusions:   eligibility.newConclusions,
				MinConclusions:   DefaultDreamMinConclusions,
				LastConclusionID: eligibility.maxConclusionID,
				CooldownUntil:    eligibility.cooldownUntil,
			}
			return base, nil
		}
		if eligibility.idleUntil > now {
			base.Evidence = DreamStatusEvidence{
				Code:             "dream_idle",
				Reason:           "observed peer has not been idle for the configured dream idle timeout",
				WorkspaceID:      s.workspaceID,
				ObserverPeerID:   s.observer,
				ObservedPeerID:   peer,
				NewConclusions:   eligibility.newConclusions,
				MinConclusions:   DefaultDreamMinConclusions,
				LastConclusionID: eligibility.maxConclusionID,
				LastActivityAt:   eligibility.lastActivityAt,
				IdleUntil:        eligibility.idleUntil,
			}
			return base, nil
		}
	}

	reason := strings.TrimSpace(params.Reason)
	if reason == "" {
		if params.Manual {
			reason = "manual"
		} else {
			reason = "eligible"
		}
	}
	intent := dreamIntent{
		WorkspaceID:      s.workspaceID,
		ObserverPeerID:   s.observer,
		ObservedPeerID:   peer,
		WorkUnitKey:      dreamWorkUnitKey(s.workspaceID, s.observer, peer),
		Status:           "pending",
		NewConclusions:   eligibility.newConclusions,
		MinConclusions:   DefaultDreamMinConclusions,
		LastConclusionID: eligibility.maxConclusionID,
		LastActivityAt:   eligibility.lastActivityAt,
		IdleUntil:        eligibility.idleUntil,
		CooldownUntil:    eligibility.cooldownUntil,
	}
	id, err := insertDreamIntent(ctx, s.db, intent, now, params.Manual, reason)
	if err != nil {
		active, findErr := findActiveDreamIntent(ctx, s.db, s.workspaceID, s.observer, peer)
		if findErr == nil && active != nil {
			return dreamResultFromIntent("reused", *active), nil
		}
		return DreamScheduleResult{}, err
	}
	intent.ID = id
	return dreamResultFromIntent("created", intent), nil
}

func (s *Service) dreamContextUnavailableEvidence(ctx context.Context, peer string) ([]ContextUnavailableEvidence, error) {
	if !s.dreamEnabled {
		return []ContextUnavailableEvidence{{
			Field:      "dream",
			Capability: "dream_disabled",
			Reason:     "dreaming is disabled; no background dream reasoning is active",
		}}, nil
	}
	present, err := sqliteTableExists(ctx, s.db, "goncho_dreams")
	if err != nil {
		return nil, err
	}
	if !present {
		return []ContextUnavailableEvidence{{
			Field:      "dream",
			Capability: "dream_unavailable",
			Reason:     "goncho_dreams scheduler table is unavailable; no background dream reasoning is active for " + peer,
		}}, nil
	}
	return nil, nil
}

func (s *Service) cancelPendingDreamsForObserved(ctx context.Context, observed string, now int64, reason string) (int64, error) {
	present, err := sqliteTableExists(ctx, s.db, "goncho_dreams")
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, nil
	}
	if strings.TrimSpace(reason) == "" {
		reason = "new_activity"
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE goncho_dreams
		SET status = 'stale',
			reason = ?,
			stale_at = ?,
			updated_at = ?
		WHERE workspace_id = ?
		  AND observed_peer_id = ?
		  AND status = 'pending'
	`, reason, now, now, s.workspaceID, observed)
	if err != nil {
		return 0, fmt.Errorf("goncho: stale pending dreams: %w", err)
	}
	return res.RowsAffected()
}

func dreamResultFromIntent(action string, intent dreamIntent) DreamScheduleResult {
	return DreamScheduleResult{
		Action:           action,
		ID:               intent.ID,
		Status:           intent.Status,
		WorkspaceID:      intent.WorkspaceID,
		ObserverPeerID:   intent.ObserverPeerID,
		ObservedPeerID:   intent.ObservedPeerID,
		WorkUnitKey:      intent.WorkUnitKey,
		NewConclusions:   intent.NewConclusions,
		MinConclusions:   intent.MinConclusions,
		LastConclusionID: intent.LastConclusionID,
		Evidence:         dreamEvidenceFromIntent(intent),
	}
}

func dreamEvidenceFromIntent(intent dreamIntent) DreamStatusEvidence {
	code := "dream_" + intent.Status
	reason := "dream work intent is " + intent.Status
	if intent.Status == "pending" {
		reason = "dream work intent is pending local execution"
	}
	if intent.Status == "in_progress" {
		reason = "dream work intent is in progress"
	}
	return DreamStatusEvidence{
		Code:             code,
		Reason:           reason,
		DreamID:          intent.ID,
		WorkspaceID:      intent.WorkspaceID,
		ObserverPeerID:   intent.ObserverPeerID,
		ObservedPeerID:   intent.ObservedPeerID,
		Status:           intent.Status,
		WorkUnitKey:      intent.WorkUnitKey,
		NewConclusions:   intent.NewConclusions,
		MinConclusions:   intent.MinConclusions,
		LastConclusionID: intent.LastConclusionID,
		LastActivityAt:   intent.LastActivityAt,
		IdleUntil:        intent.IdleUntil,
		CooldownUntil:    intent.CooldownUntil,
	}
}

func readDreamEligibility(ctx context.Context, db *sql.DB, workspaceID, observer, peer string, now int64, idleTimeout time.Duration) (dreamEligibility, error) {
	if idleTimeout <= 0 {
		idleTimeout = DefaultDreamIdleTimeout
	}
	var out dreamEligibility
	last, err := latestCompletedDream(ctx, db, workspaceID, observer, peer)
	if err != nil {
		return dreamEligibility{}, err
	}
	if last != nil {
		out.lastDreamID = last.ID
		out.lastConclusionID = last.LastConclusionID
		out.cooldownUntil = last.CooldownUntil
		if out.cooldownUntil <= 0 && last.CooldownUntil == 0 {
			out.cooldownUntil = 0
		}
	}

	var maxConclusion sql.NullInt64
	var lastActivity sql.NullInt64
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*), MAX(id), MAX(created_at)
		FROM goncho_conclusions
		WHERE workspace_id = ?
		  AND observer_peer_id = ?
		  AND peer_id = ?
		  AND status = 'processed'
		  AND id > ?
	`, workspaceID, observer, peer, out.lastConclusionID).Scan(&out.newConclusions, &maxConclusion, &lastActivity); err != nil {
		return dreamEligibility{}, fmt.Errorf("goncho: count dream conclusions: %w", err)
	}
	if maxConclusion.Valid {
		out.maxConclusionID = maxConclusion.Int64
	}
	if lastActivity.Valid {
		out.lastActivityAt = lastActivity.Int64
		out.idleUntil = out.lastActivityAt + int64(idleTimeout.Seconds())
	}
	if last != nil && last.CooldownUntil <= 0 {
		out.cooldownUntil = last.LastActivityAt + int64(DefaultDreamCooldown.Seconds())
	}
	_ = now
	return out, nil
}

func latestCompletedDream(ctx context.Context, db *sql.DB, workspaceID, observer, peer string) (*dreamIntent, error) {
	var intent dreamIntent
	var completedAt sql.NullInt64
	err := db.QueryRowContext(ctx, `
		SELECT id, workspace_id, observer_peer_id, observed_peer_id, work_unit_key,
		       status, new_conclusions, min_conclusions, last_conclusion_id,
		       COALESCE(last_activity_at, 0), COALESCE(idle_until, 0),
		       COALESCE(cooldown_until, 0), completed_at
		FROM goncho_dreams
		WHERE workspace_id = ?
		  AND observer_peer_id = ?
		  AND observed_peer_id = ?
		  AND status = 'completed'
		ORDER BY COALESCE(completed_at, updated_at) DESC, id DESC
		LIMIT 1
	`, workspaceID, observer, peer).Scan(
		&intent.ID,
		&intent.WorkspaceID,
		&intent.ObserverPeerID,
		&intent.ObservedPeerID,
		&intent.WorkUnitKey,
		&intent.Status,
		&intent.NewConclusions,
		&intent.MinConclusions,
		&intent.LastConclusionID,
		&intent.LastActivityAt,
		&intent.IdleUntil,
		&intent.CooldownUntil,
		&completedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("goncho: latest completed dream: %w", err)
	}
	if intent.CooldownUntil <= 0 && completedAt.Valid {
		intent.CooldownUntil = completedAt.Int64 + int64(DefaultDreamCooldown.Seconds())
	}
	return &intent, nil
}

func findActiveDreamIntent(ctx context.Context, db *sql.DB, workspaceID, observer, peer string) (*dreamIntent, error) {
	var intent dreamIntent
	err := db.QueryRowContext(ctx, `
		SELECT id, workspace_id, observer_peer_id, observed_peer_id, work_unit_key,
		       status, new_conclusions, min_conclusions, last_conclusion_id,
		       COALESCE(last_activity_at, 0), COALESCE(idle_until, 0), COALESCE(cooldown_until, 0)
		FROM goncho_dreams
		WHERE workspace_id = ?
		  AND observer_peer_id = ?
		  AND observed_peer_id = ?
		  AND status IN ('pending', 'in_progress')
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`, workspaceID, observer, peer).Scan(
		&intent.ID,
		&intent.WorkspaceID,
		&intent.ObserverPeerID,
		&intent.ObservedPeerID,
		&intent.WorkUnitKey,
		&intent.Status,
		&intent.NewConclusions,
		&intent.MinConclusions,
		&intent.LastConclusionID,
		&intent.LastActivityAt,
		&intent.IdleUntil,
		&intent.CooldownUntil,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("goncho: find active dream: %w", err)
	}
	return &intent, nil
}

func insertDreamIntent(ctx context.Context, db *sql.DB, intent dreamIntent, now int64, manual bool, reason string) (int64, error) {
	manualInt := 0
	if manual {
		manualInt = 1
	}
	res, err := db.ExecContext(ctx, `
		INSERT INTO goncho_dreams(
			workspace_id, observer_peer_id, observed_peer_id, work_unit_key, dream_type,
			status, manual, reason, new_conclusions, min_conclusions, last_conclusion_id,
			scheduled_for, last_activity_at, cooldown_until, idle_until, created_at, updated_at
		)
		VALUES(?, ?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		intent.WorkspaceID,
		intent.ObserverPeerID,
		intent.ObservedPeerID,
		intent.WorkUnitKey,
		defaultDreamType,
		manualInt,
		reason,
		intent.NewConclusions,
		intent.MinConclusions,
		intent.LastConclusionID,
		now,
		intent.LastActivityAt,
		intent.CooldownUntil,
		intent.IdleUntil,
		now,
		now,
	)
	if err != nil {
		return 0, fmt.Errorf("goncho: insert dream intent: %w", err)
	}
	return res.LastInsertId()
}

func dreamDisabledEvidence(workspaceID, observer, peer string) DreamStatusEvidence {
	return DreamStatusEvidence{
		Code:           "dream_disabled",
		Reason:         "dreaming is disabled; no background dream reasoning is active",
		WorkspaceID:    workspaceID,
		ObserverPeerID: observer,
		ObservedPeerID: peer,
	}
}

func dreamUnavailableEvidence(workspaceID, observer, peer string) DreamStatusEvidence {
	return DreamStatusEvidence{
		Code:           "dream_unavailable",
		Reason:         "goncho_dreams scheduler table is unavailable; no background dream reasoning is active",
		WorkspaceID:    workspaceID,
		ObserverPeerID: observer,
		ObservedPeerID: peer,
	}
}

func dreamWorkUnitKey(workspaceID, observer, peer string) string {
	return "dream:" + defaultDreamType + ":" + workspaceID + ":" + observer + ":" + peer
}

func effectiveDreamNow(now time.Time) int64 {
	if now.IsZero() {
		now = time.Now()
	}
	return now.UTC().Unix()
}

func sqliteTableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var found string
	err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("goncho: sqlite table %s: %w", name, err)
	}
	return found == name, nil
}
