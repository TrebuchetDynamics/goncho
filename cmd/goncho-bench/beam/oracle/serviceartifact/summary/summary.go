package summary

import (
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/casecontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/serviceartifact/filecontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/serviceartifact/scoring"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	sharedscore "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/score"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

// Options contains run metadata supplied by the oracle service facade.
type Options struct {
	ConfigID    string
	RunStarted  time.Time
	JudgeModel  string
	Description string
	PureRecall  bool
	Score       scoring.CaseScoreFunc
}

// Build projects a recall report into the BEAM service summary artifact contract.
func Build(report goncho.RecallBenchmarkReport, opts Options) filecontract.SummaryFile {
	type scaleStats struct {
		abilityTallies map[string]*sharedscore.ScoreTally
		overallTally   sharedscore.ScoreTally
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
			acc = &scaleStats{abilityTallies: map[string]*sharedscore.ScoreTally{}}
			stats[scale] = acc
		}
		tally := acc.abilityTallies[ability]
		if tally == nil {
			tally = &sharedscore.ScoreTally{}
			acc.abilityTallies[ability] = tally
		}
		score := opts.Score(c)
		tally.Add(score)
		acc.overallTally.Add(score)
	}
	abilitySummary := map[string]map[string]filecontract.AbilityStats{}
	for scale, acc := range stats {
		byAbility := map[string]filecontract.AbilityStats{}
		for ability, tally := range acc.abilityTallies {
			byAbility[ability] = filecontract.AbilityStats{AvgScore: tally.Average(), Count: tally.Count()}
		}
		if acc.overallTally.Count() > 0 {
			byAbility["OVERALL"] = filecontract.AbilityStats{AvgScore: acc.overallTally.Average(), Count: acc.overallTally.Count()}
		}
		abilitySummary[scale] = byAbility
	}
	return filecontract.SummaryFile{
		Date: opts.RunStarted.UTC().Format(time.RFC3339),
		Metadata: filecontract.SummaryMetadata{
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
