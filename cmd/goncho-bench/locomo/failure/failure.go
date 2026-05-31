package failure

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/locomo/rankwindow"

const (
	BucketWrongBranchRetrieval = "wrong_branch_retrieval"
	BucketMissingCompanion     = "missing_companion_memory"
	BucketMissingCandidate     = "missing_candidate"
	BucketRankTooLowCandidate  = "rank_too_low_candidate"
	BucketUnclassified         = "unclassified_failure"
)

type QuestionResult struct {
	ConversationID string
	GoldMemoryIDs  []string
	RetrievedIDs   []string
	Rank           int
	Category       string
}

func ClassifyBucket(q QuestionResult, memoryConversationIDs map[string]string) string {
	if HasOutOfConversationTopHit(q, memoryConversationIDs) {
		return BucketWrongBranchRetrieval
	}
	if HasMissingCompanion(q) {
		return BucketMissingCompanion
	}
	if q.Rank == 0 {
		return BucketMissingCandidate
	}
	if q.Rank > 1 {
		return BucketRankTooLowCandidate
	}
	if q.Category != "" {
		return q.Category
	}
	return BucketUnclassified
}

func HasOutOfConversationTopHit(q QuestionResult, memoryConversationIDs map[string]string) bool {
	if q.ConversationID == "" {
		return false
	}
	for _, id := range rankwindow.IDs(q.RetrievedIDs, 10) {
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
	retrieved := rankwindow.IDSet(q.RetrievedIDs, 10)
	matched := 0
	for _, id := range q.GoldMemoryIDs {
		if _, ok := retrieved[id]; ok {
			matched++
		}
	}
	return matched > 0 && matched < len(q.GoldMemoryIDs)
}
