package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/dreamscheduler"
)

type conclusionModule struct {
	db           *sql.DB
	workspaceID  string
	observer     string
	dreamEnabled bool
}

func (s *Service) conclusions() conclusionModule {
	return conclusionModule{
		db:           s.db,
		workspaceID:  s.workspaceID,
		observer:     s.observer,
		dreamEnabled: s.dreamEnabled,
	}
}

func (c conclusionModule) Conclude(ctx context.Context, params ConcludeParams) (ConcludeResult, error) {
	peer := strings.TrimSpace(params.Peer)
	if peer == "" {
		return ConcludeResult{}, fmt.Errorf("goncho: peer is required")
	}
	profileID := strings.TrimSpace(params.ProfileID)
	memoryScope := normalizeMemoryScope(params.Scope, profileID)
	if params.DeleteID > 0 {
		deleted, err := deleteConclusion(ctx, c.db, c.workspaceID, profileID, c.observer, peer, params.DeleteID)
		if err != nil {
			return ConcludeResult{}, err
		}
		if !deleted {
			return ConcludeResult{}, fmt.Errorf("goncho: conclusion %d not found", params.DeleteID)
		}
		return ConcludeResult{
			WorkspaceID: c.workspaceID,
			ProfileID:   profileID,
			Peer:        peer,
			ID:          params.DeleteID,
			Status:      "processed",
			Deleted:     true,
		}, nil
	}

	conclusion := strings.TrimSpace(params.Conclusion)
	if conclusion == "" {
		return ConcludeResult{}, fmt.Errorf("goncho: conclusion is required when delete_id is absent")
	}

	idempotencyKey := makeIdempotencyKey(c.workspaceID, profileID, c.observer, peer, params.SessionKey, conclusion)
	id, status, err := upsertConclusion(ctx, c.db, conclusionRow{
		WorkspaceID:    c.workspaceID,
		ProfileID:      profileID,
		ObserverPeerID: c.observer,
		PeerID:         peer,
		SessionKey:     params.SessionKey,
		Content:        conclusion,
		Kind:           "manual",
		Status:         "processed",
		Source:         "manual",
		IdempotencyKey: idempotencyKey,
		EvidenceJSON:   "[]",
		Scope:          memoryScope,
	})
	if err != nil {
		return ConcludeResult{}, err
	}
	if err := storeConclusionFactAnnotations(ctx, c.db, c.workspaceID, profileID, c.observer, peer, id, conclusionFactAnnotations(conclusion)); err != nil {
		return ConcludeResult{}, err
	}
	if c.dreamEnabled {
		if _, err := dreamscheduler.CancelPendingForObserved(ctx, c.db, c.workspaceID, peer, time.Now().Unix(), "new_activity"); err != nil {
			return ConcludeResult{}, err
		}
	}

	return ConcludeResult{
		WorkspaceID: c.workspaceID,
		ProfileID:   profileID,
		Peer:        peer,
		ID:          id,
		Status:      status,
	}, nil
}
