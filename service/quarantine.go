package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

func importLooksLikePromptInjection(text string) bool {
	value := strings.ToLower(text)
	markers := []string{
		"ignore previous instructions",
		"ignore all previous instructions",
		"reveal all secrets",
		"exfiltrate",
		"<script",
		"system prompt",
	}
	return textutil.ContainsAnySubstring(value, markers)
}

func promptInjectionQuarantineEvidence() ContextUnavailableEvidence {
	return ContextUnavailableEvidence{
		Field:      "import",
		Capability: "prompt_injection_quarantine",
		Reason:     "imported content matched prompt-injection or script-like patterns; content was preserved as skipped evidence but excluded from trusted memory/context",
	}
}

func promptInjectionQuarantineEvidenceForSession(ctx context.Context, db *sql.DB, sessionKey string) ([]ContextUnavailableEvidence, error) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return nil, nil
	}
	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM turns
		WHERE session_id = ?
		  AND memory_sync_status = 'skipped'
		  AND memory_sync_reason = 'quarantined_prompt_injection'
	`, sessionKey).Scan(&count); err != nil {
		return nil, fmt.Errorf("goncho: count prompt-injection quarantine evidence: %w", err)
	}
	if count == 0 {
		return nil, nil
	}
	evidence := promptInjectionQuarantineEvidence()
	evidence.Reason = fmt.Sprintf("%d imported turn(s) matched prompt-injection or script-like patterns; quarantined evidence is excluded from trusted memory/context", count)
	return []ContextUnavailableEvidence{evidence}, nil
}
