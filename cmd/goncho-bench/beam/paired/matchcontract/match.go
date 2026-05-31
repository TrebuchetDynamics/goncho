package matchcontract

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
)

type Outcome = shared.PairedOutcome

type Key = shared.OutcomeKey

type QuestionKey = shared.QuestionKey

type MatchedOutcome struct {
	Baseline  Outcome
	Candidate Outcome
	MatchKey  string
}

func OutcomeKey(row Outcome) Key {
	return shared.NewOutcomeKey(row.Scale, row.ConversationID, row.QID)
}

func OutcomeQuestionKey(row Outcome) QuestionKey {
	key := shared.NewQuestionKey(row.Scale, row.ConversationID, row.Ability, row.Question)
	if key.Question == "" {
		return QuestionKey{}
	}
	return key
}

func MatchOutcomes(baselineRows, candidateRows []Outcome) ([]MatchedOutcome, int, error) {
	sort.Slice(baselineRows, func(i, j int) bool { return outcomeLess(baselineRows[i], baselineRows[j]) })
	sort.Slice(candidateRows, func(i, j int) bool { return outcomeLess(candidateRows[i], candidateRows[j]) })
	candidateByQID := map[Key]int{}
	candidateByQuestion := map[QuestionKey]int{}
	candidateQuestionCounts := map[QuestionKey]int{}
	for i, row := range candidateRows {
		if key := OutcomeKey(row); key.QID != "" {
			if _, ok := candidateByQID[key]; !ok {
				candidateByQID[key] = i
			}
		}
		if key := OutcomeQuestionKey(row); key.Question != "" {
			candidateQuestionCounts[key]++
			if _, ok := candidateByQuestion[key]; !ok {
				candidateByQuestion[key] = i
			}
		}
	}
	baselineQuestionCounts := map[QuestionKey]int{}
	for _, row := range baselineRows {
		if key := OutcomeQuestionKey(row); key.Question != "" {
			baselineQuestionCounts[key]++
		}
	}
	usedCandidates := map[int]struct{}{}
	matched := []MatchedOutcome{}
	dropped := 0
	for _, base := range baselineRows {
		if idx, ok := candidateByQID[OutcomeKey(base)]; ok {
			if _, used := usedCandidates[idx]; !used {
				usedCandidates[idx] = struct{}{}
				matched = append(matched, MatchedOutcome{Baseline: base, Candidate: candidateRows[idx], MatchKey: "qid"})
				continue
			}
		}
		if questionKey := OutcomeQuestionKey(base); questionKey.Question != "" {
			candidateCount := candidateQuestionCounts[questionKey]
			baselineCount := baselineQuestionCounts[questionKey]
			if candidateCount > 0 && (baselineCount > 1 || candidateCount > 1) {
				return nil, 0, fmt.Errorf("goncho-bench: ambiguous BEAM paired question-key fallback for scale=%q conversation_id=%q ability=%q question=%q (baseline_rows=%d candidate_rows=%d)", questionKey.Scale, questionKey.ConversationID, questionKey.Ability, questionKey.Question, baselineCount, candidateCount)
			}
			if idx, ok := candidateByQuestion[questionKey]; ok {
				if _, used := usedCandidates[idx]; !used {
					usedCandidates[idx] = struct{}{}
					matched = append(matched, MatchedOutcome{Baseline: base, Candidate: candidateRows[idx], MatchKey: "question"})
					continue
				}
			}
		}
		dropped++
	}
	dropped += len(candidateRows) - len(usedCandidates)
	return matched, dropped, nil
}

func outcomeLess(a, b Outcome) bool {
	ak, bk := OutcomeKey(a), OutcomeKey(b)
	if ak != bk {
		return ak.Less(bk)
	}
	aq, bq := OutcomeQuestionKey(a), OutcomeQuestionKey(b)
	if aq.Question != bq.Question {
		return aq.Question < bq.Question
	}
	return strings.TrimSpace(a.ConfigID) < strings.TrimSpace(b.ConfigID)
}
