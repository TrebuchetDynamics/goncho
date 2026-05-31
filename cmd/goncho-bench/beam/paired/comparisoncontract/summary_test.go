package comparisoncontract

import "testing"

func TestScoreSummaryAddsRoundedAveragesDeltaAndWins(t *testing.T) {
	var summary ScoreSummary
	summary.Add(0.5, 1.0, "candidate")
	summary.Add(1.0, 0.0, "baseline")
	summary.Add(0.33333, 0.33333, "tie")

	if summary.PairedCount != 3 || summary.CandidateWins != 1 || summary.BaselineWins != 1 || summary.Ties != 1 {
		t.Fatalf("summary counts = %+v, want 3 paired with one candidate/baseline/tie", summary)
	}
	if summary.BaselineAvgScore != 0.6111 || summary.CandidateAvgScore != 0.4444 || summary.ScoreDelta != -0.1667 {
		t.Fatalf("summary scores = baseline %.4f candidate %.4f delta %.4f, want rounded averages and signed delta", summary.BaselineAvgScore, summary.CandidateAvgScore, summary.ScoreDelta)
	}
}
