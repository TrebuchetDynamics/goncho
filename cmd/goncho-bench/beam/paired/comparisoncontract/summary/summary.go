package summary

import (
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract/bootstrap"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract/score"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract/verdict"
	sharedscore "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/score"
)

func SummarizeRows(rows []Row, bootstrapSamples int, bootstrapSeed int64, effectSizeFloor float64) Report {
	report := Report{
		PairedCount:      len(rows),
		EffectSizeFloor:  sharedscore.RoundMetric(effectSizeFloor),
		BootstrapSamples: bootstrapSamples,
		BootstrapSeed:    bootstrapSeed,
		ByAbility:        map[string]Stats{},
		Rows:             append([]Row(nil), rows...),
	}
	abilityRows := map[string][]Row{}
	var summary score.Summary
	diffs := make([]float64, 0, len(rows))
	for _, row := range rows {
		summary.Add(row.BaselineScore, row.CandidateScore, row.Winner)
		diffs = append(diffs, row.CandidateScore-row.BaselineScore)
		abilityRows[row.Ability] = append(abilityRows[row.Ability], row)
	}
	report.BaselineAvgScore = summary.BaselineAvgScore
	report.CandidateAvgScore = summary.CandidateAvgScore
	report.ScoreDelta = summary.ScoreDelta
	report.CandidateWins = summary.CandidateWins
	report.BaselineWins = summary.BaselineWins
	report.Ties = summary.Ties
	report.ScoreDeltaCI95 = bootstrap.MeanCI(diffs, bootstrapSamples, bootstrapSeed)
	report.Conclusion, report.ConclusionReason = verdict.Conclusion(report.ScoreDeltaCI95, report.EffectSizeFloor)
	for ability, rows := range abilityRows {
		report.ByAbility[ability] = StatsForRows(rows, report.EffectSizeFloor)
	}
	return report
}

func StatsForRows(rows []Row, effectSizeFloor float64) Stats {
	var summary score.Summary
	for _, row := range rows {
		summary.Add(row.BaselineScore, row.CandidateScore, row.Winner)
	}
	stats := Stats{
		PairedCount:       summary.PairedCount,
		BaselineAvgScore:  summary.BaselineAvgScore,
		CandidateAvgScore: summary.CandidateAvgScore,
		ScoreDelta:        summary.ScoreDelta,
		BaselineWins:      summary.BaselineWins,
		CandidateWins:     summary.CandidateWins,
		Ties:              summary.Ties,
	}
	stats.Conclusion, stats.ConclusionReason = verdict.PointConclusion(stats.ScoreDelta, effectSizeFloor)
	return stats
}
