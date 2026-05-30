package oracle

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service"
)

type beamServiceLeakageChecks struct {
	QuestionTextInMemory     int      `json:"question_text_in_memory"`
	RelevantIDInMemory       int      `json:"relevant_id_in_memory"`
	IdealAnswerTextInMemory  int      `json:"ideal_answer_text_in_memory"`
	RubricTextInMemory       int      `json:"rubric_text_in_memory"`
	BlockingLeakageCount     int      `json:"blocking_leakage_count"`
	ReportedAnswerLabelCount int      `json:"reported_answer_label_count"`
	Examples                 []string `json:"examples,omitempty"`
}

func checkBeamServiceLeakage(cases []goncho.RecallBenchmarkServiceCase) beamServiceLeakageChecks {
	checks := beamServiceLeakageChecks{Examples: []string{}}
	for _, c := range cases {
		for _, memory := range c.Memories {
			ref := strings.TrimSpace(memory.Ref)
			if ref == "" {
				ref = "memory"
			}
			content := memory.Conclusion
			if beamServiceLeakageContainsText(content, c.Query) {
				checks.QuestionTextInMemory++
				checks.addExample("question_text_in_memory", c.ID, ref)
			}
			for _, relevantID := range c.RelevantRefs {
				if beamServiceLeakageContainsID(content, relevantID) {
					checks.RelevantIDInMemory++
					checks.addExample("relevant_id_in_memory", c.ID, ref)
				}
			}
			if beamServiceLeakageContainsText(content, c.IdealAnswer) {
				checks.IdealAnswerTextInMemory++
				checks.addExample("ideal_answer_text_in_memory", c.ID, ref)
			}
			for _, rubric := range c.Rubric {
				if beamServiceLeakageContainsText(content, rubric) {
					checks.RubricTextInMemory++
					checks.addExample("rubric_text_in_memory", c.ID, ref)
				}
			}
		}
	}
	checks.BlockingLeakageCount = checks.QuestionTextInMemory + checks.RelevantIDInMemory + checks.RubricTextInMemory
	checks.ReportedAnswerLabelCount = checks.IdealAnswerTextInMemory + checks.RubricTextInMemory
	if len(checks.Examples) == 0 {
		checks.Examples = nil
	}
	return checks
}

func beamServiceLeakageContainsText(content, needle string) bool {
	needle = strings.TrimSpace(needle)
	if len([]rune(needle)) < 8 {
		return false
	}
	return strings.Contains(strings.ToLower(content), strings.ToLower(needle))
}

func beamServiceLeakageContainsID(content, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	return strings.Contains(strings.ToLower(content), strings.ToLower(id))
}

func (c *beamServiceLeakageChecks) addExample(kind, questionID, memoryRef string) {
	if len(c.Examples) >= 10 {
		return
	}
	c.Examples = append(c.Examples, fmt.Sprintf("%s:%s:%s", strings.TrimSpace(questionID), kind, strings.TrimSpace(memoryRef)))
}

func beamServiceHasBlockingLeakage(checks beamServiceLeakageChecks) bool {
	return checks.BlockingLeakageCount > 0
}
