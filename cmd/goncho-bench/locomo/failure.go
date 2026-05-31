package locomo

import benchfailure "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/locomo/failure"

const (
	FailureBucketWrongBranchRetrieval = benchfailure.BucketWrongBranchRetrieval
	FailureBucketMissingCompanion     = benchfailure.BucketMissingCompanion
	FailureBucketMissingCandidate     = benchfailure.BucketMissingCandidate
	FailureBucketRankTooLowCandidate  = benchfailure.BucketRankTooLowCandidate
	FailureBucketUnclassified         = benchfailure.BucketUnclassified
)

type QuestionResult = benchfailure.QuestionResult

func ClassifyFailureBucket(q QuestionResult, memoryConversationIDs map[string]string) string {
	return benchfailure.ClassifyBucket(q, memoryConversationIDs)
}

func HasOutOfConversationTopHit(q QuestionResult, memoryConversationIDs map[string]string) bool {
	return benchfailure.HasOutOfConversationTopHit(q, memoryConversationIDs)
}

func HasMissingCompanion(q QuestionResult) bool {
	return benchfailure.HasMissingCompanion(q)
}
