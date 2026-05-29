package goncho

import "github.com/TrebuchetDynamics/goncho/service/internal/searchintent"

func searchFactIntentScore(query, content string) float64 {
	return searchintent.Score(query, content)
}

func searchHitFactIntentScore(query string, hit SearchHit) float64 {
	score := searchFactIntentScore(query, hit.Content)
	for _, fact := range hit.factAnnotations {
		if factScore := searchFactIntentScore(query, fact.Value); factScore > score {
			score = factScore
		}
	}
	return score
}

func searchFactIntentBonus(factScore, maxBaseScore float64) float64 {
	return searchintent.Bonus(factScore, maxBaseScore)
}

func searchTimelineAnswerParts(sentence string) (event, date string, ok bool) {
	return searchintent.TimelineAnswerParts(sentence)
}
func searchLocationAnswerParts(sentence string) (object, location string, ok bool) {
	return searchintent.LocationAnswerParts(sentence)
}
func searchPreferenceQuestion(query string) (subject, attribute string, ok bool) {
	return searchintent.PreferenceQuestion(query)
}
func searchPreferenceAnswerParts(sentence string) (subject, value, attribute string, ok bool) {
	return searchintent.PreferenceAnswerParts(sentence)
}
func searchRankTokenCoverage(want map[string]struct{}, value string) float64 {
	return searchintent.TokenCoverage(want, value)
}
func searchInstructionQuestion(query string) (subject, topic string, ok bool) {
	return searchintent.InstructionQuestion(query)
}
func searchInstructionAnswerParts(sentence string) (subject, instruction string, ok bool) {
	return searchintent.InstructionAnswerParts(sentence)
}
func searchSequenceAnswerParts(sentence string) (subject, steps string, ok bool) {
	return searchintent.SequenceAnswerParts(sentence)
}
func searchDecisionAnswerParts(sentence string) (decision string, ok bool) {
	return searchintent.DecisionAnswerParts(sentence)
}
func searchDecisionQuestionTopic(query string) (string, bool) {
	return searchintent.DecisionQuestionTopic(query)
}
func searchFactObjectLooksAssertive(object string) bool {
	return searchintent.FactObjectLooksAssertive(object)
}
func searchFactSubjectLooksAssertive(subject string) bool {
	return searchintent.FactSubjectLooksAssertive(subject)
}
func searchMetricAnswerParts(sentence string) (key, value string, ok bool) {
	return searchintent.MetricAnswerParts(sentence)
}
func searchNegationAnswerParts(sentence string) (object string, ok bool) {
	return searchintent.NegationAnswerParts(sentence)
}
func searchNegationQuestionObject(query string) (string, bool) {
	return searchintent.NegationQuestionObject(query)
}
func searchSequenceQuestionSubject(query string) (string, bool) {
	return searchintent.SequenceQuestionSubject(query)
}
func searchVersionAnswerParts(sentence string) (subject, version string, ok bool) {
	return searchintent.VersionAnswerParts(sentence)
}
func cleanFactObject(value string) string { return searchintent.CleanFactObject(value) }
func cleanFactValue(value string) string  { return searchintent.CleanFactValue(value) }
