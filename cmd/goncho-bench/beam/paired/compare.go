package paired

import (
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/matchcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
)

const beamPairedComparisonBootstrapSeed int64 = 42

type beamPairedComparisonReport = comparisoncontract.Report

type beamPairedComparisonCI = comparisoncontract.CI

type beamPairedComparisonStats = comparisoncontract.Stats

type beamPairedComparisonRow = comparisoncontract.Row

func RunPairedComparison(cfg Config) error {
	report, err := buildBeamPairedComparison(cfg)
	if err != nil {
		return err
	}
	if err := writeBeamPairedComparisonJSON(cfg.CompareJSONOut, cfg.CompareMarkdownOut, report); err != nil {
		return err
	}
	return writeBeamPairedComparisonMarkdown(cfg.CompareMarkdownOut, cfg.CompareJSONOut, report)
}

func buildBeamPairedComparison(cfg Config) (beamPairedComparisonReport, error) {
	path := strings.TrimSpace(cfg.ComparePath)
	baselineID := strings.TrimSpace(cfg.BaselineConfigID)
	candidateID := strings.TrimSpace(cfg.CandidateConfigID)
	if path == "" {
		return beamPairedComparisonReport{}, fmt.Errorf("goncho-bench: --beam-paired-compare is required")
	}
	if baselineID == "" || candidateID == "" {
		return beamPairedComparisonReport{}, fmt.Errorf("goncho-bench: --beam-paired-baseline-config-id and --beam-paired-candidate-config-id are required")
	}
	if baselineID == candidateID {
		return beamPairedComparisonReport{}, fmt.Errorf("goncho-bench: paired comparison config IDs must differ")
	}
	rows, err := loadBeamPairedOutcomes(path)
	if err != nil {
		return beamPairedComparisonReport{}, err
	}
	baselineRows, candidateRows := []servicePairedOutcome{}, []servicePairedOutcome{}
	for _, row := range rows {
		if !shared.HasNonEmptyTrimmed(row.QID) {
			continue
		}
		switch strings.TrimSpace(row.ConfigID) {
		case baselineID:
			baselineRows = append(baselineRows, row)
		case candidateID:
			candidateRows = append(candidateRows, row)
		}
	}
	matchedRows, dropped, err := matchcontract.MatchOutcomes(baselineRows, candidateRows)
	if err != nil {
		return beamPairedComparisonReport{}, err
	}
	if len(matchedRows) == 0 {
		return beamPairedComparisonReport{}, fmt.Errorf("goncho-bench: no paired BEAM outcomes for config_id %q vs %q", baselineID, candidateID)
	}
	comparisonRows := make([]beamPairedComparisonRow, 0, len(matchedRows))
	for _, matched := range matchedRows {
		base, cand := matched.Baseline, matched.Candidate
		ability := shared.FirstNonEmptyTrimmed(shared.NormalizeAbility(cand.Ability), shared.NormalizeAbility(base.Ability))
		question := shared.FirstNonEmptyTrimmed(cand.Question, base.Question)
		scale := shared.FirstNonEmptyTrimmed(cand.Scale, base.Scale)
		conversationID := shared.FirstNonEmptyTrimmed(cand.ConversationID, base.ConversationID)
		qid := shared.FirstNonEmptyTrimmed(cand.QID, base.QID)
		delta := shared.RoundSignedMetric(cand.Score - base.Score)
		comparisonRows = append(comparisonRows, beamPairedComparisonRow{
			Scale:                 scale,
			ConversationID:        conversationID,
			QID:                   qid,
			BaselineQID:           strings.TrimSpace(base.QID),
			CandidateQID:          strings.TrimSpace(cand.QID),
			BaselineSourcePath:    strings.TrimSpace(base.SourcePath),
			BaselineSourceSHA256:  strings.TrimSpace(base.SourceSHA256),
			CandidateSourcePath:   strings.TrimSpace(cand.SourcePath),
			CandidateSourceSHA256: strings.TrimSpace(cand.SourceSHA256),
			MatchKey:              matched.MatchKey,
			Ability:               ability,
			Question:              question,
			BaselineScore:         shared.RoundMetric(base.Score),
			CandidateScore:        shared.RoundMetric(cand.Score),
			ScoreDelta:            delta,
			BaselineCorrect:       base.Correct,
			CandidateCorrect:      cand.Correct,
			Winner:                comparisoncontract.Winner(base.Score, cand.Score),
		})
	}
	bootstrapSamples := cfg.CompareBootstrapSamples
	if bootstrapSamples <= 0 {
		bootstrapSamples = 5000
	}
	effectSizeFloor := cfg.CompareEffectSizeFloor
	if effectSizeFloor <= 0 {
		effectSizeFloor = 0.02
	}
	report := summarizeBeamPairedComparison(comparisonRows, bootstrapSamples, effectSizeFloor)
	report.GeneratedAt = shared.FormatArtifactTimestamp(time.Now())
	report.SourcePath = path
	report.BaselineConfigID = baselineID
	report.CandidateConfigID = candidateID
	report.DroppedUnpairedCount = dropped
	return report, nil
}

func loadBeamPairedOutcomes(path string) ([]servicePairedOutcome, error) {
	return shared.ReadJSONLFile[servicePairedOutcome](path, "goncho-bench: read BEAM paired outcomes", "goncho-bench: scan BEAM paired outcomes", "goncho-bench: decode BEAM paired outcome")
}

func summarizeBeamPairedComparison(rows []beamPairedComparisonRow, bootstrapSamples int, effectSizeFloor float64) beamPairedComparisonReport {
	return comparisoncontract.SummarizeRows(rows, bootstrapSamples, beamPairedComparisonBootstrapSeed, effectSizeFloor)
}

func writeBeamPairedComparisonJSON(jsonOut, markdownOut string, report beamPairedComparisonReport) error {
	jsonOut = strings.TrimSpace(jsonOut)
	markdownOut = strings.TrimSpace(markdownOut)
	if jsonOut == "" && markdownOut != "" {
		return nil
	}
	raw, err := shared.MarshalIndentedJSON(report)
	if err != nil {
		return fmt.Errorf("goncho-bench: encode BEAM paired comparison: %w", err)
	}
	if jsonOut == "" {
		jsonOut = "-"
	}
	return shared.WriteBytesArtifact(jsonOut, raw, "goncho-bench: create BEAM paired comparison JSON dir", "goncho-bench: write BEAM paired comparison JSON")
}

func writeBeamPairedComparisonMarkdown(path, jsonPath string, report beamPairedComparisonReport) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	var b strings.Builder
	b.WriteString("# BEAM Paired Outcome Comparison\n\n")
	b.WriteString("Deterministic paired comparison over Mnemosyne-compatible `paired_outcomes.jsonl` rows. Scores are joined by exact scale/conversation/qid first, then by exact scale/conversation/ability/question when result qids differ; unpaired rows are dropped.\n\n")
	fmt.Fprintf(&b, "- Source: `%s`\n", report.SourcePath)
	fmt.Fprintf(&b, "- Baseline config: `%s`\n", report.BaselineConfigID)
	fmt.Fprintf(&b, "- Candidate config: `%s`\n", report.CandidateConfigID)
	fmt.Fprintf(&b, "- JSON report: `%s`\n", jsonPath)
	fmt.Fprintf(&b, "- Paired questions: `%d`\n", report.PairedCount)
	fmt.Fprintf(&b, "- Dropped unpaired rows: `%d`\n", report.DroppedUnpairedCount)
	fmt.Fprintf(&b, "- Effect-size floor: `%.4f`\n", report.EffectSizeFloor)
	fmt.Fprintf(&b, "- Verdict: `%s` (`%s`)\n", report.Conclusion, report.ConclusionReason)
	fmt.Fprintf(&b, "- Bootstrap: `%d` samples, seed `%d`, score-delta 95%% CI [`%+.4f`, `%+.4f`]\n\n", report.BootstrapSamples, report.BootstrapSeed, report.ScoreDeltaCI95.Lower, report.ScoreDeltaCI95.Upper)
	b.WriteString("## Score summary\n\n")
	b.WriteString("| Ability | Paired | Baseline avg | Candidate avg | Δ score | Candidate wins | Baseline wins | Ties |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	fmt.Fprintf(&b, "| OVERALL | %d | %.4f | %.4f | %+.4f | %d | %d | %d |\n", report.PairedCount, report.BaselineAvgScore, report.CandidateAvgScore, report.ScoreDelta, report.CandidateWins, report.BaselineWins, report.Ties)
	for _, ability := range sortedBeamPairedAbilities(report.ByAbility) {
		stats := report.ByAbility[ability]
		fmt.Fprintf(&b, "| %s | %d | %.4f | %.4f | %+.4f | %d | %d | %d |\n", ability, stats.PairedCount, stats.BaselineAvgScore, stats.CandidateAvgScore, stats.ScoreDelta, stats.CandidateWins, stats.BaselineWins, stats.Ties)
	}
	b.WriteString("\n## Interpretation\n\n")
	b.WriteString("Use this report as the BEAM arm-comparison oracle: a positive Δ means the candidate config scored higher on the same paired questions. Treat CIs crossing zero as inconclusive and inspect per-ability rows before claiming superiority.\n")
	return shared.WriteFileWithParents(path, []byte(b.String()), "goncho-bench: create BEAM paired comparison Markdown dir", "goncho-bench: write BEAM paired comparison Markdown")
}

func sortedBeamPairedAbilities(byAbility map[string]beamPairedComparisonStats) []string {
	return shared.SortedStringMapKeys(byAbility)
}
