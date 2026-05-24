package goncho

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/TrebuchetDynamics/goncho/internal/observationlog"
)

type AuditAction = observationlog.AuditAction

const (
	AuditActionObserve AuditAction = observationlog.AuditActionObserve
)

type AuditTargetType = observationlog.AuditTargetType

const (
	AuditTargetObservation AuditTargetType = observationlog.AuditTargetObservation
)

type AuditEvent = observationlog.AuditEvent

type AuditQuery = observationlog.AuditQuery

type AuditResult = observationlog.AuditResult

func AuditTrail(ctx context.Context, db *sql.DB, q AuditQuery) (AuditResult, error) {
	return observationlog.AuditTrail(ctx, db, q)
}

func (s *Service) AuditTrail(ctx context.Context, q AuditQuery) (AuditResult, error) {
	if s == nil {
		return AuditResult{}, fmt.Errorf("%w: nil service", ErrObservationInvalid)
	}
	q.WorkspaceID = serviceObservationWorkspace(s.workspaceID, q.WorkspaceID)
	return AuditTrail(ctx, s.db, q)
}
