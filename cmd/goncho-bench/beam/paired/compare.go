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

type beamPairedComparisonReport struct {
	GeneratedAt          string                               `json:"generated_at"`
	SourcePath           string                               `json:"source_path"`
	BaselineConfigID     string                               `json:"baseline_config_id"`
	CandidateConfigID    string                               `json:"candidate_config_id"`
	PairedCount          int                                  `json:"paired_count"`
	DroppedUnpairedCount int                                  `json:"dropped_unpaired_count"`
	BaselineAvgScore     float64                              `json:"baseline_avg_score"`
	CandidateAvgScore    float64                              `json:"candidate_avg_score"`
	ScoreDelta           float64                              `json:"score_delta"`
	EffectSizeFloor      float64                              `json:"effect_size_floor"`
	Conclusion           string                               `json:"conclusion"`
	ConclusionReason     string                               `json:"conclusion_reason"`
	BaselineWins         int                                  `json:"baseline_wins"`
	CandidateWins        int                                  `json:"candidate_wins"`
	Ties                 int                                  `json:"ties"`
	BootstrapSamples     int                                  `json:"bootstrap_samples"`
	BootstrapSeed        int64                                `json:"bootstrap_seed"`
	ScoreDeltaCI95       beamPairedComparisonCI               `json:"score_delta_ci95"`
	ByAbility            map[string]beamPairedComparisonStats `json:"by_ability"`
	Rows                 []beamPairedComparisonRow            `json:"rows"`
}

type beamPairedComparisonCI = comparisoncontract.CI

type beamPairedComparisonStats struct {
	PairedCount       int     `json:"paired_count"`
	BaselineAvgScore  float64 `json:"baseline_avg_score"`
	CandidateAvgScore float64 `json:"candidate_avg_score"`
	ScoreDelta        float64 `json:"score_delta"`
	Conclusion        string  `json:"conclusion"`
	ConclusionReason  string  `json:"conclusion_reason"`
	BaselineWins      int     `json:"baseline_wins"`
	CandidateWins     int     `json:"candidate_wins"`
	Ties              int     `json:"ties"`
}

type beamPairedComparisonRow struct {
	Scale                 string  `json:"scale"`
	ConversationID        string  `json:"conversation_id"`
	QID                   string  `json:"qid"`
	BaselineQID           string  `json:"baseline_qid,omitempty"`
	CandidateQID          string  `json:"candidate_qid,omitempty"`
	BaselineSourcePath    string  `json:"baseline_source_path,omitempty"`
	BaselineSourceSHA256  string  `json:"baseline_source_sha256,omitempty"`
	CandidateSourcePath   string  `json:"candidate_source_path,omitempty"`
	CandidateSourceSHA256 string  `json:"candidate_source_sha256,omitempty"`
	MatchKey              string  `json:"match_key"`
	Ability               string  `json:"ability"`
	Question              string  `json:"question,omitempty"`
	BaselineScore         float64 `json:"baseline_score"`
	CandidateScore        float64 `json:"candidate_score"`
	ScoreDelta            float64 `json:"score_delta"`
	BaselineCorrect       bool    `json:"baseline_correct"`
	CandidateCorrect      bool    `json:"candidate_correct"`
	Winner                string  `json:"winner"`
}

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
	report := beamPairedComparisonReport{
		PairedCount:      len(rows),
		EffectSizeFloor:  shared.RoundMetric(effectSizeFloor),
		BootstrapSamples: bootstrapSamples,
		BootstrapSeed:    beamPairedComparisonBootstrapSeed,
		ByAbility:        map[string]beamPairedComparisonStats{},
		Rows:             append([]beamPairedComparisonRow(nil), rows...),
	}
	abilityRows := map[string][]beamPairedComparisonRow{}
	var baselineTally, candidateTally shared.ScoreTally
	diffs := make([]float64, 0, len(rows))
	for _, row := range rows {
		baselineTally.Add(row.BaselineScore)
		candidateTally.Add(row.CandidateScore)
		diffs = append(diffs, row.CandidateScore-row.BaselineScore)
		switch row.Winner {
		case "candidate":
			report.CandidateWins++
		case "baseline":
			report.BaselineWins++
		default:
			report.Ties++
		}
		abilityRows[row.Ability] = append(abilityRows[row.Ability], row)
	}
	report.BaselineAvgScore = baselineTally.Average()
	report.CandidateAvgScore = candidateTally.Average()
	report.ScoreDelta = shared.RoundSignedMetric(report.CandidateAvgScore - report.BaselineAvgScore)
	report.ScoreDeltaCI95 = comparisoncontract.BootstrapMeanCI(diffs, bootstrapSamples, beamPairedComparisonBootstrapSeed)
	report.Conclusion, report.ConclusionReason = comparisoncontract.Conclusion(report.ScoreDeltaCI95, report.EffectSizeFloor)
	for ability, rows := range abilityRows {
		report.ByAbility[ability] = beamPairedComparisonStatsForRows(rows, report.EffectSizeFloor)
	}
	return report
}

func beamPairedComparisonStatsForRows(rows []beamPairedComparisonRow, effectSizeFloor float64) beamPairedComparisonStats {
	stats := beamPairedComparisonStats{PairedCount: len(rows)}
	var baselineTally, candidateTally shared.ScoreTally
	for _, row := range rows {
		baselineTally.Add(row.BaselineScore)
		candidateTally.Add(row.CandidateScore)
		switch row.Winner {
		case "candidate":
			stats.CandidateWins++
		case "baseline":
			stats.BaselineWins++
		default:
			stats.Ties++
		}
	}
	stats.BaselineAvgScore = baselineTally.Average()
	stats.CandidateAvgScore = candidateTally.Average()
	stats.ScoreDelta = shared.RoundSignedMetric(stats.CandidateAvgScore - stats.BaselineAvgScore)
	stats.Conclusion, stats.ConclusionReason = comparisoncontract.PointConclusion(stats.ScoreDelta, effectSizeFloor)
	return stats
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
