package comparisoncontract

import "testing"

func TestScoreSummaryAddsRoundedAveragesDeltaAndWins(t *testing.T) {
	var summary ScoreSummary
	summary.Add(0.5, 1.0, WinnerCandidate)
	summary.Add(1.0, 0.0, WinnerBaseline)
	summary.Add(0.33333, 0.33333, WinnerTie)

	if summary.PairedCount != 3 || summary.CandidateWins != 1 || summary.BaselineWins != 1 || summary.Ties != 1 {
		t.Fatalf("summary counts = %+v, want 3 paired with one candidate/baseline/tie", summary)
	}
	if summary.BaselineAvgScore != 0.6111 || summary.CandidateAvgScore != 0.4444 || summary.ScoreDelta != -0.1667 {
		t.Fatalf("summary scores = baseline %.4f candidate %.4f delta %.4f, want rounded averages and signed delta", summary.BaselineAvgScore, summary.CandidateAvgScore, summary.ScoreDelta)
	}
}

func TestSummarizeRowsBuildsOverallAndAbilityStats(t *testing.T) {
	report := SummarizeRows([]Row{
		{Ability: "IE", BaselineScore: 0.5, CandidateScore: 1.0, Winner: WinnerCandidate},
		{Ability: "IE", BaselineScore: 1.0, CandidateScore: 0.0, Winner: WinnerBaseline},
		{Ability: "MR", BaselineScore: 0.0, CandidateScore: 1.0, Winner: WinnerCandidate},
	}, 200, 42, 0.02)

	if report.PairedCount != 3 || report.BaselineWins != 1 || report.CandidateWins != 2 || report.Ties != 0 {
		t.Fatalf("report counts = %+v, want overall win counts from rows", report)
	}
	if report.BaselineAvgScore != 0.5 || report.CandidateAvgScore != 0.6667 || report.ScoreDelta != 0.1667 {
		t.Fatalf("report scores = baseline %.4f candidate %.4f delta %.4f, want rounded overall scores", report.BaselineAvgScore, report.CandidateAvgScore, report.ScoreDelta)
	}
	if report.BootstrapSamples != 200 || report.BootstrapSeed != 42 || report.ScoreDeltaCI95 == (CI{}) {
		t.Fatalf("report bootstrap = samples %d seed %d ci %+v, want deterministic non-empty CI", report.BootstrapSamples, report.BootstrapSeed, report.ScoreDeltaCI95)
	}
	if got := report.ByAbility["MR"]; got.PairedCount != 1 || got.ScoreDelta != 1 || got.Conclusion != ConclusionCandidateSuperior {
		t.Fatalf("MR stats = %+v, want candidate-superior single-row ability stats", got)
	}
	if len(report.Rows) != 3 || report.Rows[0].Ability != "IE" {
		t.Fatalf("report rows = %+v, want preserved row copy", report.Rows)
	}
}
