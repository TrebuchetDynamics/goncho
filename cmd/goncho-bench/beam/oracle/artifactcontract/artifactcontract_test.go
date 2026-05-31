package artifactcontract

import (
	"reflect"
	"testing"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestBuildCaseFieldsCanonicalizesAndCopiesArtifactRowInputs(t *testing.T) {
	candidateIDs := []string{"mem-1", "mem-2"}
	selectedIDs := []string{"mem-1"}
	matches := []string{"match-1"}
	got := BuildCaseFields(goncho.RecallBenchmarkCaseReport{
		Scale:                "",
		ConversationID:       " conv-1 ",
		ID:                   "q-1",
		Ability:              " Graph ",
		Question:             "  What happened?  ",
		CandidateMemoryIDs:   candidateIDs,
		SelectedMemoryIDs:    selectedIDs,
		RubricContextScore:   0.75,
		RubricContextMatches: matches,
	})

	candidateIDs[0] = "mutated"
	selectedIDs[0] = "mutated"
	matches[0] = "mutated"

	if got.Scale != "100K" {
		t.Fatalf("Scale = %q", got.Scale)
	}
	if got.ConversationID != "conv-1" {
		t.Fatalf("ConversationID = %q", got.ConversationID)
	}
	if got.QID != "q-1" {
		t.Fatalf("QID = %q", got.QID)
	}
	if got.Ability != "GRAPH" {
		t.Fatalf("Ability = %q", got.Ability)
	}
	if got.Question != "What happened?" {
		t.Fatalf("Question = %q", got.Question)
	}
	if !reflect.DeepEqual(got.CandidateMemoryIDs, []string{"mem-1", "mem-2"}) {
		t.Fatalf("CandidateMemoryIDs = %#v", got.CandidateMemoryIDs)
	}
	if !reflect.DeepEqual(got.SelectedMemoryIDs, []string{"mem-1"}) {
		t.Fatalf("SelectedMemoryIDs = %#v", got.SelectedMemoryIDs)
	}
	if got.RubricContextScore != 0.75 {
		t.Fatalf("RubricContextScore = %v", got.RubricContextScore)
	}
	if !reflect.DeepEqual(got.RubricContextMatches, []string{"match-1"}) {
		t.Fatalf("RubricContextMatches = %#v", got.RubricContextMatches)
	}
}

func TestBuildRecallProvenanceCopiesIDsAndClassifiesVoices(t *testing.T) {
	candidateIDs := []string{"mem-1", "mem-2"}
	selectedIDs := []string{"mem-1"}
	got := BuildRecallProvenance(goncho.RecallBenchmarkCaseReport{
		CandidateMemoryIDs:    candidateIDs,
		SelectedMemoryIDs:     selectedIDs,
		SelectedEvidenceKinds: []string{"Fact", "graph", ""},
		TopEvidenceKinds:      []string{"note", "fact"},
	})

	candidateIDs[0] = "mutated"
	selectedIDs[0] = "mutated"

	if got.Engine != "goncho-service-recall" {
		t.Fatalf("Engine = %q", got.Engine)
	}
	if got.KeptCount != 2 {
		t.Fatalf("KeptCount = %d", got.KeptCount)
	}
	if !reflect.DeepEqual(got.CandidateMemoryIDs, []string{"mem-1", "mem-2"}) {
		t.Fatalf("CandidateMemoryIDs = %#v", got.CandidateMemoryIDs)
	}
	if !reflect.DeepEqual(got.SelectedMemoryIDs, []string{"mem-1"}) {
		t.Fatalf("SelectedMemoryIDs = %#v", got.SelectedMemoryIDs)
	}
	if !reflect.DeepEqual(got.VoiceSums, map[string]float64{"fact": 1, "graph": 1}) {
		t.Fatalf("VoiceSums = %#v", got.VoiceSums)
	}
	if !reflect.DeepEqual(got.TopResultVoices, map[string]float64{"note": 1, "fact": 1}) {
		t.Fatalf("TopResultVoices = %#v", got.TopResultVoices)
	}
	if got.TopResultTier != "structured" {
		t.Fatalf("TopResultTier = %q", got.TopResultTier)
	}
}
