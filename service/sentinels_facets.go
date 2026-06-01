package goncho

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/hashutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sqlutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/timeutil"
)

var sentinelFacetDDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_sentinels (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		peer_id TEXT NOT NULL DEFAULT '',
		scope TEXT NOT NULL DEFAULT 'workspace',
		expected TEXT NOT NULL,
		condition TEXT NOT NULL DEFAULT 'must_include',
		status TEXT NOT NULL DEFAULT 'active',
		review_status TEXT NOT NULL DEFAULT 'reviewed',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_sentinels_workspace_peer ON goncho_sentinels(workspace_id, peer_id, status)`,
	`CREATE TABLE IF NOT EXISTS goncho_memory_facets (
		workspace_id TEXT NOT NULL,
		stable_id TEXT NOT NULL,
		facet TEXT NOT NULL,
		value TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL,
		PRIMARY KEY(workspace_id, stable_id, facet, value)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_memory_facets_lookup ON goncho_memory_facets(workspace_id, facet, value)`,
}

type SentinelParams struct {
	ID        string `json:"id"`
	Peer      string `json:"peer,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Expected  string `json:"expected"`
	Condition string `json:"condition,omitempty"`
}

type SentinelRecord struct {
	ID           string    `json:"id"`
	WorkspaceID  string    `json:"workspace_id"`
	Peer         string    `json:"peer,omitempty"`
	Scope        string    `json:"scope"`
	Expected     string    `json:"expected"`
	Condition    string    `json:"condition"`
	Status       string    `json:"status"`
	ReviewStatus string    `json:"review_status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type FacetParams struct {
	StableID string `json:"stable_id"`
	Facet    string `json:"facet"`
	Value    string `json:"value,omitempty"`
}

type ViewerMemoryReport struct {
	Status      string              `json:"status"`
	ReadOnly    bool                `json:"read_only"`
	WorkspaceID string              `json:"workspace_id"`
	Facet       string              `json:"facet,omitempty"`
	Value       string              `json:"value,omitempty"`
	AgentScope  *AgentScopeEvidence `json:"agent_scope,omitempty"`
	Items       []ViewerConclusion  `json:"items"`
	GeneratedAt time.Time           `json:"generated_at"`
}

func (s *Service) UpsertSentinel(ctx context.Context, params SentinelParams) (SentinelRecord, error) {
	if s == nil || s.db == nil {
		return SentinelRecord{}, fmt.Errorf("goncho: nil service")
	}
	expected := strings.TrimSpace(params.Expected)
	if expected == "" {
		return SentinelRecord{}, fmt.Errorf("goncho: sentinel expected text is required")
	}
	id := strings.TrimSpace(params.ID)
	if id == "" {
		id = "sentinel:" + hashutil.SHA256HexStringPrefix(expected, 12)
	}
	scope := strings.TrimSpace(params.Scope)
	if scope == "" {
		scope = MemoryScopeWorkspace
	}
	condition := strings.TrimSpace(params.Condition)
	if condition == "" {
		condition = "must_include"
	}
	now := timeutil.NowUTC().Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO goncho_sentinels(id, workspace_id, peer_id, scope, expected, condition, status, review_status, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET peer_id=excluded.peer_id, scope=excluded.scope, expected=excluded.expected, condition=excluded.condition, status='active', review_status='reviewed', updated_at=excluded.updated_at`, id, s.workspaceID, strings.TrimSpace(params.Peer), scope, expected, condition, "active", "reviewed", now, now)
	if err != nil {
		return SentinelRecord{}, fmt.Errorf("goncho: upsert sentinel: %w", err)
	}
	return SentinelRecord{ID: id, WorkspaceID: s.workspaceID, Peer: strings.TrimSpace(params.Peer), Scope: scope, Expected: expected, Condition: condition, Status: "active", ReviewStatus: "reviewed", CreatedAt: time.Unix(now, 0).UTC(), UpdatedAt: time.Unix(now, 0).UTC()}, nil
}

func (s *Service) UpsertMemoryFacet(ctx context.Context, params FacetParams) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("goncho: nil service")
	}
	stableID := strings.TrimSpace(params.StableID)
	facet := strings.TrimSpace(strings.ToLower(params.Facet))
	if stableID == "" || facet == "" {
		return fmt.Errorf("goncho: facet stable_id and facet are required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO goncho_memory_facets(workspace_id, stable_id, facet, value, created_at) VALUES(?,?,?,?,?)`, s.workspaceID, stableID, facet, strings.TrimSpace(strings.ToLower(params.Value)), timeutil.NowUTC().Unix())
	if err != nil {
		return fmt.Errorf("goncho: upsert memory facet: %w", err)
	}
	return nil
}

func (s *Service) sentinelContextWarnings(ctx context.Context, peer string, result ContextResult) ([]ContextUnavailableEvidence, error) {
	sentinels, err := s.activeSentinels(ctx, peer)
	if err != nil {
		return nil, err
	}
	haystack := strings.ToLower(result.Representation + "\n" + strings.Join(result.Conclusions, "\n"))
	warnings := []ContextUnavailableEvidence{}
	for _, sentinel := range sentinels {
		if strings.TrimSpace(sentinel.Expected) != "" && !strings.Contains(haystack, strings.ToLower(sentinel.Expected)) {
			warnings = append(warnings, ContextUnavailableEvidence{Field: "sentinel", Capability: "sentinel_missing", Reason: fmt.Sprintf("reviewed sentinel %s expected %q was absent from context", sentinel.ID, sentinel.Expected)})
		}
	}
	return warnings, nil
}

func (s *Service) sentinelRecallWarnings(ctx context.Context, q RecallQuery, trace RecallTrace) ([]RecallWarning, error) {
	sentinels, err := s.activeSentinels(ctx, q.Peer)
	if err != nil {
		return nil, err
	}
	haystack := strings.ToLower(q.Query + "\n" + recallTraceSelectedText(trace))
	warnings := []RecallWarning{}
	for _, sentinel := range sentinels {
		expected := strings.ToLower(sentinel.Expected)
		if expected == "" || !strings.Contains(strings.ToLower(q.Query), firstSentinelToken(expected)) {
			continue
		}
		if !strings.Contains(haystack, expected) {
			warnings = append(warnings, RecallWarning{Code: "sentinel_missing", Stage: RecallStageProject, Severity: RecallWarningDegraded, Message: "reviewed sentinel absent from selected recall context", Evidence: map[string]string{"sentinel_id": sentinel.ID, "expected": sentinel.Expected}})
		}
	}
	return warnings, nil
}

func (s *Service) activeSentinels(ctx context.Context, peer string) ([]SentinelRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, peer_id, scope, expected, condition, status, review_status, created_at, updated_at FROM goncho_sentinels WHERE workspace_id = ? AND status = 'active' AND review_status = 'reviewed' AND (peer_id = '' OR peer_id = ?) ORDER BY updated_at DESC, id ASC`, s.workspaceID, strings.TrimSpace(peer))
	if err != nil {
		if sqlutil.IsSQLiteNoSuchTableError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("goncho: list sentinels: %w", err)
	}
	defer rows.Close()
	out := []SentinelRecord{}
	for rows.Next() {
		var rec SentinelRecord
		var created, updated int64
		if err := rows.Scan(&rec.ID, &rec.WorkspaceID, &rec.Peer, &rec.Scope, &rec.Expected, &rec.Condition, &rec.Status, &rec.ReviewStatus, &created, &updated); err != nil {
			return nil, err
		}
		rec.CreatedAt = time.Unix(created, 0).UTC()
		rec.UpdatedAt = time.Unix(updated, 0).UTC()
		out = append(out, rec)
	}
	return out, rows.Err()
}

func recallTraceSelectedText(trace RecallTrace) string {
	parts := sliceutil.Map(trace.Selected, func(item ScoredRecallCandidate) string { return item.Candidate.Content })
	return strings.Join(parts, "\n")
}

func firstSentinelToken(expected string) string {
	for _, token := range strings.Fields(expected) {
		if len(token) > 2 {
			return token
		}
	}
	return expected
}

func (s *Service) ViewerMemoryReport(ctx context.Context, facet, value string, limit int) (ViewerMemoryReport, error) {
	limit = limitutil.DefaultClamped(limit, 20, 100)
	facet = strings.TrimSpace(strings.ToLower(facet))
	value = strings.TrimSpace(strings.ToLower(value))
	query := `SELECT c.id, c.workspace_id, c.profile_id, c.peer_id, COALESCE(c.session_key,''), c.content, c.scope, c.status, c.created_at FROM goncho_conclusions c`
	args := []any{s.workspaceID, s.observer}
	if facet != "" {
		query += ` JOIN goncho_memory_facets f ON f.workspace_id = c.workspace_id AND f.stable_id = ('conclusion:' || c.id) AND f.facet = ?`
		args = []any{facet, s.workspaceID, s.observer}
		if value != "" {
			query += ` AND f.value = ?`
			args = []any{facet, value, s.workspaceID, s.observer}
		}
	}
	query += ` WHERE c.workspace_id = ? AND c.observer_peer_id = ? AND c.status IN ('processed','active') ORDER BY c.created_at DESC, c.id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ViewerMemoryReport{}, fmt.Errorf("goncho: viewer memory report: %w", err)
	}
	defer rows.Close()
	items := []ViewerConclusion{}
	for rows.Next() {
		var item ViewerConclusion
		var created int64
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.ProfileID, &item.PeerID, &item.SessionKey, &item.Content, &item.Scope, &item.Status, &created); err != nil {
			return ViewerMemoryReport{}, err
		}
		item.CreatedAt = time.Unix(created, 0).UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ViewerMemoryReport{}, err
	}
	return ViewerMemoryReport{Status: "ok", ReadOnly: true, WorkspaceID: s.workspaceID, Facet: facet, Value: value, AgentScope: s.agentScopeEvidence(), Items: items, GeneratedAt: timeutil.NowUTC()}, nil
}
