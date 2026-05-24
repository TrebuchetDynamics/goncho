package goncho

import (
	"context"
	"database/sql"

	"github.com/TrebuchetDynamics/goncho/internal/dreamscheduler"
)

type DreamScheduleParams = dreamscheduler.DreamScheduleParams

type DreamScheduleResult = dreamscheduler.DreamScheduleResult

type DreamStatusEvidence = dreamscheduler.DreamStatusEvidence

type DreamQueueStatus = dreamscheduler.DreamQueueStatus

type QueueStatusConfig = dreamscheduler.QueueStatusConfig

type dreamIntent = dreamscheduler.Intent

func (s *Service) ScheduleDream(ctx context.Context, params DreamScheduleParams) (DreamScheduleResult, error) {
	return dreamscheduler.Schedule(ctx, dreamscheduler.ScheduleConfig{
		DB:             s.db,
		WorkspaceID:    s.workspaceID,
		ObserverPeerID: s.observer,
		DreamEnabled:   s.dreamEnabled,
		IdleTimeout:    s.dreamIdle,
		MinConclusions: DefaultDreamMinConclusions,
		Cooldown:       DefaultDreamCooldown,
	}, params)
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
	return dreamscheduler.CancelPendingForObserved(ctx, s.db, s.workspaceID, observed, now, reason)
}

func dreamEvidenceFromIntent(intent dreamIntent) DreamStatusEvidence {
	return dreamscheduler.EvidenceFromIntent(intent)
}

func dreamDisabledEvidence(workspaceID, observer, peer string) DreamStatusEvidence {
	return dreamscheduler.DisabledEvidence(workspaceID, observer, peer)
}

func dreamUnavailableEvidence(workspaceID, observer, peer string) DreamStatusEvidence {
	return dreamscheduler.UnavailableEvidence(workspaceID, observer, peer)
}

func sqliteTableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	return dreamscheduler.TableExists(ctx, db, name)
}
