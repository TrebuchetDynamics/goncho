package convertpolicy

import (
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/convertcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/jsonlcontract"
)

func TestStableIDSegmentNormalizesConversationID(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{in: "  Conv 42/A  ", want: "conv-42-a"},
		{in: "!!!", want: "conversation"},
		{in: "Already--Split", want: "already-split"},
	} {
		if got := StableIDSegment(tc.in); got != tc.want {
			t.Fatalf("StableIDSegment(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPythonLiteralToJSONishConvertsPythonBarewordsAndQuotes(t *testing.T) {
	got := PythonLiteralToJSONish(`{'ok': True, "missing": None, 'bad': False, 'quote': 'a"b'}`)
	want := `{"ok": true, "missing": null, "bad": false, "quote": "a\"b"}`
	if got != want {
		t.Fatalf("PythonLiteralToJSONish() = %q, want %q", got, want)
	}
}

func TestParseQuestionsAcceptsEncodedPythonLiteralAndNormalizesAbilities(t *testing.T) {
	encoded, err := json.Marshal(`{'fact': [{'qid': 'q1', 'question': 'What?', 'relevant_message_indices': [0], 'expected_no_answer': False}], ' ': [{'qid': 'ignored'}]}`)
	if err != nil {
		t.Fatalf("marshal encoded questions: %v", err)
	}

	got, err := ParseQuestions(encoded)
	if err != nil {
		t.Fatalf("ParseQuestions error = %v", err)
	}
	if len(got) != 1 || len(got["FACT"]) != 1 {
		t.Fatalf("questions = %#v", got)
	}
	question := got["FACT"][0]
	if question.QID != "q1" || question.Question != "What?" || question.ExpectedNoAnswer || len(question.RelevantMessageIdxs) != 1 || question.RelevantMessageIdxs[0] != 0 {
		t.Fatalf("question = %#v", question)
	}
}

func TestRelevantIDsPrefersExplicitIDsAndDeduplicatesIndexedRefs(t *testing.T) {
	explicit, err := RelevantIDs(convertcontract.Question{RelevantIDs: []string{"m-explicit"}, RelevantMessageIdxs: []int{1}}, []string{"m0", "m1"})
	if err != nil {
		t.Fatalf("RelevantIDs explicit error = %v", err)
	}
	if len(explicit) != 1 || explicit[0] != "m-explicit" {
		t.Fatalf("explicit = %#v", explicit)
	}

	indexed, err := RelevantIDs(convertcontract.Question{EvidenceMessageIdxs: []int{1, 0, 1}}, []string{"m0", "m1"})
	if err != nil {
		t.Fatalf("RelevantIDs indexed error = %v", err)
	}
	if len(indexed) != 2 || indexed[0] != "m1" || indexed[1] != "m0" {
		t.Fatalf("indexed = %#v", indexed)
	}
}

func TestRelevantIDsRejectsOutOfRangeIndex(t *testing.T) {
	_, err := RelevantIDs(convertcontract.Question{SourceMessageIdxs: []int{2}}, []string{"m0"})
	if err == nil {
		t.Fatal("RelevantIDs expected range error")
	}
}

func TestSummarizeRecordsCountsQuestionsAndUnscorableWarnings(t *testing.T) {
	got := SummarizeRecords([]jsonlcontract.Record{
		{Type: "meta", Scale: "1M"},
		{Type: "memory", ConversationID: "conv-1", ID: "mem-1"},
		{Type: "question", ConversationID: "conv-1", ID: "q1", Ability: "fact", RelevantIDs: []string{"mem-1"}},
		{Type: "question", ConversationID: "", ID: "q2", Ability: "abs", ExpectedNoAnswer: true},
		{Type: "question", ConversationID: "conv-2", ID: "q3", Ability: "", Query: "missing refs"},
	})

	if got.Source != "huggingface-beam-jsonl" {
		t.Fatalf("Source = %q", got.Source)
	}
	if got.ConversationCount != 3 || got.MemoryCount != 1 || got.QuestionCount != 3 {
		t.Fatalf("counts = conversations:%d memories:%d questions:%d", got.ConversationCount, got.MemoryCount, got.QuestionCount)
	}
	if got.ExpectedNoAnswerQuestionCount != 1 || got.UnscorableQuestionCount != 1 {
		t.Fatalf("question classifications = expected-no-answer:%d unscorable:%d", got.ExpectedNoAnswerQuestionCount, got.UnscorableQuestionCount)
	}
	if got.QuestionsByAbility["FACT"] != 1 || got.QuestionsByAbility["ABS"] != 1 || got.QuestionsByAbility["UNKNOWN"] != 1 {
		t.Fatalf("QuestionsByAbility = %#v", got.QuestionsByAbility)
	}
	if len(got.Warnings) != 1 || got.Warnings[0].Code != "beam_question_missing_relevant_ids" || got.Warnings[0].QID != "q3" {
		t.Fatalf("Warnings = %#v", got.Warnings)
	}
}
