package locomo

import benchcomparison "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/locomo/comparison"

const NotFoundRank = benchcomparison.NotFoundRank

type ComparisonRow = benchcomparison.ComparisonRow

func NormalizedRank(rank int) int {
	return benchcomparison.NormalizedRank(rank)
}

func CompareWinner(aRank, bRank int) string {
	return benchcomparison.CompareWinner(aRank, bRank)
}

func ClassifyDeltaBucket(row ComparisonRow) string {
	return benchcomparison.ClassifyDeltaBucket(row)
}

func ClassifyComparison(row ComparisonRow) string {
	return benchcomparison.ClassifyComparison(row)
}
