package goncho

import "testing"

func TestGraphDiagnosticsReportParserRejectsAndOrphans(t *testing.T) {
	report := BuildGraphHealthDiagnostics(GraphExpansionIndex{
		Memories: map[string]RecallCandidate{"mem-source": {MemoryID: "mem-source"}},
		Relations: []GraphRelation{
			{FromMemoryID: "mem-source", ToMemoryID: "mem-missing", Relation: "depends_on"},
			{FromMemoryID: " ", ToMemoryID: "mem-target", Relation: ""},
		},
	})
	if report.Status != "degraded" || report.RelationCount != 2 || report.OrphanRelationCount != 1 || report.ParserRejectCount != 1 {
		t.Fatalf("graph diagnostics = %+v", report)
	}
}

func TestHookDiagnosticsNormalizeProjectBasename(t *testing.T) {
	report := DiagnoseHookReplay(HostHookEvent{Event: HostHookPrompt, Metadata: map[string]string{"project": "/home/user/private/repo", "safe": "value"}})
	if report.Status != "ok" || report.Project != "repo" || !report.AbsolutePathRedacted || report.AbsolutePathLeakCount != 0 {
		t.Fatalf("hook diagnostics = %+v", report)
	}
}
