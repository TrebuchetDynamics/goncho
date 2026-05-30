package main

import benchlocomo "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/locomo"

func classifyLocomoFailureBucket(q locomoQuestionResult, memoryConversationIDs map[string]string) string {
	return benchlocomo.ClassifyFailureBucket(toBenchLocomoQuestionResult(q), memoryConversationIDs)
}

func locomoFailureHasOutOfConversationTopHit(q locomoQuestionResult, memoryConversationIDs map[string]string) bool {
	return benchlocomo.HasOutOfConversationTopHit(toBenchLocomoQuestionResult(q), memoryConversationIDs)
}

func locomoFailureHasMissingCompanion(q locomoQuestionResult) bool {
	return benchlocomo.HasMissingCompanion(toBenchLocomoQuestionResult(q))
}

func toBenchLocomoQuestionResult(q locomoQuestionResult) benchlocomo.QuestionResult {
	return benchlocomo.QuestionResult{
		ConversationID: q.ConversationID,
		GoldMemoryIDs:  append([]string(nil), q.GoldMemoryIDs...),
		RetrievedIDs:   append([]string(nil), q.RetrievedIDs...),
		Rank:           q.Rank,
		Category:       q.Category,
	}
}
