package main

func classifyLocomoFailureBucket(q locomoQuestionResult, memoryConversationIDs map[string]string) string {
	if locomoFailureHasOutOfConversationTopHit(q, memoryConversationIDs) {
		return "wrong_branch_retrieval"
	}
	if locomoFailureHasMissingCompanion(q) {
		return "missing_companion_memory"
	}
	if q.Rank == 0 {
		return "missing_candidate"
	}
	if q.Rank > 1 {
		return "rank_too_low_candidate"
	}
	if q.Category != "" {
		return q.Category
	}
	return "unclassified_failure"
}

func locomoFailureHasOutOfConversationTopHit(q locomoQuestionResult, memoryConversationIDs map[string]string) bool {
	if q.ConversationID == "" {
		return false
	}
	for _, id := range q.RetrievedIDs[:min(10, len(q.RetrievedIDs))] {
		conversationID := memoryConversationIDs[id]
		if conversationID != "" && conversationID != q.ConversationID {
			return true
		}
	}
	return false
}

func locomoFailureHasMissingCompanion(q locomoQuestionResult) bool {
	if len(q.GoldMemoryIDs) < 2 {
		return false
	}
	retrieved := make(map[string]struct{}, min(10, len(q.RetrievedIDs)))
	for _, id := range q.RetrievedIDs[:min(10, len(q.RetrievedIDs))] {
		if id != "" {
			retrieved[id] = struct{}{}
		}
	}
	matched := 0
	for _, id := range q.GoldMemoryIDs {
		if _, ok := retrieved[id]; ok {
			matched++
		}
	}
	return matched > 0 && matched < len(q.GoldMemoryIDs)
}
