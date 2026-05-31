package goncho

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

const DefaultObserverPeerID = "gormes"

var (
	ErrWorkspacePerUser               = errors.New("goncho: workspace-per-user topology is not allowed")
	ErrWorkspaceRequiresHardIsolation = errors.New("goncho: explicit workspace requires hard isolation")
	ErrWorkspaceRequired              = errors.New("goncho: hard-isolation workspace is required")
	ErrPeerIdentityRequired           = errors.New("goncho: peer identity requires user_id or source/chat_id")
	ErrSessionBoundaryRequired        = errors.New("goncho: session boundary is required")
)

const (
	EvidenceDefaultWorkspace     = "default_workspace:" + DefaultWorkspaceID
	EvidenceHardIsolation        = "workspace:hard_isolation"
	EvidenceCanonicalUserID      = "peer:session_metadata_user_id"
	EvidenceExternalPeerFallback = "peer:source_prefixed_external_fallback"
	EvidenceCrossPeerOptIn       = "observation:cross_peer_opt_in"
)

type WorkspaceStrategy string

const (
	WorkspaceStrategyDefault       WorkspaceStrategy = "default"
	WorkspaceStrategyHardIsolation WorkspaceStrategy = "hard_isolation"
	WorkspaceStrategyPerUser       WorkspaceStrategy = "per_user"
)

type WorkspaceRequest struct {
	Strategy    WorkspaceStrategy
	WorkspaceID string
	UserID      string
}

type WorkspaceDecision struct {
	WorkspaceID   string
	HardIsolation bool
	Evidence      []string
}

func ResolveWorkspace(req WorkspaceRequest) (WorkspaceDecision, error) {
	strategy := WorkspaceStrategy(textutil.LowerTrimmed(string(req.Strategy)))
	switch strategy {
	case "", WorkspaceStrategyDefault:
		if strings.TrimSpace(req.WorkspaceID) != "" {
			return WorkspaceDecision{}, fmt.Errorf("%w: %q", ErrWorkspaceRequiresHardIsolation, strings.TrimSpace(req.WorkspaceID))
		}
		return WorkspaceDecision{
			WorkspaceID: DefaultWorkspaceID,
			Evidence:    []string{EvidenceDefaultWorkspace},
		}, nil
	case WorkspaceStrategyHardIsolation:
		workspaceID := strings.TrimSpace(req.WorkspaceID)
		if workspaceID == "" {
			return WorkspaceDecision{}, ErrWorkspaceRequired
		}
		return WorkspaceDecision{
			WorkspaceID:   workspaceID,
			HardIsolation: true,
			Evidence:      []string{EvidenceHardIsolation},
		}, nil
	case WorkspaceStrategyPerUser:
		userID := strings.TrimSpace(req.UserID)
		if userID == "" {
			return WorkspaceDecision{}, ErrWorkspacePerUser
		}
		return WorkspaceDecision{}, fmt.Errorf("%w: %s", ErrWorkspacePerUser, userID)
	default:
		return WorkspaceDecision{}, fmt.Errorf("goncho: unsupported workspace strategy %q", req.Strategy)
	}
}

type PeerIdentityDecision struct {
	PeerID   string
	Degraded bool
	Evidence []string
}

func ResolvePeerID(meta SessionMetadata) (PeerIdentityDecision, error) {
	userID := strings.TrimSpace(meta.UserID)
	if userID != "" {
		return PeerIdentityDecision{
			PeerID:   userID,
			Evidence: []string{EvidenceCanonicalUserID},
		}, nil
	}

	source := textutil.LowerTrimmed(meta.Source)
	chatID := strings.TrimSpace(meta.ChatID)
	if source == "" || chatID == "" {
		return PeerIdentityDecision{}, ErrPeerIdentityRequired
	}
	return PeerIdentityDecision{
		PeerID:   source + ":" + chatID,
		Degraded: true,
		Evidence: []string{EvidenceExternalPeerFallback},
	}, nil
}

type PeerRole string

const (
	PeerRoleHuman                  PeerRole = "human"
	PeerRoleGormesAssistant        PeerRole = "gormes_assistant"
	PeerRoleDeterministicAssistant PeerRole = "deterministic_assistant"
	PeerRoleTransportBot           PeerRole = "transport_bot"
	PeerRoleImportHelper           PeerRole = "import_helper"
	PeerRoleParentAgent            PeerRole = "parent_agent"
)

type ObservationRequest struct {
	Role                 PeerRole
	CrossPeerObservation bool
}

type ObservationDecision struct {
	ObserveMe     bool
	ObserveOthers bool
	Evidence      []string
}

func DefaultObservation(req ObservationRequest) ObservationDecision {
	role := PeerRole(textutil.LowerTrimmed(string(req.Role)))
	observeMe := true
	switch role {
	case PeerRoleGormesAssistant, PeerRoleDeterministicAssistant, PeerRoleTransportBot, PeerRoleImportHelper, PeerRoleParentAgent:
		observeMe = false
	case "", PeerRoleHuman:
		observeMe = true
	default:
		observeMe = true
	}

	out := ObservationDecision{ObserveMe: observeMe}
	if req.CrossPeerObservation {
		out.ObserveOthers = true
		out.Evidence = append(out.Evidence, EvidenceCrossPeerOptIn)
	}
	return out
}

type SessionBoundaryKind string

const (
	SessionBoundaryThread            SessionBoundaryKind = "thread"
	SessionBoundaryChannel           SessionBoundaryKind = "channel"
	SessionBoundaryRepository        SessionBoundaryKind = "repository"
	SessionBoundaryImportBatch       SessionBoundaryKind = "import_batch"
	SessionBoundaryDelegatedChildRun SessionBoundaryKind = "delegated_child_run"
)

type SessionBoundaryRequest struct {
	Kind   SessionBoundaryKind
	Key    string
	Source string
}

type SessionBoundaryDecision struct {
	Kind       SessionBoundaryKind
	SessionKey string
	Evidence   []string
}

func ResolveSessionBoundary(req SessionBoundaryRequest) (SessionBoundaryDecision, error) {
	kind := SessionBoundaryKind(textutil.LowerTrimmed(string(req.Kind)))
	key := strings.TrimSpace(req.Key)
	if key == "" {
		return SessionBoundaryDecision{}, ErrSessionBoundaryRequired
	}
	source := textutil.LowerTrimmed(req.Source)

	switch kind {
	case SessionBoundaryThread:
		return SessionBoundaryDecision{Kind: kind, SessionKey: joinSessionKey(source, "thread", key)}, nil
	case SessionBoundaryChannel:
		return SessionBoundaryDecision{Kind: kind, SessionKey: joinSessionKey(source, "channel", key)}, nil
	case SessionBoundaryRepository:
		return SessionBoundaryDecision{Kind: kind, SessionKey: "repo:" + key}, nil
	case SessionBoundaryImportBatch:
		return SessionBoundaryDecision{Kind: kind, SessionKey: "import:" + key}, nil
	case SessionBoundaryDelegatedChildRun:
		return SessionBoundaryDecision{Kind: kind, SessionKey: "child-run:" + key}, nil
	case "":
		return SessionBoundaryDecision{}, ErrSessionBoundaryRequired
	default:
		return SessionBoundaryDecision{}, fmt.Errorf("goncho: unsupported session boundary %q", req.Kind)
	}
}

func joinSessionKey(source, boundary, key string) string {
	if source == "" {
		return boundary + ":" + key
	}
	return source + ":" + boundary + ":" + key
}
