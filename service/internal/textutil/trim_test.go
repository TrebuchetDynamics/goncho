package textutil

import "testing"

func TestTrimSentenceBoundary(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "trims_terminal_sentence_punctuation", value: "...hello world!?", want: "hello world"},
		{name: "trims_surrounding_space_after_punctuation", value: "!?  hello world  .", want: "hello world"},
		{name: "preserves_punctuation_inside_outer_space", value: " who owns api.v2?! ", want: "who owns api.v2?!"},
		{name: "empty", value: "?!.", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimSentenceBoundary(tt.value); got != tt.want {
				t.Fatalf("TrimSentenceBoundary(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestTrimQuestionPunctuation(t *testing.T) {
	if got := TrimQuestionPunctuation(" who owns api?! "); got != "who owns api?!" {
		t.Fatalf("TrimQuestionPunctuation preserved spaced punctuation = %q", got)
	}
	if got := TrimQuestionPunctuation("?!who owns api."); got != "who owns api" {
		t.Fatalf("TrimQuestionPunctuation trimmed punctuation = %q", got)
	}
}

func TestTrimQuestionPhraseBoundary(t *testing.T) {
	if got := TrimQuestionPhraseBoundary(" ?! who owns api . "); got != "who owns api" {
		t.Fatalf("TrimQuestionPhraseBoundary = %q", got)
	}
}
