package goncho

import (
	"strings"
	"testing"
	"time"
)

func TestSessionEvidenceProjectionDistinguishesScopesWithoutRawMemory(t *testing.T) {
	projection := ProjectSessionEvidence(SessionEvidenceInput{
		WorkspaceID: "gormes",
		SessionIndexes: []SessionEvidenceIndex{
			{Scope: SessionEvidenceScopeRoot, Path: "/tmp/.gormes/sessions/index.yaml", SessionCount: 1, LineageCount: 1, UpdatedAt: time.Unix(100, 0).UTC()},
			{Scope: SessionEvidenceScopeProfile, ProfileID: "mineru", Path: "/tmp/.gormes/profiles/mineru/sessions/index.yaml", SessionCount: 1, LineageCount: 2, UpdatedAt: time.Unix(90, 0).UTC()},
		},
		MemoryFiles: []SessionEvidenceMemoryFile{
			{Scope: SessionEvidenceScopeRoot, Path: "/tmp/.gormes/memory/MEMORY.md", Kind: "root_memory", SizeBytes: 120, NonEmptyLines: 2, GonchoMentions: 1, RawContent: "private root memory must not leak"},
			{Scope: SessionEvidenceScopeWorkspaceTemplate, Path: "/tmp/.gormes/workspace/memory/MEMORY.md", Kind: "workspace_template_memory", SizeBytes: 80, HeadingCount: 2, RawContent: "private template memory must not leak"},
			{Scope: SessionEvidenceScopeProfile, ProfileID: "mineru", Path: "/tmp/.gormes/profiles/mineru/GONCHO_MEMORY.md", Kind: "profile_goncho_memory", SizeBytes: 90, GonchoMentions: 1, RawContent: "private profile memory must not leak"},
		},
	})

	if projection.WorkspaceID != "gormes" || projection.Root.SessionCount != 1 || projection.Root.LineageCount != 1 {
		t.Fatalf("projection root = %+v", projection)
	}
	mineru, ok := projection.Profile("mineru")
	if !ok {
		t.Fatalf("profiles = %+v, missing mineru", projection.Profiles)
	}
	if mineru.SessionCount != 1 || mineru.LineageCount != 2 || mineru.MemoryFileCount != 1 || mineru.GonchoMentions != 1 {
		t.Fatalf("mineru evidence = %+v", mineru)
	}
	if projection.WorkspaceTemplate.MemoryFileCount != 1 || projection.WorkspaceTemplate.HeadingCount != 2 {
		t.Fatalf("workspace template evidence = %+v", projection.WorkspaceTemplate)
	}
	if projection.TotalMemoryFiles != 3 || projection.TotalGonchoMentions != 2 {
		t.Fatalf("totals = files:%d goncho:%d", projection.TotalMemoryFiles, projection.TotalGonchoMentions)
	}
	serialized := projection.String()
	for _, leaked := range []string{"private root memory", "private template memory", "private profile memory"} {
		if strings.Contains(serialized, leaked) {
			t.Fatalf("projection leaked raw memory %q in %s", leaked, serialized)
		}
	}
}
