package judgerequestcontract

import (
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/judgeprompt"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestBuildRowsProjectsCanonicalArtifactAndPromptFields(t *testing.T) {
	started := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	report := goncho.RecallBenchmarkReport{Cases: []goncho.RecallBenchmarkCaseReport{{
		ID:                    "q-1",
		Scale:                 "official",
		ConversationID:        "conv-1",
		Ability:               "  reasoning ",
		Question:              " Where is the map? ",
		IdealAnswer:           "In the desk.",
		Rubric:                []string{"mentions desk"},
		SelectedContext:       " selected context ",
		CandidateMemoryIDs:    []string{"mem-1", "mem-2"},
		SelectedMemoryIDs:     []string{"mem-1"},
		SelectedEvidenceKinds: []string{"fact"},
		TopEvidenceKinds:      []string{"episodic"},
		RubricContextScore:    0.75,
		RubricContextMatches:  []string{"desk"},
	}}}

	rows := BuildRows(report, "cfg-1", started)
	if len(rows) != 1 {
		t.Fatalf("BuildRows() len = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.ConfigID != "cfg-1" || row.RunStartedAt != "2025-01-02T03:04:05Z" {
		t.Fatalf("BuildRows() metadata = (%q, %q), want cfg-1 timestamp", row.ConfigID, row.RunStartedAt)
	}
	if row.Scale != "official" || row.ConversationID != "conv-1" || row.QID != "q-1" || row.Ability != "REASONING" || row.Question != "Where is the map?" {
		t.Fatalf("BuildRows() canonical fields = %#v", row)
	}
	if !row.PureRecall {
		t.Fatal("BuildRows() PureRecall = false, want true")
	}
	if row.AnswerRequest.System != judgeprompt.AnswerSystemPrompt || row.AnswerRequest.Context != "selected context" {
		t.Fatalf("BuildRows() answer request = %#v", row.AnswerRequest)
	}
	if row.JudgeRequest.System != judgeprompt.JudgeSystemPrompt || row.JudgeRequest.Question != "Where is the map?" {
		t.Fatalf("BuildRows() judge request = %#v", row.JudgeRequest)
	}
	if row.RecallProvenance.TopResultTier != "episodic" || row.RecallProvenance.VoiceSums["fact"] != 1 {
		t.Fatalf("BuildRows() recall provenance = %#v", row.RecallProvenance)
	}
	if row.CandidateMemoryIDs[0] != "mem-1" || row.SelectedMemoryIDs[0] != "mem-1" || row.RubricContextMatches[0] != "desk" {
		t.Fatalf("BuildRows() slices = %#v", row)
	}
}
