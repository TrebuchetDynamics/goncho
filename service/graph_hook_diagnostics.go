package goncho

import (
	"path/filepath"
	"strings"
)

type GraphHealthDiagnostics struct {
	Status              string `json:"status"`
	RelationCount       int    `json:"relation_count"`
	OrphanRelationCount int    `json:"orphan_relation_count"`
	ParserRejectCount   int    `json:"parser_reject_count"`
	Freshness           string `json:"freshness"`
}

type HookReplayDiagnostics struct {
	Status                string `json:"status"`
	Event                 string `json:"event"`
	Project               string `json:"project,omitempty"`
	AbsolutePathRedacted  bool   `json:"absolute_path_redacted"`
	AbsolutePathLeakCount int    `json:"absolute_path_leak_count"`
}

func BuildGraphHealthDiagnostics(index GraphExpansionIndex) GraphHealthDiagnostics {
	report := GraphHealthDiagnostics{Status: "ok", RelationCount: len(index.Relations), Freshness: "not_persisted"}
	for _, relation := range index.Relations {
		endpoints, ok := graphRelationStableEndpoints(relation)
		if !ok || strings.TrimSpace(relation.Relation) == "" {
			report.ParserRejectCount++
			report.Status = "degraded"
			continue
		}
		if _, ok := index.Memories[endpoints.FromMemoryID]; !ok {
			report.OrphanRelationCount++
			report.Status = "degraded"
		}
		if _, ok := index.Memories[endpoints.ToMemoryID]; !ok {
			report.OrphanRelationCount++
			report.Status = "degraded"
		}
	}
	return report
}

func DiagnoseHookReplay(event HostHookEvent) HookReplayDiagnostics {
	project, redacted := normalizedHookProject(event)
	leaks := hookAbsolutePathLeakCount(event, project)
	status := "ok"
	if leaks > 0 {
		status = "degraded"
	}
	return HookReplayDiagnostics{Status: status, Event: string(event.Event), Project: project, AbsolutePathRedacted: redacted, AbsolutePathLeakCount: leaks}
}

func normalizedHookProject(event HostHookEvent) (string, bool) {
	for _, key := range []string{"project", "cwd", "working_dir", "workspace"} {
		value := strings.TrimSpace(event.Metadata[key])
		if value == "" {
			continue
		}
		base := filepath.Base(value)
		if base == "." || base == string(filepath.Separator) || base == "" {
			return value, false
		}
		return base, base != value
	}
	return "", false
}

func hookAbsolutePathLeakCount(event HostHookEvent, normalizedProject string) int {
	count := 0
	for key, value := range event.Metadata {
		if key == "project" || key == "cwd" || key == "working_dir" || key == "workspace" {
			continue
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || trimmed == normalizedProject {
			continue
		}
		if filepath.IsAbs(trimmed) {
			count++
		}
	}
	return count
}
