package goncho

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type SessionEvidenceScope string

const (
	SessionEvidenceScopeRoot              SessionEvidenceScope = "root"
	SessionEvidenceScopeProfile           SessionEvidenceScope = "profile"
	SessionEvidenceScopeWorkspaceTemplate SessionEvidenceScope = "workspace_template"
)

type SessionEvidenceInput struct {
	WorkspaceID    string                      `json:"workspace_id"`
	SessionIndexes []SessionEvidenceIndex      `json:"session_indexes,omitempty"`
	MemoryFiles    []SessionEvidenceMemoryFile `json:"memory_files,omitempty"`
	Messages       []MessageRecord             `json:"messages,omitempty"`
	Observations   []Observation               `json:"observations,omitempty"`
	Summaries      []SessionSummary            `json:"summaries,omitempty"`
}

type SessionEvidenceIndex struct {
	Scope        SessionEvidenceScope `json:"scope"`
	ProfileID    string               `json:"profile_id,omitempty"`
	Path         string               `json:"path"`
	SessionCount int                  `json:"session_count"`
	LineageCount int                  `json:"lineage_count"`
	UpdatedAt    time.Time            `json:"updated_at,omitempty"`
}

type SessionEvidenceMemoryFile struct {
	Scope          SessionEvidenceScope `json:"scope"`
	ProfileID      string               `json:"profile_id,omitempty"`
	Path           string               `json:"path"`
	Kind           string               `json:"kind"`
	SizeBytes      int64                `json:"size_bytes"`
	ModifiedAt     time.Time            `json:"modified_at,omitempty"`
	NonEmptyLines  int                  `json:"non_empty_lines"`
	HeadingCount   int                  `json:"heading_count"`
	GonchoMentions int                  `json:"goncho_mentions"`
	GormesMentions int                  `json:"gormes_mentions"`
	RawContent     string               `json:"-"`
}

type SessionEvidenceProjection struct {
	WorkspaceID         string                          `json:"workspace_id"`
	Root                SessionEvidenceScopeSummary     `json:"root"`
	WorkspaceTemplate   SessionEvidenceScopeSummary     `json:"workspace_template"`
	Profiles            []SessionEvidenceProfileSummary `json:"profiles"`
	TotalMemoryFiles    int                             `json:"total_memory_files"`
	TotalGonchoMentions int                             `json:"total_goncho_mentions"`
	TotalGormesMentions int                             `json:"total_gormes_mentions"`
	TimelineEventCount  int                             `json:"timeline_event_count"`
}

type SessionEvidenceScopeSummary struct {
	SessionCount    int       `json:"session_count"`
	LineageCount    int       `json:"lineage_count"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
	MemoryFileCount int       `json:"memory_file_count"`
	SizeBytes       int64     `json:"size_bytes"`
	NonEmptyLines   int       `json:"non_empty_lines"`
	HeadingCount    int       `json:"heading_count"`
	GonchoMentions  int       `json:"goncho_mentions"`
	GormesMentions  int       `json:"gormes_mentions"`
}

type SessionEvidenceProfileSummary struct {
	ProfileID string `json:"profile_id"`
	SessionEvidenceScopeSummary
}

func ProjectSessionEvidence(input SessionEvidenceInput) SessionEvidenceProjection {
	projection := SessionEvidenceProjection{WorkspaceID: strings.TrimSpace(input.WorkspaceID)}
	profiles := map[string]*SessionEvidenceProfileSummary{}
	profileFor := func(profileID string) *SessionEvidenceProfileSummary {
		profileID = strings.TrimSpace(profileID)
		if profileID == "" {
			profileID = "default"
		}
		if existing := profiles[profileID]; existing != nil {
			return existing
		}
		created := &SessionEvidenceProfileSummary{ProfileID: profileID}
		profiles[profileID] = created
		return created
	}
	for _, index := range input.SessionIndexes {
		summary := summaryForScope(&projection, profiles, profileFor, index.Scope, index.ProfileID)
		summary.SessionCount += index.SessionCount
		summary.LineageCount += index.LineageCount
		if index.UpdatedAt.After(summary.UpdatedAt) {
			summary.UpdatedAt = index.UpdatedAt
		}
	}
	for _, file := range input.MemoryFiles {
		summary := summaryForScope(&projection, profiles, profileFor, file.Scope, file.ProfileID)
		summary.MemoryFileCount++
		summary.SizeBytes += file.SizeBytes
		summary.NonEmptyLines += file.NonEmptyLines
		summary.HeadingCount += file.HeadingCount
		summary.GonchoMentions += file.GonchoMentions
		summary.GormesMentions += file.GormesMentions
		projection.TotalMemoryFiles++
		projection.TotalGonchoMentions += file.GonchoMentions
		projection.TotalGormesMentions += file.GormesMentions
	}
	projection.TimelineEventCount = len(input.Messages) + len(input.Observations) + len(input.Summaries)
	for _, profile := range profiles {
		projection.Profiles = append(projection.Profiles, *profile)
	}
	sort.Slice(projection.Profiles, func(i, j int) bool { return projection.Profiles[i].ProfileID < projection.Profiles[j].ProfileID })
	return projection
}

func summaryForScope(projection *SessionEvidenceProjection, profiles map[string]*SessionEvidenceProfileSummary, profileFor func(string) *SessionEvidenceProfileSummary, scope SessionEvidenceScope, profileID string) *SessionEvidenceScopeSummary {
	switch scope {
	case SessionEvidenceScopeProfile:
		return &profileFor(profileID).SessionEvidenceScopeSummary
	case SessionEvidenceScopeWorkspaceTemplate:
		return &projection.WorkspaceTemplate
	default:
		return &projection.Root
	}
}

func (p SessionEvidenceProjection) Profile(profileID string) (SessionEvidenceProfileSummary, bool) {
	profileID = strings.TrimSpace(profileID)
	for _, profile := range p.Profiles {
		if profile.ProfileID == profileID {
			return profile, true
		}
	}
	return SessionEvidenceProfileSummary{}, false
}

func (p SessionEvidenceProjection) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "workspace=%s root_sessions=%d root_lineage=%d profiles=%d memory_files=%d goncho_mentions=%d timeline_events=%d", p.WorkspaceID, p.Root.SessionCount, p.Root.LineageCount, len(p.Profiles), p.TotalMemoryFiles, p.TotalGonchoMentions, p.TimelineEventCount)
	for _, profile := range p.Profiles {
		fmt.Fprintf(&b, " profile[%s]=sessions:%d,lineage:%d,memory:%d,goncho:%d", profile.ProfileID, profile.SessionCount, profile.LineageCount, profile.MemoryFileCount, profile.GonchoMentions)
	}
	return b.String()
}
