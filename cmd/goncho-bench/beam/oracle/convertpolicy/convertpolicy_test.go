package convertpolicy

import (
	"testing"

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
