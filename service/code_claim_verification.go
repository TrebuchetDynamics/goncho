package goncho

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type VerifiedCodeContextParams struct {
	Peer       string `json:"peer"`
	SessionKey string `json:"session_key,omitempty"`
	Query      string `json:"query,omitempty"`
	RepoRoot   string `json:"repo_root"`
	MaxTokens  int    `json:"max_tokens,omitempty"`
}

type VerifiedCodeClaim struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

type VerifiedCodeContextResult struct {
	WorkspaceID    string                       `json:"workspace_id"`
	Peer           string                       `json:"peer"`
	SessionKey     string                       `json:"session_key,omitempty"`
	Representation string                       `json:"representation"`
	VerifiedClaims []VerifiedCodeClaim          `json:"verified_claims"`
	StaleClaims    []VerifiedCodeClaim          `json:"stale_claims"`
	Unavailable    []ContextUnavailableEvidence `json:"unavailable,omitempty"`
}

var codeClaimPathPattern = regexp.MustCompile(`\b[[:alnum:]_./-]+\.(?:go|ts|tsx|js|jsx|py|rs|cs|java|cpp|c|h|hpp)\b`)

func (s *Service) VerifiedCodeContext(ctx context.Context, params VerifiedCodeContextParams) (VerifiedCodeContextResult, error) {
	repoRoot := strings.TrimSpace(params.RepoRoot)
	if repoRoot == "" {
		return VerifiedCodeContextResult{}, fmt.Errorf("goncho: repo_root is required")
	}
	base, err := s.Context(ctx, ContextParams{Peer: params.Peer, Query: params.Query, SessionKey: params.SessionKey, MaxTokens: params.MaxTokens})
	if err != nil {
		return VerifiedCodeContextResult{}, err
	}
	verified := []VerifiedCodeClaim{}
	stale := []VerifiedCodeClaim{}
	for _, conclusion := range base.Conclusions {
		paths := codeClaimPathPattern.FindAllString(conclusion, -1)
		for _, path := range paths {
			claim := VerifiedCodeClaim{Path: path, Content: conclusion}
			if liveCodeClaimPathExists(repoRoot, path) {
				claim.Status = "verified"
				verified = append(verified, claim)
			} else {
				claim.Status = "stale"
				stale = append(stale, claim)
			}
		}
	}

	unavailable := append([]ContextUnavailableEvidence(nil), base.Unavailable...)
	if len(stale) > 0 {
		unavailable = append(unavailable, ContextUnavailableEvidence{
			Field:      "code_claim",
			Capability: "stale_code_claim",
			Reason:     fmt.Sprintf("%d code claim(s) referenced paths missing from live repo state", len(stale)),
		})
	}
	return VerifiedCodeContextResult{
		WorkspaceID:    base.WorkspaceID,
		Peer:           base.Peer,
		SessionKey:     base.SessionKey,
		Representation: buildVerifiedCodeRepresentation(base.Peer, verified),
		VerifiedClaims: verified,
		StaleClaims:    stale,
		Unavailable:    unavailable,
	}, nil
}

func liveCodeClaimPathExists(repoRoot, rel string) bool {
	rel = filepath.Clean(strings.TrimSpace(rel))
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false
	}
	info, err := os.Stat(filepath.Join(repoRoot, rel))
	return err == nil && !info.IsDir()
}

func buildVerifiedCodeRepresentation(peer string, verified []VerifiedCodeClaim) string {
	if len(verified) == 0 {
		return "No live-verified code claims for " + peer + "."
	}
	var b strings.Builder
	b.WriteString("Live-verified code claims for ")
	b.WriteString(peer)
	b.WriteString(":")
	for _, claim := range verified {
		b.WriteString("\n- ")
		b.WriteString(claim.Path)
		b.WriteString(": ")
		b.WriteString(claim.Content)
	}
	return b.String()
}
