package gormes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

type LiveRootReport struct {
	Root             string              `json:"root"`
	RootSessionIndex SessionIndexSummary `json:"root_session_index"`
	Profiles         []ProfileSummary    `json:"profiles"`
	MemoryFiles      []MemoryFileSummary `json:"memory_files"`
}

type ProfileSummary struct {
	ProfileID    string              `json:"profile_id"`
	SessionIndex SessionIndexSummary `json:"session_index"`
	MemoryFiles  []MemoryFileSummary `json:"memory_files"`
}

type SessionIndexSummary struct {
	Path         string    `json:"path"`
	Present      bool      `json:"present"`
	SessionCount int       `json:"session_count"`
	LineageCount int       `json:"lineage_count"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

type MemoryFileSummary struct {
	Path           string    `json:"path"`
	Kind           string    `json:"kind"`
	SizeBytes      int64     `json:"size_bytes"`
	ModifiedAt     time.Time `json:"modified_at"`
	NonEmptyLines  int       `json:"non_empty_lines"`
	HeadingCount   int       `json:"heading_count"`
	GonchoMentions int       `json:"goncho_mentions"`
	GormesMentions int       `json:"gormes_mentions"`
}

func InspectLiveRoot(ctx context.Context, root string) (LiveRootReport, error) {
	if err := ctx.Err(); err != nil {
		return LiveRootReport{}, err
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return LiveRootReport{}, fmt.Errorf("gormes live root: root path is required")
	}
	root = filepath.Clean(root)
	if info, err := os.Stat(root); err != nil {
		return LiveRootReport{}, fmt.Errorf("gormes live root: stat root: %w", err)
	} else if !info.IsDir() {
		return LiveRootReport{}, fmt.Errorf("gormes live root: root is not a directory")
	}

	report := LiveRootReport{Root: root}
	report.RootSessionIndex = inspectSessionIndex(filepath.Join(root, "sessions", "index.yaml"))
	for _, candidate := range []struct{ path, kind string }{
		{filepath.Join(root, "memory", "MEMORY.md"), "root_memory"},
		{filepath.Join(root, "memory", "USER.md"), "root_user"},
		{filepath.Join(root, "workspace", "memory", "MEMORY.md"), "workspace_template_memory"},
		{filepath.Join(root, "workspace", "memory", "USER.md"), "workspace_template_user"},
	} {
		if summary, ok := inspectMemoryFile(candidate.path, candidate.kind); ok {
			report.MemoryFiles = append(report.MemoryFiles, summary)
		}
	}

	profilesDir := filepath.Join(root, "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err == nil {
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return LiveRootReport{}, err
			}
			if !entry.IsDir() {
				continue
			}
			profileID := entry.Name()
			profileDir := filepath.Join(profilesDir, profileID)
			profile := ProfileSummary{
				ProfileID:    profileID,
				SessionIndex: inspectSessionIndex(filepath.Join(profileDir, "sessions", "index.yaml")),
			}
			for _, candidate := range []struct{ path, kind string }{
				{filepath.Join(profileDir, "GONCHO_MEMORY.md"), "profile_goncho_memory"},
				{filepath.Join(profileDir, "memory", "MEMORY.md"), "profile_memory"},
				{filepath.Join(profileDir, "memory", "USER.md"), "profile_user"},
			} {
				if summary, ok := inspectMemoryFile(candidate.path, candidate.kind); ok {
					profile.MemoryFiles = append(profile.MemoryFiles, summary)
					report.MemoryFiles = append(report.MemoryFiles, summary)
				}
			}
			report.Profiles = append(report.Profiles, profile)
		}
	}
	sort.Slice(report.Profiles, func(i, j int) bool { return report.Profiles[i].ProfileID < report.Profiles[j].ProfileID })
	sort.Slice(report.MemoryFiles, func(i, j int) bool { return report.MemoryFiles[i].Path < report.MemoryFiles[j].Path })
	return report, nil
}

func (r LiveRootReport) Profile(profileID string) (ProfileSummary, bool) {
	for _, profile := range r.Profiles {
		if profile.ProfileID == profileID {
			return profile, true
		}
	}
	return ProfileSummary{}, false
}

func (r LiveRootReport) GonchoMentionCount() int {
	total := 0
	for _, file := range r.MemoryFiles {
		total += file.GonchoMentions
	}
	return total
}

func (r LiveRootReport) SessionEvidenceInput(workspaceID string) goncho.SessionEvidenceInput {
	out := goncho.SessionEvidenceInput{WorkspaceID: strings.TrimSpace(workspaceID)}
	out.SessionIndexes = append(out.SessionIndexes, goncho.SessionEvidenceIndex{
		Scope:        goncho.SessionEvidenceScopeRoot,
		Path:         r.RootSessionIndex.Path,
		SessionCount: r.RootSessionIndex.SessionCount,
		LineageCount: r.RootSessionIndex.LineageCount,
		UpdatedAt:    r.RootSessionIndex.UpdatedAt,
	})
	for _, profile := range r.Profiles {
		out.SessionIndexes = append(out.SessionIndexes, goncho.SessionEvidenceIndex{
			Scope:        goncho.SessionEvidenceScopeProfile,
			ProfileID:    profile.ProfileID,
			Path:         profile.SessionIndex.Path,
			SessionCount: profile.SessionIndex.SessionCount,
			LineageCount: profile.SessionIndex.LineageCount,
			UpdatedAt:    profile.SessionIndex.UpdatedAt,
		})
	}
	for _, file := range r.MemoryFiles {
		out.MemoryFiles = append(out.MemoryFiles, goncho.SessionEvidenceMemoryFile{
			Scope:          sessionEvidenceScopeForMemoryKind(file.Kind),
			ProfileID:      profileIDForMemoryPath(r.Profiles, file.Path),
			Path:           file.Path,
			Kind:           file.Kind,
			SizeBytes:      file.SizeBytes,
			ModifiedAt:     file.ModifiedAt,
			NonEmptyLines:  file.NonEmptyLines,
			HeadingCount:   file.HeadingCount,
			GonchoMentions: file.GonchoMentions,
			GormesMentions: file.GormesMentions,
		})
	}
	return out
}

func sessionEvidenceScopeForMemoryKind(kind string) goncho.SessionEvidenceScope {
	switch kind {
	case "profile_goncho_memory", "profile_memory", "profile_user":
		return goncho.SessionEvidenceScopeProfile
	case "workspace_template_memory", "workspace_template_user":
		return goncho.SessionEvidenceScopeWorkspaceTemplate
	default:
		return goncho.SessionEvidenceScopeRoot
	}
}

func profileIDForMemoryPath(profiles []ProfileSummary, path string) string {
	for _, profile := range profiles {
		for _, file := range profile.MemoryFiles {
			if file.Path == path {
				return profile.ProfileID
			}
		}
	}
	return ""
}

func (r LiveRootReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "root=%s root_sessions=%d root_lineage=%d profiles=%d memory_files=%d goncho_mentions=%d", r.Root, r.RootSessionIndex.SessionCount, r.RootSessionIndex.LineageCount, len(r.Profiles), len(r.MemoryFiles), r.GonchoMentionCount())
	for _, profile := range r.Profiles {
		fmt.Fprintf(&b, " profile[%s]=sessions:%d,lineage:%d,memory:%d", profile.ProfileID, profile.SessionIndex.SessionCount, profile.SessionIndex.LineageCount, len(profile.MemoryFiles))
	}
	return b.String()
}

func inspectSessionIndex(path string) SessionIndexSummary {
	out := SessionIndexSummary{Path: path}
	content, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	out.Present = true
	text := string(content)
	out.SessionCount = countYAMLMapEntries(text, "sessions")
	out.LineageCount = countYAMLMapEntries(text, "lineage")
	if raw := regexp.MustCompile(`(?m)^updated_at:\s*(.+)$`).FindStringSubmatch(text); len(raw) == 2 {
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw[1])); err == nil {
			out.UpdatedAt = parsed
		}
	}
	return out
}

func countYAMLMapEntries(text, section string) int {
	lines := strings.Split(text, "\n")
	sectionIndex := -1
	for i, line := range lines {
		if line == section+":" {
			sectionIndex = i
			break
		}
	}
	if sectionIndex == -1 {
		return 0
	}
	count := 0
	for _, line := range lines[sectionIndex+1:] {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			break
		}
		if regexp.MustCompile(`^\s{2}[^\s#][^:]*:\s*(?:\S.*)?$`).MatchString(line) {
			count++
		}
	}
	return count
}

func inspectMemoryFile(path, kind string) (MemoryFileSummary, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return MemoryFileSummary{}, false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return MemoryFileSummary{}, false
	}
	text := string(content)
	out := MemoryFileSummary{Path: path, Kind: kind, SizeBytes: info.Size(), ModifiedAt: info.ModTime()}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			out.NonEmptyLines++
		}
		if strings.HasPrefix(trimmed, "#") {
			out.HeadingCount++
		}
	}
	out.GonchoMentions = len(regexp.MustCompile(`(?i)\bgoncho\b`).FindAllStringIndex(text, -1))
	out.GormesMentions = len(regexp.MustCompile(`(?i)\bgormes\b`).FindAllStringIndex(text, -1))
	return out, true
}
