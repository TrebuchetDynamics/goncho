package serviceartifact

import (
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/casecontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

// SummaryOptions contains run metadata supplied by the oracle service facade.
type SummaryOptions struct {
	ConfigID    string
	RunStarted  time.Time
	JudgeModel  string
	Description string
	PureRecall  bool
	Score       CaseScoreFunc
}

// BuildSummary projects a recall report into the BEAM service summary artifact contract.
func BuildSummary(report goncho.RecallBenchmarkReport, opts SummaryOptions) SummaryFile {
	type scaleStats struct {
		abilityTallies map[string]*shared.ScoreTally
		overallTally   shared.ScoreTally
	}
	stats := map[string]*scaleStats{}
	for _, c := range report.Cases {
		ability := shared.NormalizeAbility(c.Ability)
		if ability == "" {
			continue
		}
		scale := casecontract.Scale(c)
		acc := stats[scale]
		if acc == nil {
			acc = &scaleStats{abilityTallies: map[string]*shared.ScoreTally{}}
			stats[scale] = acc
		}
		tally := acc.abilityTallies[ability]
		if tally == nil {
			tally = &shared.ScoreTally{}
			acc.abilityTallies[ability] = tally
		}
		score := opts.Score(c)
		tally.Add(score)
		acc.overallTally.Add(score)
	}
	abilitySummary := map[string]map[string]AbilityStats{}
	for scale, acc := range stats {
		byAbility := map[string]AbilityStats{}
		for ability, tally := range acc.abilityTallies {
			byAbility[ability] = AbilityStats{AvgScore: tally.Average(), Count: tally.Count()}
		}
		if acc.overallTally.Count() > 0 {
			byAbility["OVERALL"] = AbilityStats{AvgScore: acc.overallTally.Average(), Count: acc.overallTally.Count()}
		}
		abilitySummary[scale] = byAbility
	}
	return SummaryFile{
		Date: opts.RunStarted.UTC().Format(time.RFC3339),
		Metadata: SummaryMetadata{
			Model:       casecontract.ModelName,
			SampleSize:  report.CaseCount,
			JudgeModel:  opts.JudgeModel,
			ConfigID:    opts.ConfigID,
			PureRecall:  opts.PureRecall,
			Service:     report.Service,
			Corpus:      report.CorpusVersion,
			CaseCount:   report.CaseCount,
			Description: opts.Description,
		},
		AbilitySummary: abilitySummary,
	}
}
