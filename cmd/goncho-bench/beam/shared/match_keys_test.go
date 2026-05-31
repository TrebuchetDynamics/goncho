package shared

import "testing"

func TestNewOutcomeKeyNormalizesIdentityFields(t *testing.T) {
	key := NewOutcomeKey(" small ", " conv-1 ", " q-1 ")
	want := OutcomeKey{Scale: "small", ConversationID: "conv-1", QID: "q-1"}
	if key != want {
		t.Fatalf("NewOutcomeKey() = %#v, want %#v", key, want)
	}
}

func TestNewQuestionKeyNormalizesFallbackIdentityFields(t *testing.T) {
	key := NewQuestionKey(" small ", " conv-1 ", " rw ", "  Who   Knows?  ")
	want := QuestionKey{Scale: "small", ConversationID: "conv-1", Ability: "RW", Question: "who knows?"}
	if key != want {
		t.Fatalf("NewQuestionKey() = %#v, want %#v", key, want)
	}
}

func TestOutcomeKeyLessSortsByScaleConversationThenQID(t *testing.T) {
	if !NewOutcomeKey("a", "b", "c").Less(NewOutcomeKey("a", "b", "d")) {
		t.Fatal("expected qid to break ties")
	}
	if NewOutcomeKey("a", "b", "d").Less(NewOutcomeKey("a", "b", "c")) {
		t.Fatal("expected later qid not to sort before earlier qid")
	}
}
