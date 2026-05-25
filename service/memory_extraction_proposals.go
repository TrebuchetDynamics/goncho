package goncho

import (
	"context"
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
)

type MemoryProposalOperation string

const (
	MemoryProposalAdd       MemoryProposalOperation = "add"
	MemoryProposalUpdate    MemoryProposalOperation = "update"
	MemoryProposalSupersede MemoryProposalOperation = "supersede"
	MemoryProposalDelete    MemoryProposalOperation = "delete"
	MemoryProposalNoop      MemoryProposalOperation = "noop"
)

type MemoryProposalStatus string

const (
	MemoryProposalReady          MemoryProposalStatus = "ready"
	MemoryProposalReviewRequired MemoryProposalStatus = "review_required"
)

type MemoryProposalKind string

const (
	MemoryProposalFact       MemoryProposalKind = "fact"
	MemoryProposalPreference MemoryProposalKind = "preference"
	MemoryProposalProcedure  MemoryProposalKind = "procedure"
)

type ExtractMemoryProposalsParams struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer_id"`
	SessionKey  string `json:"session_key"`
	Window      int    `json:"window,omitempty"`
}

type ExtractMemoryProposalsResult struct {
	WorkspaceID        string           `json:"workspace_id"`
	ProfileID          string           `json:"profile_id,omitempty"`
	Peer               string           `json:"peer_id"`
	SessionKey         string           `json:"session_key"`
	Window             ProposalWindow   `json:"window"`
	Proposals          []MemoryProposal `json:"proposals"`
	ActiveMemoryWrites int              `json:"active_memory_writes"`
}

type ProposalWindow struct {
	Requested    int  `json:"requested"`
	MessageCount int  `json:"message_count"`
	Total        int  `json:"total"`
	Truncated    bool `json:"truncated"`
}

type MemoryProposal struct {
	ID           string                  `json:"id"`
	Operation    MemoryProposalOperation `json:"operation"`
	Status       MemoryProposalStatus    `json:"status"`
	Kind         MemoryProposalKind      `json:"kind"`
	WorkspaceID  string                  `json:"workspace_id"`
	ProfileID    string                  `json:"profile_id,omitempty"`
	Peer         string                  `json:"peer_id"`
	SessionKey   string                  `json:"session_key"`
	Scope        string                  `json:"scope"`
	Subject      string                  `json:"subject,omitempty"`
	Content      string                  `json:"content,omitempty"`
	Confidence   float64                 `json:"confidence"`
	ExpiryHint   string                  `json:"expiry_hint,omitempty"`
	EvidenceIDs  []string                `json:"evidence_ids"`
	RelatedIDs   []string                `json:"related_ids,omitempty"`
	ReviewItemID string                  `json:"review_item_id,omitempty"`
	ReviewReason string                  `json:"review_reason,omitempty"`
}

var durableFactPattern = regexp.MustCompile(`(?i)^(.+?)\s+(?:is|are|lives in|uses|owns)\s+(.+)$`)

func (s *Service) ExtractMemoryProposals(ctx context.Context, params ExtractMemoryProposalsParams) (ExtractMemoryProposalsResult, error) {
	if s == nil {
		return ExtractMemoryProposalsResult{}, fmt.Errorf("goncho: nil service")
	}
	workspaceID := serviceObservationWorkspace(s.workspaceID, params.WorkspaceID)
	if strings.TrimSpace(workspaceID) == "" {
		workspaceID = s.workspaceID
	}
	peer := strings.TrimSpace(params.Peer)
	if peer == "" {
		return ExtractMemoryProposalsResult{}, fmt.Errorf("goncho: peer_id is required")
	}
	sessionKey := strings.TrimSpace(params.SessionKey)
	if sessionKey == "" {
		return ExtractMemoryProposalsResult{}, fmt.Errorf("goncho: session_key is required")
	}
	messages, err := listLifecycleMessages(ctx, s.db, workspaceID, sessionKey)
	if err != nil {
		return ExtractMemoryProposalsResult{}, err
	}
	messages = filterProposalMessagesByPeer(messages, peer)
	total := len(messages)
	window := params.Window
	if window <= 0 {
		window = 20
	}
	truncated := false
	if total > window {
		messages = messages[total-window:]
		truncated = true
	}
	result := ExtractMemoryProposalsResult{
		WorkspaceID: workspaceID,
		ProfileID:   strings.TrimSpace(params.ProfileID),
		Peer:        peer,
		SessionKey:  sessionKey,
		Window: ProposalWindow{
			Requested:    window,
			MessageCount: len(messages),
			Total:        total,
			Truncated:    truncated,
		},
		Proposals:          []MemoryProposal{},
		ActiveMemoryWrites: 0,
	}
	for _, msg := range messages {
		proposal, ok := s.memoryProposalFromMessage(ctx, result, msg)
		if !ok {
			continue
		}
		if proposal.Status == MemoryProposalReviewRequired {
			item, err := s.CreateReviewItem(ctx, ReviewItemCreateParams{
				Kind:        ReviewKindConflict,
				WorkspaceID: workspaceID,
				PeerID:      peer,
				SessionKey:  sessionKey,
				SubjectID:   proposal.ID,
				RelatedID:   firstString(proposal.RelatedIDs),
				Reason:      proposal.ReviewReason,
				EvidenceIDs: proposal.EvidenceIDs,
			})
			if err != nil {
				return ExtractMemoryProposalsResult{}, err
			}
			proposal.ReviewItemID = item.ID
		}
		result.Proposals = append(result.Proposals, proposal)
	}
	return result, nil
}

func (s *Service) memoryProposalFromMessage(ctx context.Context, scope ExtractMemoryProposalsResult, msg MessageRecord) (MemoryProposal, bool) {
	content := strings.TrimSpace(msg.Content)
	evidenceIDs := []string{fmt.Sprintf("msg:%d", msg.ID)}
	base := MemoryProposal{
		Status:      MemoryProposalReady,
		Kind:        MemoryProposalFact,
		WorkspaceID: scope.WorkspaceID,
		ProfileID:   scope.ProfileID,
		Peer:        scope.Peer,
		SessionKey:  scope.SessionKey,
		Scope:       MemoryScopeWorkspace,
		Confidence:  0.82,
		EvidenceIDs: evidenceIDs,
	}
	prefix, body, marked := splitMemoryProposalMarker(content)
	if !marked {
		base.Operation = MemoryProposalNoop
		base.Content = content
		base.Confidence = 0.1
		base.ID = memoryProposalID(base.Operation, base.Kind, content, evidenceIDs)
		return base, true
	}
	base.Content = strings.TrimSpace(body)
	base.Subject = proposalSubject(base.Content)
	if base.Subject == "" {
		base.Subject = strings.TrimSpace(base.Content)
	}
	switch prefix {
	case "remember":
		base.Operation = MemoryProposalAdd
	case "update":
		base.Operation = MemoryProposalUpdate
	case "supersede":
		base.Operation = MemoryProposalSupersede
	case "forget", "delete":
		base.Operation = MemoryProposalDelete
	case "preference":
		base.Operation = MemoryProposalAdd
		base.Kind = MemoryProposalPreference
		base.Scope = MemoryScopeProfile
		base.ExpiryHint = "stable preference; review if contradicted or not observed again"
	case "procedure", "lesson":
		base.Operation = MemoryProposalAdd
		base.Kind = MemoryProposalProcedure
		base.ExpiryHint = "reusable workflow; review after failure or project change"
	}
	if proposalIsLowConfidence(base.Content) {
		base.Status = MemoryProposalReviewRequired
		base.Confidence = 0.45
		base.ReviewReason = "memory extraction proposal is low-confidence and requires operator review before promotion"
	}
	if proposalIsPrivacySensitive(base.Content) {
		base.Status = MemoryProposalReviewRequired
		base.Confidence = 0.2
		base.ReviewReason = "memory extraction proposal appears privacy-sensitive or secret-like and must not be written as active memory"
	}
	if base.Operation == MemoryProposalAdd && base.Kind == MemoryProposalFact && base.Status == MemoryProposalReady {
		if related, conflict := s.findContradictoryMemory(ctx, scope.Peer, scope.SessionKey, base.Subject, base.Content); conflict {
			base.Status = MemoryProposalReviewRequired
			base.Confidence = 0.55
			base.RelatedIDs = related
			base.ReviewReason = "memory extraction proposal contradicts existing local memory and requires review"
		}
	}
	base.ID = memoryProposalID(base.Operation, base.Kind, base.Content, evidenceIDs)
	return base, true
}

func splitMemoryProposalMarker(content string) (string, string, bool) {
	prefix, body, ok := strings.Cut(content, ":")
	if !ok {
		return "", "", false
	}
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	switch prefix {
	case "remember", "update", "supersede", "forget", "delete", "preference", "procedure", "lesson":
		return prefix, strings.TrimSpace(body), true
	default:
		return "", "", false
	}
}

func proposalSubject(content string) string {
	content = strings.Trim(strings.TrimSpace(content), ".")
	matches := durableFactPattern.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return firstWords(content, 5)
}

func (s *Service) findContradictoryMemory(ctx context.Context, peer, sessionKey, subject, content string) ([]string, bool) {
	if strings.TrimSpace(subject) == "" {
		return nil, false
	}
	found, err := s.Search(ctx, SearchParams{Peer: peer, Query: subject, Limit: 5})
	if err != nil {
		return nil, false
	}
	related := []string{}
	for _, hit := range found.Results {
		if hit.Source != "conclusion" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(hit.Content), strings.TrimSpace(content)) {
			continue
		}
		if strings.Contains(strings.ToLower(hit.Content), strings.ToLower(subject)) {
			related = append(related, fmt.Sprintf("%s:%d", hit.Source, hit.ID))
		}
	}
	return related, len(related) > 0
}

func filterProposalMessagesByPeer(messages []MessageRecord, peer string) []MessageRecord {
	out := make([]MessageRecord, 0, len(messages))
	for _, msg := range messages {
		if strings.TrimSpace(msg.Peer) == peer {
			out = append(out, msg)
		}
	}
	return out
}

func proposalIsLowConfidence(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "maybe ") || strings.Contains(lower, " might ") || strings.Contains(lower, "not sure") || strings.Contains(lower, "i think")
}

func proposalIsPrivacySensitive(content string) bool {
	lower := strings.ToLower(content)
	for _, needle := range []string{"password", "api token", "secret", "private key", "sk-live", "bearer "} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func memoryProposalID(operation MemoryProposalOperation, kind MemoryProposalKind, content string, evidence []string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(string(operation)))
	_, _ = h.Write([]byte("\x00" + string(kind) + "\x00" + content + "\x00" + strings.Join(evidence, "\x00")))
	return fmt.Sprintf("proposal_%016x", h.Sum64())
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func firstWords(content string, n int) string {
	words := strings.Fields(content)
	if len(words) <= n {
		return strings.TrimSpace(content)
	}
	return strings.Join(words[:n], " ")
}
