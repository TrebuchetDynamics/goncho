package judgmentcontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestLoadJSONLFindsByOutcomeAndReportsDiagnostics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "judgments.jsonl")
	contents := strings.Join([]string{
		`{"scale":"official","conversation_id":"conv-1","qid":"q1","ability":"ret","question":"Where is the key?","score":0.75}`,
		`{"scale":"official","conversation_id":"conv-1","qid":"extra","score":0.25}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	set, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	row, ok := set.Find(goncho.RecallBenchmarkCaseReport{ID: " q1 ", Scale: "official", ConversationID: "conv-1"})
	if !ok {
		t.Fatal("Find() did not match outcome key")
	}
	if row.Score != 0.75 || row.Ability != "RET" {
		t.Fatalf("Find() row = %+v", row)
	}

	diag := set.Diagnostics(goncho.RecallBenchmarkReport{Cases: []goncho.RecallBenchmarkCaseReport{
		{ID: "q1", Scale: "official", ConversationID: "conv-1"},
		{ID: "missing", Scale: "official", ConversationID: "conv-1"},
	}})
	if diag.AppliedCount != 1 || diag.MissingCount != 1 || diag.UnmatchedCount != 1 {
		t.Fatalf("Diagnostics() = %+v", diag)
	}
}

func TestLoadNestedFindsByQuestionFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested.json")
	contents := `{"results":[{"scale":"official","conversation_id":"conv-2","results":[{"qid":"q2","ability":"RET","question":"Where is the map?","score":1}]}]}`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	set, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	row, ok := set.Find(goncho.RecallBenchmarkCaseReport{Scale: "official", ConversationID: "conv-2", Ability: "ret", Question: " Where is the map? "})
	if !ok {
		t.Fatal("Find() did not match normalized question fallback")
	}
	if row.QID != "q2" || row.Score != 1 {
		t.Fatalf("Find() row = %+v", row)
	}
}

func TestRequireCompleteRejectsMissingOrUnmatchedRows(t *testing.T) {
	set := Set{
		Source:       "test",
		Rows:         map[shared.OutcomeKey]Judgment{shared.NewOutcomeKey("official", "conv-1", "extra"): {Scale: "official", ConversationID: "conv-1", QID: "extra"}},
		QuestionRows: map[shared.QuestionKey]Judgment{},
		RowCount:     1,
	}
	err := RequireComplete(set, goncho.RecallBenchmarkReport{Cases: []goncho.RecallBenchmarkCaseReport{{ID: "missing", Scale: "official", ConversationID: "conv-1"}}})
	if err == nil {
		t.Fatal("RequireComplete() error = nil")
	}
	if got := err.Error(); !strings.Contains(got, "missing=1") || !strings.Contains(got, "unmatched=1") {
		t.Fatalf("RequireComplete() error = %q", got)
	}
}
