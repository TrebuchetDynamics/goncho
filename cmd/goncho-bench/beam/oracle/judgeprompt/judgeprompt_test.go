package judgeprompt

import (
	"strings"
	"testing"
)

func TestBuildAnswerRequestTrimsAndUsesNoMemoryPlaceholder(t *testing.T) {
	req := BuildAnswerRequest("  What happened?  ", "   ")
	if req.System != AnswerSystemPrompt {
		t.Fatalf("system prompt = %q", req.System)
	}
	if req.Context != "[No memories found]" {
		t.Fatalf("context = %q", req.Context)
	}
	if !strings.Contains(req.User, "QUESTION: What happened?") {
		t.Fatalf("user prompt did not trim question: %q", req.User)
	}
}

func TestBuildJudgePromptFormatsRubricAndCopiesInput(t *testing.T) {
	rubric := []string{" first ", "", "third"}
	prompt := BuildJudgePrompt("  Why?  ", "  Because  ", rubric)
	rubric[0] = "mutated"
	if prompt.System != JudgeSystemPrompt {
		t.Fatalf("system prompt = %q", prompt.System)
	}
	if prompt.Question != "Why?" {
		t.Fatalf("question = %q", prompt.Question)
	}
	if prompt.IdealAnswer != "Because" {
		t.Fatalf("ideal answer = %q", prompt.IdealAnswer)
	}
	if prompt.AnswerPlaceholder != AnswerPlaceholder {
		t.Fatalf("placeholder = %q", prompt.AnswerPlaceholder)
	}
	if prompt.Rubric[0] != " first " {
		t.Fatalf("rubric was not copied: %#v", prompt.Rubric)
	}
	if !strings.Contains(prompt.User, "1. first\n3. third") {
		t.Fatalf("rubric not formatted with original item numbering: %q", prompt.User)
	}
}
