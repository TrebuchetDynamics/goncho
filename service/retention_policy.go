package goncho

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/importance"
)

const RetentionActionArchive RetentionAction = "archive"

var retentionAuditDDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_retention_audit (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id TEXT NOT NULL,
		stable_id TEXT NOT NULL,
		target_type TEXT NOT NULL,
		action TEXT NOT NULL,
		reason TEXT NOT NULL,
		applied_by TEXT NOT NULL,
		created_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_retention_audit_stable_id ON goncho_retention_audit(workspace_id, stable_id, created_at DESC)`,
}

type RetentionPolicy struct {
	Now                    time.Time
	MinAge                 time.Duration
	MinEffectiveImportance float64
	Limit                  int

	KeepForever        bool          `json:"keep_forever,omitempty"`
	MaxAge             time.Duration `json:"max_age,omitempty"`
	MaxDBBytes         int64         `json:"max_db_bytes,omitempty"`
	MaxImageBytes      int64         `json:"max_image_bytes,omitempty"`
	MaxVectorBytes     int64         `json:"max_vector_bytes,omitempty"`
	PerWorkspaceLimit  int           `json:"per_workspace_limit,omitempty"`
	ArchiveBeforeEvict bool          `json:"archive_before_evict,omitempty"`
	ImageDir           string        `json:"image_dir,omitempty"`
	VectorDir          string        `json:"vector_dir,omitempty"`
}

type RetentionPreview struct {
	WorkspaceID string               `json:"workspace_id"`
	Mutates     bool                 `json:"mutates"`
	Policy      RetentionPolicyView  `json:"policy"`
	Disk        DiskUsageDiagnostics `json:"disk"`
	Candidates  []EvictionCandidate  `json:"candidates"`
}

type RetentionPolicyView struct {
	KeepForever        bool   `json:"keep_forever,omitempty"`
	MaxAgeSeconds      int64  `json:"max_age_seconds,omitempty"`
	MaxDBBytes         int64  `json:"max_db_bytes,omitempty"`
	MaxImageBytes      int64  `json:"max_image_bytes,omitempty"`
	MaxVectorBytes     int64  `json:"max_vector_bytes,omitempty"`
	PerWorkspaceLimit  int    `json:"per_workspace_limit,omitempty"`
	ArchiveBeforeEvict bool   `json:"archive_before_evict,omitempty"`
	ImageDir           string `json:"image_dir,omitempty"`
	VectorDir          string `json:"vector_dir,omitempty"`
}

type EvictionCandidate struct {
	StableID    string          `json:"stable_id"`
	TargetType  string          `json:"target_type"`
	Action      RetentionAction `json:"action"`
	WorkspaceID string          `json:"workspace_id"`
	PeerID      string          `json:"peer_id,omitempty"`
	SessionKey  string          `json:"session_key,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	Bytes       int64           `json:"bytes,omitempty"`
	Reasons     []string        `json:"reasons"`
	Preview     string          `json:"preview,omitempty"`
}

type RetentionApplyResult struct {
	WorkspaceID    string              `json:"workspace_id"`
	Mutates        bool                `json:"mutates"`
	AppliedBy      string              `json:"applied_by"`
	Applied        []EvictionCandidate `json:"applied"`
	ImagesArchived int                 `json:"images_archived,omitempty"`
	VectorCleanup  int                 `json:"vector_cleanup,omitempty"`
}

type RetentionAuditQuery struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	StableID    string `json:"stable_id,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type RetentionAuditEvent struct {
	ID          int64           `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	StableID    string          `json:"stable_id"`
	TargetType  string          `json:"target_type"`
	Action      RetentionAction `json:"action"`
	Reason      string          `json:"reason"`
	AppliedBy   string          `json:"applied_by"`
	CreatedAt   time.Time       `json:"created_at"`
}

type RetentionAuditResult struct {
	Events []RetentionAuditEvent `json:"events"`
}

type DiskUsageDiagnostics struct {
	DB      DiskUsageComponent `json:"db"`
	Images  DiskUsageComponent `json:"images"`
	Vectors DiskUsageComponent `json:"vectors"`
}

type DiskUsageComponent struct {
	Path       string `json:"path,omitempty"`
	Bytes      int64  `json:"bytes"`
	LimitBytes int64  `json:"limit_bytes,omitempty"`
	OverBudget bool   `json:"over_budget,omitempty"`
}

func (s *ImportanceScorer) ReviewRetentionCandidates(entries []MemoryToolEntry, policy RetentionPolicy) []RetentionCandidate {
	candidates := s.module().ReviewRetentionCandidates(toImportanceEntries(entries), importance.RetentionPolicy{Now: policy.Now, MinAge: policy.MinAge, MinEffectiveImportance: policy.MinEffectiveImportance, Limit: policy.Limit})
	out := make([]RetentionCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, RetentionCandidate{
			Entry:               fromImportanceEntry(candidate.Entry),
			Age:                 candidate.Age,
			EffectiveImportance: candidate.EffectiveImportance,
			Action:              RetentionAction(candidate.Action),
			Reason:              candidate.Reason,
		})
	}
	return out
}

func (s *Service) PreviewRetention(ctx context.Context, policy RetentionPolicy) (RetentionPreview, error) {
	if s == nil || s.db == nil {
		return RetentionPreview{}, fmt.Errorf("goncho: nil service")
	}
	usage, err := s.DiskUsage(ctx, policy)
	if err != nil {
		return RetentionPreview{}, err
	}
	candidates, err := s.retentionCandidates(ctx, policy)
	if err != nil {
		return RetentionPreview{}, err
	}
	return RetentionPreview{WorkspaceID: s.workspaceID, Mutates: false, Policy: retentionPolicyView(policy), Disk: usage, Candidates: candidates}, nil
}

func (s *Service) ApplyRetention(ctx context.Context, policy RetentionPolicy, appliedBy string) (RetentionApplyResult, error) {
	if strings.TrimSpace(appliedBy) == "" {
		return RetentionApplyResult{}, fmt.Errorf("goncho: retention apply requires applied_by")
	}
	preview, err := s.PreviewRetention(ctx, policy)
	if err != nil {
		return RetentionApplyResult{}, err
	}
	result := RetentionApplyResult{WorkspaceID: s.workspaceID, Mutates: true, AppliedBy: strings.TrimSpace(appliedBy), Applied: []EvictionCandidate{}}
	for _, candidate := range preview.Candidates {
		switch candidate.TargetType {
		case "conclusion":
			id, err := retentionConclusionID(candidate.StableID)
			if err != nil {
				return RetentionApplyResult{}, err
			}
			res, err := s.db.ExecContext(ctx, `UPDATE goncho_conclusions SET status = 'archived', updated_at = ? WHERE id = ? AND workspace_id = ? AND status IN ('processed', 'active')`, time.Now().Unix(), id, s.workspaceID)
			if err != nil {
				return RetentionApplyResult{}, fmt.Errorf("goncho: archive conclusion: %w", err)
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				continue
			}
		case "image":
			id, err := retentionTypedID(candidate.StableID, "image:")
			if err != nil {
				return RetentionApplyResult{}, err
			}
			res, err := s.db.ExecContext(ctx, `UPDATE goncho_image_memories SET embedding_status = 'archived', updated_at = ? WHERE id = ? AND workspace_id = ? AND embedding_status != 'archived'`, time.Now().Unix(), id, s.workspaceID)
			if err != nil {
				return RetentionApplyResult{}, fmt.Errorf("goncho: archive image ref: %w", err)
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				continue
			}
			result.ImagesArchived++
		default:
			continue
		}
		if err := s.insertRetentionAudit(ctx, candidate, appliedBy); err != nil {
			return RetentionApplyResult{}, err
		}
		result.Applied = append(result.Applied, candidate)
	}
	return result, nil
}

func (s *Service) RetentionAudit(ctx context.Context, q RetentionAuditQuery) (RetentionAuditResult, error) {
	workspaceID := serviceObservationWorkspace(s.workspaceID, q.WorkspaceID)
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, workspace_id, stable_id, target_type, action, reason, applied_by, created_at FROM goncho_retention_audit WHERE workspace_id = ?`
	args := []any{workspaceID}
	if stableID := strings.TrimSpace(q.StableID); stableID != "" {
		query += ` AND stable_id = ?`
		args = append(args, stableID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return RetentionAuditResult{}, fmt.Errorf("goncho: retention audit: %w", err)
	}
	defer rows.Close()
	out := RetentionAuditResult{Events: []RetentionAuditEvent{}}
	for rows.Next() {
		var event RetentionAuditEvent
		var action string
		var created int64
		if err := rows.Scan(&event.ID, &event.WorkspaceID, &event.StableID, &event.TargetType, &action, &event.Reason, &event.AppliedBy, &created); err != nil {
			return RetentionAuditResult{}, fmt.Errorf("goncho: scan retention audit: %w", err)
		}
		event.Action = RetentionAction(action)
		event.CreatedAt = time.Unix(created, 0).UTC()
		out.Events = append(out.Events, event)
	}
	return out, rows.Err()
}

func (s *Service) DiskUsage(ctx context.Context, policy RetentionPolicy) (DiskUsageDiagnostics, error) {
	dbPath := databasePath(ctx, s.db)
	dbBytes, err := fileBytes(dbPath)
	if err != nil {
		return DiskUsageDiagnostics{}, err
	}
	imageBytes, err := pathBytes(policy.ImageDir)
	if err != nil {
		return DiskUsageDiagnostics{}, err
	}
	vectorBytes, err := pathBytes(policy.VectorDir)
	if err != nil {
		return DiskUsageDiagnostics{}, err
	}
	return DiskUsageDiagnostics{
		DB:      diskComponent(dbPath, dbBytes, policy.MaxDBBytes),
		Images:  diskComponent(policy.ImageDir, imageBytes, policy.MaxImageBytes),
		Vectors: diskComponent(policy.VectorDir, vectorBytes, policy.MaxVectorBytes),
	}, nil
}

func (s *Service) retentionCandidates(ctx context.Context, policy RetentionPolicy) ([]EvictionCandidate, error) {
	if policy.KeepForever {
		return []EvictionCandidate{}, nil
	}
	now := policy.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var candidates []EvictionCandidate
	if policy.MaxAge > 0 {
		cutoff := now.Add(-policy.MaxAge).Unix()
		rows, err := s.db.QueryContext(ctx, `SELECT id, peer_id, COALESCE(session_key, ''), content, created_at FROM goncho_conclusions WHERE workspace_id = ? AND observer_peer_id = ? AND status IN ('processed', 'active') AND created_at < ? ORDER BY created_at ASC`, s.workspaceID, s.observer, cutoff)
		if err != nil {
			return nil, fmt.Errorf("goncho: retention conclusion candidates: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var peer, session, content string
			var created int64
			if err := rows.Scan(&id, &peer, &session, &content, &created); err != nil {
				return nil, err
			}
			candidates = append(candidates, EvictionCandidate{StableID: "conclusion:" + strconv.FormatInt(id, 10), TargetType: "conclusion", Action: RetentionActionArchive, WorkspaceID: s.workspaceID, PeerID: peer, SessionKey: session, CreatedAt: time.Unix(created, 0).UTC(), Bytes: int64(len(content)), Reasons: []string{fmt.Sprintf("max_age exceeded: older than %s", policy.MaxAge)}, Preview: previewRetentionContent(content)})
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	if policy.MaxImageBytes > 0 {
		usage, err := s.DiskUsage(ctx, policy)
		if err != nil {
			return nil, err
		}
		if usage.Images.OverBudget {
			images, err := s.retentionImageCandidates(ctx, "image_dir over max_image_bytes")
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, images...)
		}
	}
	if policy.PerWorkspaceLimit > 0 && len(candidates) > policy.PerWorkspaceLimit {
		candidates = candidates[:policy.PerWorkspaceLimit]
	}
	return candidates, nil
}

func (s *Service) retentionImageCandidates(ctx context.Context, reason string) ([]EvictionCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, peer_id, COALESCE(session_key, ''), image_ref, updated_at FROM goncho_image_memories WHERE workspace_id = ? AND embedding_status != 'archived' ORDER BY updated_at ASC`, s.workspaceID)
	if err != nil {
		return nil, fmt.Errorf("goncho: retention image candidates: %w", err)
	}
	defer rows.Close()
	out := []EvictionCandidate{}
	for rows.Next() {
		var id, updated int64
		var peer, session, ref string
		if err := rows.Scan(&id, &peer, &session, &ref, &updated); err != nil {
			return nil, err
		}
		out = append(out, EvictionCandidate{StableID: "image:" + strconv.FormatInt(id, 10), TargetType: "image", Action: RetentionActionArchive, WorkspaceID: s.workspaceID, PeerID: peer, SessionKey: session, CreatedAt: time.Unix(updated, 0).UTC(), Reasons: []string{reason}, Preview: ref})
	}
	return out, rows.Err()
}

func (s *Service) insertRetentionAudit(ctx context.Context, candidate EvictionCandidate, appliedBy string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO goncho_retention_audit(workspace_id, stable_id, target_type, action, reason, applied_by, created_at) VALUES(?, ?, ?, ?, ?, ?, ?)`, s.workspaceID, candidate.StableID, candidate.TargetType, string(candidate.Action), strings.Join(candidate.Reasons, "; "), strings.TrimSpace(appliedBy), time.Now().Unix())
	if err != nil {
		return fmt.Errorf("goncho: insert retention audit: %w", err)
	}
	return nil
}

func retentionPolicyView(policy RetentionPolicy) RetentionPolicyView {
	return RetentionPolicyView{KeepForever: policy.KeepForever, MaxAgeSeconds: int64(policy.MaxAge.Seconds()), MaxDBBytes: policy.MaxDBBytes, MaxImageBytes: policy.MaxImageBytes, MaxVectorBytes: policy.MaxVectorBytes, PerWorkspaceLimit: policy.PerWorkspaceLimit, ArchiveBeforeEvict: policy.ArchiveBeforeEvict, ImageDir: policy.ImageDir, VectorDir: policy.VectorDir}
}

func retentionConclusionID(stableID string) (int64, error) {
	return retentionTypedID(stableID, "conclusion:")
}

func retentionTypedID(stableID, prefix string) (int64, error) {
	if !strings.HasPrefix(stableID, prefix) {
		return 0, fmt.Errorf("goncho: invalid retention stable id %q", stableID)
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(stableID, prefix), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("goncho: parse retention stable id %q: %w", stableID, err)
	}
	return id, nil
}

func previewRetentionContent(content string) string {
	content = strings.TrimSpace(content)
	if len(content) <= 120 {
		return content
	}
	return content[:120] + "…"
}

func fileBytes(path string) (int64, error) {
	if strings.TrimSpace(path) == "" {
		return 0, nil
	}
	info, err := os.Stat(path)
	if errorsIsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("goncho: stat %s: %w", path, err)
	}
	if info.IsDir() {
		return pathBytes(path)
	}
	return info.Size(), nil
}

func pathBytes(path string) (int64, error) {
	if strings.TrimSpace(path) == "" {
		return 0, nil
	}
	var total int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if errorsIsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("goncho: walk disk usage %s: %w", path, err)
	}
	return total, nil
}

func errorsIsNotExist(err error) bool { return err != nil && os.IsNotExist(err) }

func diskComponent(path string, bytes, limit int64) DiskUsageComponent {
	return DiskUsageComponent{Path: path, Bytes: bytes, LimitBytes: limit, OverBudget: limit > 0 && bytes > limit}
}
