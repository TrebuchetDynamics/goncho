package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/observationlog"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

var (
	ErrObservationConflict      = observationlog.ErrObservationConflict
	ErrObservationNotFound      = observationlog.ErrObservationNotFound
	ErrObservationSchemaMissing = observationlog.ErrObservationSchemaMissing
	ErrObservationInvalid       = observationlog.ErrObservationInvalid
)

type ObservationKind = observationlog.ObservationKind

const (
	ObservationKindSessionStart      ObservationKind = observationlog.ObservationKindSessionStart
	ObservationKindUserPrompt        ObservationKind = observationlog.ObservationKindUserPrompt
	ObservationKindToolCall          ObservationKind = observationlog.ObservationKindToolCall
	ObservationKindToolResult        ObservationKind = observationlog.ObservationKindToolResult
	ObservationKindToolError         ObservationKind = observationlog.ObservationKindToolError
	ObservationKindAssistantResponse ObservationKind = observationlog.ObservationKindAssistantResponse
	ObservationKindCompact           ObservationKind = observationlog.ObservationKindCompact
	ObservationKindSessionEnd        ObservationKind = observationlog.ObservationKindSessionEnd
	ObservationKindCustom            ObservationKind = observationlog.ObservationKindCustom
)

type ObservationParams = observationlog.ObservationParams

type Observation = observationlog.Observation

type ObservationResult = observationlog.ObservationResult

type ObservationQuery = observationlog.ObservationQuery

type ObservationList = observationlog.ObservationList

func Observe(ctx context.Context, db *sql.DB, p ObservationParams) (ObservationResult, error) {
	return observationlog.Observe(ctx, db, p)
}

func ListObservations(ctx context.Context, db *sql.DB, q ObservationQuery) (ObservationList, error) {
	return observationlog.ListObservations(ctx, db, q)
}

func (s *Service) Observe(ctx context.Context, p ObservationParams) (ObservationResult, error) {
	if s == nil {
		return ObservationResult{}, fmt.Errorf("%w: nil service", ErrObservationInvalid)
	}
	if textutil.EqualTrimmed(p.WorkspaceID, "*") {
		return ObservationResult{}, fmt.Errorf("%w: wildcard workspace is not valid for observe", ErrObservationInvalid)
	}
	if strings.TrimSpace(p.WorkspaceID) == "" {
		p.WorkspaceID = s.workspaceID
	}
	return Observe(ctx, s.db, p)
}

func (s *Service) ListObservations(ctx context.Context, q ObservationQuery) (ObservationList, error) {
	if s == nil {
		return ObservationList{}, fmt.Errorf("%w: nil service", ErrObservationInvalid)
	}
	q.WorkspaceID = serviceObservationWorkspace(s.workspaceID, q.WorkspaceID)
	return ListObservations(ctx, s.db, q)
}

func serviceObservationWorkspace(defaultWorkspace, requested string) string {
	requested = strings.TrimSpace(requested)
	if requested == "*" {
		return ""
	}
	if requested == "" {
		return defaultWorkspace
	}
	return requested
}
