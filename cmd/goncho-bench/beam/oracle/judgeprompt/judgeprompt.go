package judgeprompt

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
)

const (
	AnswerSystemPrompt = "You answer BEAM memory benchmark questions using only the retrieved memory context. If the context is insufficient, say you do not know."
	JudgeSystemPrompt  = "You are an expert evaluator for a memory benchmark. Score the AI answer against each rubric item and return JSON with scores and overall_score."
	AnswerPlaceholder  = "[AI_ANSWER]"
)

type AnswerRequest struct {
	System  string `json:"system"`
	User    string `json:"user"`
	Context string `json:"context"`
}

type JudgePrompt struct {
	System            string   `json:"system"`
	User              string   `json:"user"`
	Question          string   `json:"question"`
	IdealAnswer       string   `json:"ideal_answer,omitempty"`
	Rubric            []string `json:"rubric,omitempty"`
	AnswerPlaceholder string   `json:"answer_placeholder"`
}

func BuildAnswerRequest(question, context string) AnswerRequest {
	if !shared.HasNonEmptyTrimmed(context) {
		context = "[No memories found]"
	}
	return AnswerRequest{
		System:  AnswerSystemPrompt,
		User:    fmt.Sprintf("RETRIEVED MEMORIES:\n%s\n\nQUESTION: %s\n\nANSWER:", strings.TrimSpace(context), strings.TrimSpace(question)),
		Context: strings.TrimSpace(context),
	}
}

func BuildJudgePrompt(question, idealAnswer string, rubric []string) JudgePrompt {
	rubricText := ""
	if len(rubric) > 0 {
		var b strings.Builder
		for i, item := range rubric {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			fmt.Fprintf(&b, "%d. %s", i+1, item)
		}
		rubricText = b.String()
	}
	user := fmt.Sprintf("QUESTION: %s\n\nRUBRIC ITEMS:\n%s\n\nAI's ANSWER: %s\n\nFor each rubric item, score how well the AI's answer matches. Return JSON with scores array and overall_score.", strings.TrimSpace(question), rubricText, AnswerPlaceholder)
	return JudgePrompt{
		System:            JudgeSystemPrompt,
		User:              user,
		Question:          strings.TrimSpace(question),
		IdealAnswer:       strings.TrimSpace(idealAnswer),
		Rubric:            append([]string(nil), rubric...),
		AnswerPlaceholder: AnswerPlaceholder,
	}
}
