package locomo

const (
	FailureBucketWrongBranchRetrieval = "wrong_branch_retrieval"
	FailureBucketMissingCompanion     = "missing_companion_memory"
	FailureBucketMissingCandidate     = "missing_candidate"
	FailureBucketRankTooLowCandidate  = "rank_too_low_candidate"
	FailureBucketUnclassified         = "unclassified_failure"
)

type QuestionResult struct {
	ConversationID string
	GoldMemoryIDs  []string
	RetrievedIDs   []string
	Rank           int
	Category       string
}

func ClassifyFailureBucket(q QuestionResult, memoryConversationIDs map[string]string) string {
	if HasOutOfConversationTopHit(q, memoryConversationIDs) {
		return FailureBucketWrongBranchRetrieval
	}
	if HasMissingCompanion(q) {
		return FailureBucketMissingCompanion
	}
	if q.Rank == 0 {
		return FailureBucketMissingCandidate
	}
	if q.Rank > 1 {
		return FailureBucketRankTooLowCandidate
	}
	if q.Category != "" {
		return q.Category
	}
	return FailureBucketUnclassified
}

func HasOutOfConversationTopHit(q QuestionResult, memoryConversationIDs map[string]string) bool {
	if q.ConversationID == "" {
		return false
	}
	limit := len(q.RetrievedIDs)
	if limit > 10 {
		limit = 10
	}
	for _, id := range q.RetrievedIDs[:limit] {
		conversationID := memoryConversationIDs[id]
		if conversationID != "" && conversationID != q.ConversationID {
			return true
		}
	}
	return false
}

func HasMissingCompanion(q QuestionResult) bool {
	if len(q.GoldMemoryIDs) < 2 {
		return false
	}
	limit := len(q.RetrievedIDs)
	if limit > 10 {
		limit = 10
	}
	retrieved := make(map[string]struct{}, limit)
	for _, id := range q.RetrievedIDs[:limit] {
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
