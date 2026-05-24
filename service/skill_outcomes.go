package goncho

import (
	"context"
	"fmt"
)

// SkillOutcome records the result of a skill execution.
type SkillOutcome struct {
	SkillName string
	Success   bool
	Lesson    string
}

// RecordSkillOutcome writes a skill execution result as a Goncho conclusion.
// The source prefix "skill:<name>" is embedded in the conclusion text for later querying.
func (s *Service) RecordSkillOutcome(ctx context.Context, outcome SkillOutcome) error {
	status := "success"
	if !outcome.Success {
		status = "failure"
	}

	conclusion := fmt.Sprintf("[%s] skill:%s: %s", status, outcome.SkillName, outcome.Lesson)
	_, err := s.Conclude(ctx, ConcludeParams{
		Peer:       s.observer,
		Conclusion: conclusion,
		SessionKey: "",
	})
	return err
}

// SearchSkillOutcomes returns conclusions from skill executions matching the given skill name.
func (s *Service) SearchSkillOutcomes(ctx context.Context, skillName string, limit int) ([]string, error) {
	source := "skill:" + skillName
	result, err := s.Search(ctx, SearchParams{
		Peer:  s.observer,
		Query: source,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	var outcomes []string
	for _, hit := range result.Results {
		outcomes = append(outcomes, hit.Content)
	}
	return outcomes, nil
}
