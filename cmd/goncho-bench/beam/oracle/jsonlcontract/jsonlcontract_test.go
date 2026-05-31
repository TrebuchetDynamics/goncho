package jsonlcontract

import "testing"

func TestNormalizeConversationIDDefaultsBlank(t *testing.T) {
	if got := NormalizeConversationID("  "); got != "goncho-service-memoria-fixtures" {
		t.Fatalf("NormalizeConversationID blank = %q", got)
	}
	if got := NormalizeConversationID(" conv "); got != "conv" {
		t.Fatalf("NormalizeConversationID explicit = %q", got)
	}
}

func TestScoringConfigWeightsGraphRequiredEvidence(t *testing.T) {
	cfg := ScoringConfig(Question{Ability: "Graph", RequiredEvidenceKinds: []string{" graph "}})
	if cfg.Version != "beam-jsonl-graph-v1" {
		t.Fatalf("version = %q", cfg.Version)
	}
	if cfg.Weights["graph"] != 0.80 || cfg.Weights["fact"] != 0.10 {
		t.Fatalf("graph weights = %#v", cfg.Weights)
	}
}

func TestScoringConfigWeightsFactDefault(t *testing.T) {
	cfg := ScoringConfig(Question{Ability: "Fact"})
	if cfg.Weights["fact"] != 0.75 || cfg.Weights["graph"] != 0.05 {
		t.Fatalf("default weights = %#v", cfg.Weights)
	}
}

func TestServiceCasesFromRecordsProjectsMemoriesAndQuestions(t *testing.T) {
	records := []Record{
		{Type: "meta", Scale: "medium"},
		{Type: "memory", ID: " mem-1 ", ConversationID: " conv-1 ", Peer: " peer-a ", SessionKey: " sess-a ", Content: " remembered fact "},
		{Type: "question", ID: " q-1 ", ConversationID: " conv-1 ", Ability: " Fact ", Query: " what happened? ", IdealAnswer: " answer ", Rubric: []string{"specific"}, RelevantIDs: []string{"mem-1"}, ContextContains: []string{"fact"}, Limit: 3, MaxTokens: 99},
	}

	cases, err := ServiceCasesFromRecords(records, "fallback")
	if err != nil {
		t.Fatalf("ServiceCasesFromRecords error = %v", err)
	}
	if len(cases) != 1 {
		t.Fatalf("case count = %d", len(cases))
	}
	got := cases[0]
	if got.ID != "q-1" || got.Scale != "medium" || got.ConversationID != "conv-1" || got.Ability != "FACT" || got.Query != "what happened?" {
		t.Fatalf("case fields = %#v", got)
	}
	if len(got.Memories) != 1 || got.Memories[0].Ref != "mem-1" || got.Memories[0].Conclusion != "remembered fact" {
		t.Fatalf("memories = %#v", got.Memories)
	}
	if got.ScoringConfig.Version != "beam-jsonl-fact-v1" {
		t.Fatalf("scoring config = %#v", got.ScoringConfig)
	}
}

func TestServiceCasesFromRecordsAllowsExpectedNoAnswerWithoutMemories(t *testing.T) {
	cases, err := ServiceCasesFromRecords([]Record{{Type: "question", ID: "q-empty", Query: "anything?", ExpectedNoAnswer: true}}, "")
	if err != nil {
		t.Fatalf("ServiceCasesFromRecords expected no answer error = %v", err)
	}
	if len(cases) != 1 || !cases[0].ExpectedNoAnswer {
		t.Fatalf("cases = %#v", cases)
	}
}

func TestServiceCasesFromRecordsRejectsQuestionWithoutMemories(t *testing.T) {
	_, err := ServiceCasesFromRecords([]Record{{Type: "question", ID: "q-missing", Query: "anything?"}}, "")
	if err == nil {
		t.Fatal("ServiceCasesFromRecords expected missing-memory error")
	}
}
