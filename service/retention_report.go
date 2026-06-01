package goncho

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/idutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/timeutil"
)

type RetentionAccessReportParams struct {
	Now            time.Time     `json:"-"`
	StaleAfter     time.Duration `json:"-"`
	OversizedBytes int           `json:"oversized_bytes,omitempty"`
	BudgetBytes    int64         `json:"budget_bytes,omitempty"`
	Limit          int           `json:"limit,omitempty"`
}

type RetentionAccessReport struct {
	Status      string                      `json:"status"`
	Mutates     bool                        `json:"mutates"`
	WorkspaceID string                      `json:"workspace_id"`
	GeneratedAt time.Time                   `json:"generated_at"`
	Policy      RetentionAccessReportPolicy `json:"policy"`
	Counts      map[string]int              `json:"counts"`
	Items       []RetentionAccessReportItem `json:"items"`
}

type RetentionAccessReportPolicy struct {
	StaleAfterSeconds int64 `json:"stale_after_seconds"`
	OversizedBytes    int   `json:"oversized_bytes"`
	BudgetBytes       int64 `json:"budget_bytes,omitempty"`
}

type RetentionAccessReportItem struct {
	StableID    string   `json:"stable_id"`
	TargetType  string   `json:"target_type"`
	WorkspaceID string   `json:"workspace_id"`
	PeerID      string   `json:"peer_id,omitempty"`
	SessionKey  string   `json:"session_key,omitempty"`
	Categories  []string `json:"categories"`
	Bytes       int64    `json:"bytes,omitempty"`
	AgeSeconds  int64    `json:"age_seconds,omitempty"`
	ReviewOpen  bool     `json:"review_open,omitempty"`
	Reasons     []string `json:"reasons"`
	Preview     string   `json:"preview,omitempty"`
}

func (s *Service) RetentionAccessReport(ctx context.Context, params RetentionAccessReportParams) (RetentionAccessReport, error) {
	if err := ctx.Err(); err != nil {
		return RetentionAccessReport{}, err
	}
	if s == nil || s.db == nil {
		return RetentionAccessReport{}, fmt.Errorf("goncho: nil service")
	}
	now := params.Now
	if now.IsZero() {
		now = timeutil.NowUTC()
	}
	staleAfter := params.StaleAfter
	if staleAfter <= 0 {
		staleAfter = 30 * 24 * time.Hour
	}
	oversized := params.OversizedBytes
	if oversized <= 0 {
		oversized = 512
	}
	limit := limitutil.Default(params.Limit, 100)
	openReviews, err := s.openReviewSubjects(ctx)
	if err != nil {
		return RetentionAccessReport{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, peer_id, COALESCE(session_key,''), content, created_at FROM goncho_conclusions WHERE workspace_id = ? AND observer_peer_id = ? AND status IN ('processed','active') ORDER BY created_at ASC, id ASC LIMIT ?`, s.workspaceID, s.observer, limit)
	if err != nil {
		return RetentionAccessReport{}, fmt.Errorf("goncho: retention access report conclusions: %w", err)
	}
	defer rows.Close()
	items := []RetentionAccessReportItem{}
	counts := map[string]int{"least_used": 0, "stale": 0, "high_risk": 0, "oversized": 0, "unreviewed": 0, "over_budget": 0}
	var totalBytes int64
	for rows.Next() {
		var id int64
		var peer, session, content string
		var createdUnix int64
		if err := rows.Scan(&id, &peer, &session, &content, &createdUnix); err != nil {
			return RetentionAccessReport{}, fmt.Errorf("goncho: scan retention report conclusion: %w", err)
		}
		stableID := "conclusion:" + idutil.Decimal(id)
		created := time.Unix(createdUnix, 0).UTC()
		age := now.Sub(created)
		bytes := int64(len(content))
		totalBytes += bytes
		item := RetentionAccessReportItem{StableID: stableID, TargetType: "conclusion", WorkspaceID: s.workspaceID, PeerID: peer, SessionKey: session, Bytes: bytes, AgeSeconds: int64(age.Seconds()), Preview: textutilPreview(content)}
		item.addCategory(counts, "least_used", "no access telemetry has promoted this memory")
		if age > staleAfter {
			item.addCategory(counts, "stale", fmt.Sprintf("age_seconds=%d exceeds stale_after_seconds=%d", int64(age.Seconds()), int64(staleAfter.Seconds())))
		}
		if bytes > int64(oversized) {
			item.addCategory(counts, "oversized", fmt.Sprintf("bytes=%d exceeds oversized_bytes=%d", bytes, oversized))
		}
		if highRiskRetentionContent(content) {
			item.addCategory(counts, "high_risk", "content resembles secret, token, password, or credential memory")
		}
		if openReviews[stableID] {
			item.ReviewOpen = true
			item.addCategory(counts, "unreviewed", "open review item references this memory")
		}
		if len(item.Categories) > 0 {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return RetentionAccessReport{}, fmt.Errorf("goncho: iterate retention report conclusions: %w", err)
	}
	if params.BudgetBytes > 0 && totalBytes > params.BudgetBytes {
		counts["over_budget"] = 1
		items = append(items, RetentionAccessReportItem{StableID: "workspace:" + s.workspaceID, TargetType: "workspace", WorkspaceID: s.workspaceID, Categories: []string{"over_budget"}, Bytes: totalBytes, Reasons: []string{fmt.Sprintf("conclusion_bytes=%d exceeds budget_bytes=%d", totalBytes, params.BudgetBytes)}})
	}
	return RetentionAccessReport{Status: "ok", Mutates: false, WorkspaceID: s.workspaceID, GeneratedAt: now.UTC(), Policy: RetentionAccessReportPolicy{StaleAfterSeconds: int64(staleAfter.Seconds()), OversizedBytes: oversized, BudgetBytes: params.BudgetBytes}, Counts: counts, Items: items}, nil
}

func (s *Service) openReviewSubjects(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT subject_id FROM goncho_review_items WHERE workspace_id = ? AND status = 'open'`, s.workspaceID)
	if err != nil {
		return nil, fmt.Errorf("goncho: retention report open reviews: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var subject string
		if err := rows.Scan(&subject); err != nil {
			return nil, err
		}
		subject = strings.TrimSpace(subject)
		if subject != "" {
			out[subject] = true
		}
	}
	return out, rows.Err()
}

func (item *RetentionAccessReportItem) addCategory(counts map[string]int, category, reason string) {
	item.Categories = append(item.Categories, category)
	item.Reasons = append(item.Reasons, reason)
	counts[category]++
}

func highRiskRetentionContent(content string) bool {
	lower := strings.ToLower(content)
	for _, marker := range []string{"password", "secret", "token", "credential", "api key", "apikey"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func textutilPreview(content string) string {
	content = strings.TrimSpace(content)
	if len(content) <= 120 {
		return content
	}
	return content[:120]
}
