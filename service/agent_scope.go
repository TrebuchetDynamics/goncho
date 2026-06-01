package goncho

import "strings"

func normalizeAgentScopeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case AgentScopeIsolated:
		return AgentScopeIsolated
	default:
		return AgentScopeShared
	}
}

func (s *Service) agentScopeEvidence() *AgentScopeEvidence {
	if s == nil || s.agentScopeMode != AgentScopeIsolated || strings.TrimSpace(s.agentRoleID) == "" {
		return nil
	}
	return &AgentScopeEvidence{Mode: AgentScopeIsolated, AgentID: s.agentRoleID, Applied: true, Source: "service_config"}
}

func (s *Service) applyAgentScopeToRecallQuery(q RecallQuery) RecallQuery {
	if evidence := s.agentScopeEvidence(); evidence != nil {
		q.AgentID = evidence.AgentID
		q.AgentScopeMode = evidence.Mode
	}
	return q
}

func cloneAgentScopeEvidence(evidence *AgentScopeEvidence) *AgentScopeEvidence {
	if evidence == nil {
		return nil
	}
	clone := *evidence
	return &clone
}

func (s *Service) agentScopeRecallWarning() (RecallWarning, bool) {
	evidence := s.agentScopeEvidence()
	if evidence == nil {
		return RecallWarning{}, false
	}
	return RecallWarning{
		Code:     RecallWarningAgentScopeApplied,
		Stage:    RecallStageGenerate,
		Severity: RecallWarningInfo,
		Message:  "agent role isolation filtered recall to the configured agent role",
		Evidence: map[string]string{"agent_id": evidence.AgentID, "mode": evidence.Mode, "source": evidence.Source},
	}, true
}
