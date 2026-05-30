package goncho

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/goncho/service/internal/pathutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

const defaultFilesystemWatcherPreviewBytes = 4 * 1024

// FilesystemWatcherImportParams describes a bounded batch of changed files from
// a local filesystem watcher. It intentionally requires explicit include globs
// so a connector cannot silently ingest an entire project tree.
type FilesystemWatcherImportParams struct {
	WorkspaceID     string   `json:"workspace_id,omitempty"`
	ProfileID       string   `json:"profile_id,omitempty"`
	PeerID          string   `json:"peer_id"`
	SessionKey      string   `json:"session_key"`
	RootDir         string   `json:"root_dir"`
	Paths           []string `json:"paths"`
	IncludeGlobs    []string `json:"include_globs"`
	ExcludeGlobs    []string `json:"exclude_globs,omitempty"`
	ChangeKind      string   `json:"change_kind,omitempty"`
	MaxPreviewBytes int      `json:"max_preview_bytes,omitempty"`
	AllowBinary     bool     `json:"allow_binary,omitempty"`
}

type FilesystemWatcherCandidate struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
	ChangeKind   string `json:"change_kind"`
	SizeBytes    int64  `json:"size_bytes"`
	Checksum     string `json:"checksum"`
	Content      string `json:"content,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
}

type FilesystemWatcherSkipped struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path,omitempty"`
	Reason       string `json:"reason"`
}

type FilesystemWatcherPreview struct {
	Mutates         bool                         `json:"mutates"`
	RootDir         string                       `json:"root_dir"`
	IncludeGlobs    []string                     `json:"include_globs"`
	ExcludeGlobs    []string                     `json:"exclude_globs,omitempty"`
	Candidates      []FilesystemWatcherCandidate `json:"candidates"`
	Skipped         []FilesystemWatcherSkipped   `json:"skipped"`
	ImportableCount int                          `json:"importable_count"`
	SkippedCount    int                          `json:"skipped_count"`
}

type FilesystemWatcherImportResult struct {
	Mutates       bool                         `json:"mutates"`
	Preview       FilesystemWatcherPreview     `json:"preview"`
	Observations  []ObservationResult          `json:"observations"`
	ImportedCount int                          `json:"imported_count"`
	ReplayedCount int                          `json:"replayed_count"`
	Skipped       []FilesystemWatcherSkipped   `json:"skipped,omitempty"`
	Candidates    []FilesystemWatcherCandidate `json:"candidates,omitempty"`
}

func (s *Service) PreviewFilesystemWatcherImport(ctx context.Context, params FilesystemWatcherImportParams) (FilesystemWatcherPreview, error) {
	if err := ctx.Err(); err != nil {
		return FilesystemWatcherPreview{}, err
	}
	norm, err := s.normalizeFilesystemWatcherParams(params)
	if err != nil {
		return FilesystemWatcherPreview{}, err
	}
	preview := FilesystemWatcherPreview{Mutates: false, RootDir: norm.RootDir, IncludeGlobs: cloneStrings(norm.IncludeGlobs), ExcludeGlobs: cloneStrings(norm.ExcludeGlobs), Candidates: []FilesystemWatcherCandidate{}, Skipped: []FilesystemWatcherSkipped{}}
	for _, rawPath := range norm.Paths {
		candidate, skipped, err := filesystemWatcherCandidate(rawPath, norm)
		if err != nil {
			return FilesystemWatcherPreview{}, err
		}
		if skipped.Reason != "" {
			preview.Skipped = append(preview.Skipped, skipped)
			continue
		}
		preview.Candidates = append(preview.Candidates, candidate)
	}
	preview.ImportableCount = len(preview.Candidates)
	preview.SkippedCount = len(preview.Skipped)
	return preview, nil
}

func (s *Service) ImportFilesystemWatcherChanges(ctx context.Context, params FilesystemWatcherImportParams) (FilesystemWatcherImportResult, error) {
	preview, err := s.PreviewFilesystemWatcherImport(ctx, params)
	if err != nil {
		return FilesystemWatcherImportResult{}, err
	}
	norm, err := s.normalizeFilesystemWatcherParams(params)
	if err != nil {
		return FilesystemWatcherImportResult{}, err
	}
	result := FilesystemWatcherImportResult{Mutates: true, Preview: preview, Skipped: preview.Skipped, Candidates: preview.Candidates, Observations: []ObservationResult{}}
	for _, candidate := range preview.Candidates {
		obs, err := s.Observe(ctx, ObservationParams{
			ID:          filesystemWatcherObservationID(norm, candidate),
			Kind:        ObservationKindCustom,
			WorkspaceID: norm.WorkspaceID,
			ProfileID:   norm.ProfileID,
			PeerID:      norm.PeerID,
			SessionKey:  norm.SessionKey,
			Input:       candidate.Content,
			Success:     filesystemWatcherBoolPtr(true),
			Metadata: map[string]string{
				"custom_kind":    "filesystem_watcher",
				"connector":      "filesystem_watcher",
				"change_kind":    candidate.ChangeKind,
				"path":           candidate.RelativePath,
				"checksum":       candidate.Checksum,
				"size_bytes":     fmt.Sprintf("%d", candidate.SizeBytes),
				"truncated":      fmt.Sprintf("%t", candidate.Truncated),
				"content_source": "local_file_preview",
			},
			ObservedAt: time.Now().UTC(),
			Reason:     "filesystem watcher imported changed local file as scoped observation",
		})
		if err != nil {
			return FilesystemWatcherImportResult{}, err
		}
		if obs.Replayed {
			result.ReplayedCount++
		} else {
			result.ImportedCount++
		}
		result.Observations = append(result.Observations, obs)
	}
	return result, nil
}

func (s *Service) normalizeFilesystemWatcherParams(params FilesystemWatcherImportParams) (FilesystemWatcherImportParams, error) {
	workspaceID := firstNonBlank(params.WorkspaceID, s.workspaceID)
	root, err := filepath.Abs(strings.TrimSpace(params.RootDir))
	if err != nil || strings.TrimSpace(params.RootDir) == "" {
		return FilesystemWatcherImportParams{}, fmt.Errorf("goncho: filesystem watcher root_dir is required")
	}
	peer := strings.TrimSpace(params.PeerID)
	session := strings.TrimSpace(params.SessionKey)
	if workspaceID == "" || peer == "" || session == "" {
		return FilesystemWatcherImportParams{}, fmt.Errorf("goncho: filesystem watcher workspace_id, peer_id, and session_key are required")
	}
	include := normalizeWatcherGlobs(params.IncludeGlobs)
	if len(include) == 0 {
		return FilesystemWatcherImportParams{}, fmt.Errorf("goncho: filesystem watcher include_globs are required")
	}
	changeKind := strings.TrimSpace(params.ChangeKind)
	if changeKind == "" {
		changeKind = "file_change"
	}
	maxPreview := params.MaxPreviewBytes
	if maxPreview <= 0 {
		maxPreview = defaultFilesystemWatcherPreviewBytes
	}
	return FilesystemWatcherImportParams{WorkspaceID: workspaceID, ProfileID: strings.TrimSpace(params.ProfileID), PeerID: peer, SessionKey: session, RootDir: root, Paths: cloneStrings(params.Paths), IncludeGlobs: include, ExcludeGlobs: normalizeWatcherGlobs(params.ExcludeGlobs), ChangeKind: changeKind, MaxPreviewBytes: maxPreview, AllowBinary: params.AllowBinary}, nil
}

func filesystemWatcherCandidate(rawPath string, params FilesystemWatcherImportParams) (FilesystemWatcherCandidate, FilesystemWatcherSkipped, error) {
	absPath, err := filepath.Abs(strings.TrimSpace(rawPath))
	if err != nil || strings.TrimSpace(rawPath) == "" {
		return FilesystemWatcherCandidate{}, FilesystemWatcherSkipped{Path: rawPath, Reason: "invalid_path"}, nil
	}
	rel, err := filepath.Rel(params.RootDir, absPath)
	if err != nil || pathutil.IsUnsafeRelative(rel) {
		return FilesystemWatcherCandidate{}, FilesystemWatcherSkipped{Path: absPath, Reason: "outside_root"}, nil
	}
	rel = pathutil.CleanSlashPath(rel)
	if rel == "." || rel == "" {
		return FilesystemWatcherCandidate{}, FilesystemWatcherSkipped{Path: absPath, RelativePath: rel, Reason: "not_file"}, nil
	}
	if matchesAnyWatcherGlob(rel, params.ExcludeGlobs) {
		return FilesystemWatcherCandidate{}, FilesystemWatcherSkipped{Path: absPath, RelativePath: rel, Reason: "excluded"}, nil
	}
	if !matchesAnyWatcherGlob(rel, params.IncludeGlobs) {
		return FilesystemWatcherCandidate{}, FilesystemWatcherSkipped{Path: absPath, RelativePath: rel, Reason: "not_included"}, nil
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return FilesystemWatcherCandidate{}, FilesystemWatcherSkipped{Path: absPath, RelativePath: rel, Reason: "unreadable"}, nil
	}
	if info.IsDir() {
		return FilesystemWatcherCandidate{}, FilesystemWatcherSkipped{Path: absPath, RelativePath: rel, Reason: "not_file"}, nil
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return FilesystemWatcherCandidate{}, FilesystemWatcherSkipped{Path: absPath, RelativePath: rel, Reason: "unreadable"}, nil
	}
	if !params.AllowBinary && looksBinary(raw) {
		return FilesystemWatcherCandidate{}, FilesystemWatcherSkipped{Path: absPath, RelativePath: rel, Reason: "binary"}, nil
	}
	sum := sha256.Sum256(raw)
	content := string(raw)
	if !utf8.Valid(raw) {
		content = hex.EncodeToString(raw)
	}
	truncated := false
	if len([]byte(content)) > params.MaxPreviewBytes {
		content = truncateFilesystemWatcherUTF8(content, params.MaxPreviewBytes)
		truncated = true
	}
	return FilesystemWatcherCandidate{Path: absPath, RelativePath: rel, ChangeKind: params.ChangeKind, SizeBytes: info.Size(), Checksum: hex.EncodeToString(sum[:]), Content: content, Truncated: truncated}, FilesystemWatcherSkipped{}, nil
}

func normalizeWatcherGlobs(values []string) []string {
	return textutil.NormalizeUnique(values, pathutil.NormalizeSlashPattern, false)
}

func matchesAnyWatcherGlob(rel string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchWatcherGlob(rel, pattern) {
			return true
		}
	}
	return false
}

func matchWatcherGlob(rel, pattern string) bool {
	rel = pathutil.NormalizeSlashPattern(rel)
	pattern = pathutil.NormalizeSlashPattern(pattern)
	base := pathutil.SlashBase(rel)
	if pattern == rel || pattern == base || pattern == "**" || pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return rel == prefix || strings.HasPrefix(rel, prefix+"/")
	}
	if strings.HasPrefix(pattern, "**/") {
		tail := strings.TrimPrefix(pattern, "**/")
		if ok, _ := filepath.Match(tail, base); ok {
			return true
		}
		return strings.HasSuffix(rel, strings.TrimPrefix(tail, "*"))
	}
	if ok, _ := filepath.Match(pattern, rel); ok {
		return true
	}
	if ok, _ := filepath.Match(pattern, base); ok {
		return true
	}
	return false
}

func looksBinary(raw []byte) bool {
	for _, b := range raw {
		if b == 0 {
			return true
		}
	}
	return !utf8.Valid(raw)
}

func filesystemWatcherObservationID(params FilesystemWatcherImportParams, candidate FilesystemWatcherCandidate) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{params.WorkspaceID, params.ProfileID, params.PeerID, params.SessionKey, candidate.RelativePath, candidate.Checksum}, "\x00")))
	return "fswatch_" + hex.EncodeToString(sum[:16])
}

func truncateFilesystemWatcherUTF8(value string, limit int) string {
	return textutil.TruncateUTF8Bytes(value, limit)
}

func filesystemWatcherBoolPtr(v bool) *bool { return &v }
