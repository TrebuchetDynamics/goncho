package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

type beamPairedComparisonCI struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
}

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
	Scale            string  `json:"scale"`
	ConversationID   string  `json:"conversation_id"`
	QID              string  `json:"qid"`
	BaselineQID      string  `json:"baseline_qid,omitempty"`
	CandidateQID     string  `json:"candidate_qid,omitempty"`
	MatchKey         string  `json:"match_key"`
	Ability          string  `json:"ability"`
	Question         string  `json:"question,omitempty"`
	BaselineScore    float64 `json:"baseline_score"`
	CandidateScore   float64 `json:"candidate_score"`
	ScoreDelta       float64 `json:"score_delta"`
	BaselineCorrect  bool    `json:"baseline_correct"`
	CandidateCorrect bool    `json:"candidate_correct"`
	Winner           string  `json:"winner"`
}

type beamPairedComparisonKey struct {
	scale          string
	conversationID string
	qid            string
}

type beamPairedComparisonQuestionKey struct {
	scale          string
	conversationID string
	ability        string
	question       string
}

type beamPairedMatchedOutcome struct {
	baseline  beamServicePairedOutcome
	candidate beamServicePairedOutcome
	matchKey  string
}

func runBeamPairedComparison(cfg config) error {
	report, err := buildBeamPairedComparison(cfg)
	if err != nil {
		return err
	}
	if err := writeBeamPairedComparisonJSON(cfg.BeamPairedCompareJSONOut, cfg.BeamPairedCompareMarkdownOut, report); err != nil {
		return err
	}
	return writeBeamPairedComparisonMarkdown(cfg.BeamPairedCompareMarkdownOut, cfg.BeamPairedCompareJSONOut, report)
}

func buildBeamPairedComparison(cfg config) (beamPairedComparisonReport, error) {
	path := strings.TrimSpace(cfg.BeamPairedComparePath)
	baselineID := strings.TrimSpace(cfg.BeamPairedBaselineConfigID)
	candidateID := strings.TrimSpace(cfg.BeamPairedCandidateConfigID)
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
	baselineRows, candidateRows := []beamServicePairedOutcome{}, []beamServicePairedOutcome{}
	for _, row := range rows {
		if strings.TrimSpace(row.QID) == "" {
			continue
		}
		switch strings.TrimSpace(row.ConfigID) {
		case baselineID:
			baselineRows = append(baselineRows, row)
		case candidateID:
			candidateRows = append(candidateRows, row)
		}
	}
	matchedRows, dropped := matchBeamPairedOutcomes(baselineRows, candidateRows)
	if len(matchedRows) == 0 {
		return beamPairedComparisonReport{}, fmt.Errorf("goncho-bench: no paired BEAM outcomes for config_id %q vs %q", baselineID, candidateID)
	}
	comparisonRows := make([]beamPairedComparisonRow, 0, len(matchedRows))
	for _, matched := range matchedRows {
		base, cand := matched.baseline, matched.candidate
		ability := strings.ToUpper(strings.TrimSpace(cand.Ability))
		if ability == "" {
			ability = strings.ToUpper(strings.TrimSpace(base.Ability))
		}
		question := strings.TrimSpace(cand.Question)
		if question == "" {
			question = strings.TrimSpace(base.Question)
		}
		scale := strings.TrimSpace(cand.Scale)
		if scale == "" {
			scale = strings.TrimSpace(base.Scale)
		}
		conversationID := strings.TrimSpace(cand.ConversationID)
		if conversationID == "" {
			conversationID = strings.TrimSpace(base.ConversationID)
		}
		qid := strings.TrimSpace(cand.QID)
		if qid == "" {
			qid = strings.TrimSpace(base.QID)
		}
		delta := roundSignedMetric(cand.Score - base.Score)
		comparisonRows = append(comparisonRows, beamPairedComparisonRow{
			Scale:            scale,
			ConversationID:   conversationID,
			QID:              qid,
			BaselineQID:      strings.TrimSpace(base.QID),
			CandidateQID:     strings.TrimSpace(cand.QID),
			MatchKey:         matched.matchKey,
			Ability:          ability,
			Question:         question,
			BaselineScore:    roundMetric(base.Score),
			CandidateScore:   roundMetric(cand.Score),
			ScoreDelta:       delta,
			BaselineCorrect:  base.Correct,
			CandidateCorrect: cand.Correct,
			Winner:           beamPairedComparisonWinner(base.Score, cand.Score),
		})
	}
	bootstrapSamples := cfg.BeamPairedCompareBootstrapSamples
	if bootstrapSamples <= 0 {
		bootstrapSamples = 5000
	}
	effectSizeFloor := cfg.BeamPairedCompareEffectSizeFloor
	if effectSizeFloor <= 0 {
		effectSizeFloor = 0.02
	}
	report := summarizeBeamPairedComparison(comparisonRows, bootstrapSamples, effectSizeFloor)
	report.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	report.SourcePath = path
	report.BaselineConfigID = baselineID
	report.CandidateConfigID = candidateID
	report.DroppedUnpairedCount = dropped
	return report, nil
}

func loadBeamPairedOutcomes(path string) ([]beamServicePairedOutcome, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("goncho-bench: read BEAM paired outcomes: %w", err)
	}
	defer file.Close()
	rows := []beamServicePairedOutcome{}
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row beamServicePairedOutcome
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("goncho-bench: decode BEAM paired outcome line %d: %w", lineNumber, err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("goncho-bench: scan BEAM paired outcomes: %w", err)
	}
	return rows, nil
}

func beamPairedOutcomeKey(row beamServicePairedOutcome) beamPairedComparisonKey {
	return beamPairedComparisonKey{
		scale:          strings.TrimSpace(row.Scale),
		conversationID: strings.TrimSpace(row.ConversationID),
		qid:            strings.TrimSpace(row.QID),
	}
}

func beamPairedOutcomeQuestionKey(row beamServicePairedOutcome) beamPairedComparisonQuestionKey {
	question := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(row.Question))), " ")
	if question == "" {
		return beamPairedComparisonQuestionKey{}
	}
	return beamPairedComparisonQuestionKey{
		scale:          strings.TrimSpace(row.Scale),
		conversationID: strings.TrimSpace(row.ConversationID),
		ability:        strings.ToUpper(strings.TrimSpace(row.Ability)),
		question:       question,
	}
}

func matchBeamPairedOutcomes(baselineRows, candidateRows []beamServicePairedOutcome) ([]beamPairedMatchedOutcome, int) {
	sort.Slice(baselineRows, func(i, j int) bool { return beamServicePairedOutcomeLess(baselineRows[i], baselineRows[j]) })
	sort.Slice(candidateRows, func(i, j int) bool { return beamServicePairedOutcomeLess(candidateRows[i], candidateRows[j]) })
	candidateByQID := map[beamPairedComparisonKey]int{}
	candidateByQuestion := map[beamPairedComparisonQuestionKey]int{}
	for i, row := range candidateRows {
		if key := beamPairedOutcomeKey(row); key.qid != "" {
			if _, ok := candidateByQID[key]; !ok {
				candidateByQID[key] = i
			}
		}
		if key := beamPairedOutcomeQuestionKey(row); key.question != "" {
			if _, ok := candidateByQuestion[key]; !ok {
				candidateByQuestion[key] = i
			}
		}
	}
	usedCandidates := map[int]struct{}{}
	matched := []beamPairedMatchedOutcome{}
	dropped := 0
	for _, base := range baselineRows {
		if idx, ok := candidateByQID[beamPairedOutcomeKey(base)]; ok {
			if _, used := usedCandidates[idx]; !used {
				usedCandidates[idx] = struct{}{}
				matched = append(matched, beamPairedMatchedOutcome{baseline: base, candidate: candidateRows[idx], matchKey: "qid"})
				continue
			}
		}
		if questionKey := beamPairedOutcomeQuestionKey(base); questionKey.question != "" {
			if idx, ok := candidateByQuestion[questionKey]; ok {
				if _, used := usedCandidates[idx]; !used {
					usedCandidates[idx] = struct{}{}
					matched = append(matched, beamPairedMatchedOutcome{baseline: base, candidate: candidateRows[idx], matchKey: "question"})
					continue
				}
			}
		}
		dropped++
	}
	dropped += len(candidateRows) - len(usedCandidates)
	return matched, dropped
}

func beamServicePairedOutcomeLess(a, b beamServicePairedOutcome) bool {
	ak, bk := beamPairedOutcomeKey(a), beamPairedOutcomeKey(b)
	if !beamPairedComparisonKeyEqual(ak, bk) {
		return beamPairedComparisonKeyLess(ak, bk)
	}
	aq, bq := beamPairedOutcomeQuestionKey(a), beamPairedOutcomeQuestionKey(b)
	if aq.question != bq.question {
		return aq.question < bq.question
	}
	return strings.TrimSpace(a.ConfigID) < strings.TrimSpace(b.ConfigID)
}

func beamPairedComparisonKeyEqual(a, b beamPairedComparisonKey) bool {
	return a.scale == b.scale && a.conversationID == b.conversationID && a.qid == b.qid
}

func beamPairedComparisonKeyLess(a, b beamPairedComparisonKey) bool {
	if a.scale != b.scale {
		return a.scale < b.scale
	}
	if a.conversationID != b.conversationID {
		return a.conversationID < b.conversationID
	}
	return a.qid < b.qid
}

func beamPairedComparisonWinner(baseScore, candidateScore float64) string {
	switch {
	case candidateScore > baseScore:
		return "candidate"
	case baseScore > candidateScore:
		return "baseline"
	default:
		return "tie"
	}
}

func summarizeBeamPairedComparison(rows []beamPairedComparisonRow, bootstrapSamples int, effectSizeFloor float64) beamPairedComparisonReport {
	report := beamPairedComparisonReport{
		PairedCount:      len(rows),
		EffectSizeFloor:  roundMetric(effectSizeFloor),
		BootstrapSamples: bootstrapSamples,
		BootstrapSeed:    beamPairedComparisonBootstrapSeed,
		ByAbility:        map[string]beamPairedComparisonStats{},
		Rows:             append([]beamPairedComparisonRow(nil), rows...),
	}
	abilityRows := map[string][]beamPairedComparisonRow{}
	baseTotal, candidateTotal := 0.0, 0.0
	diffs := make([]float64, 0, len(rows))
	for _, row := range rows {
		baseTotal += row.BaselineScore
		candidateTotal += row.CandidateScore
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
	n := float64(len(rows))
	report.BaselineAvgScore = roundMetric(baseTotal / n)
	report.CandidateAvgScore = roundMetric(candidateTotal / n)
	report.ScoreDelta = roundSignedMetric(report.CandidateAvgScore - report.BaselineAvgScore)
	report.ScoreDeltaCI95 = bootstrapMeanCI(diffs, bootstrapSamples)
	report.Conclusion, report.ConclusionReason = beamPairedComparisonConclusion(report.ScoreDeltaCI95, report.EffectSizeFloor)
	for ability, rows := range abilityRows {
		report.ByAbility[ability] = beamPairedComparisonStatsForRows(rows, report.EffectSizeFloor)
	}
	return report
}

func beamPairedComparisonStatsForRows(rows []beamPairedComparisonRow, effectSizeFloor float64) beamPairedComparisonStats {
	stats := beamPairedComparisonStats{PairedCount: len(rows)}
	baseTotal, candidateTotal := 0.0, 0.0
	for _, row := range rows {
		baseTotal += row.BaselineScore
		candidateTotal += row.CandidateScore
		switch row.Winner {
		case "candidate":
			stats.CandidateWins++
		case "baseline":
			stats.BaselineWins++
		default:
			stats.Ties++
		}
	}
	n := float64(len(rows))
	stats.BaselineAvgScore = roundMetric(baseTotal / n)
	stats.CandidateAvgScore = roundMetric(candidateTotal / n)
	stats.ScoreDelta = roundSignedMetric(stats.CandidateAvgScore - stats.BaselineAvgScore)
	stats.Conclusion, stats.ConclusionReason = beamPairedComparisonPointConclusion(stats.ScoreDelta, effectSizeFloor)
	return stats
}

func beamPairedComparisonConclusion(ci beamPairedComparisonCI, effectSizeFloor float64) (string, string) {
	if ci.Lower > effectSizeFloor {
		return "candidate_superior", "candidate_ci_above_effect_floor"
	}
	if ci.Upper < -effectSizeFloor {
		return "baseline_superior", "baseline_ci_below_negative_effect_floor"
	}
	return "inconclusive", "ci_overlaps_effect_floor"
}

func beamPairedComparisonPointConclusion(delta, effectSizeFloor float64) (string, string) {
	if delta > effectSizeFloor {
		return "candidate_superior", "candidate_delta_above_effect_floor"
	}
	if delta < -effectSizeFloor {
		return "baseline_superior", "baseline_delta_below_negative_effect_floor"
	}
	return "inconclusive", "delta_within_effect_floor"
}

func roundSignedMetric(v float64) float64 {
	if v < 0 {
		return -roundMetric(-v)
	}
	return roundMetric(v)
}

func bootstrapMeanCI(values []float64, samples int) beamPairedComparisonCI {
	if len(values) == 0 || samples <= 0 {
		return beamPairedComparisonCI{}
	}
	rng := rand.New(rand.NewSource(beamPairedComparisonBootstrapSeed))
	means := make([]float64, 0, samples)
	for i := 0; i < samples; i++ {
		total := 0.0
		for range values {
			total += values[rng.Intn(len(values))]
		}
		means = append(means, total/float64(len(values)))
	}
	sort.Float64s(means)
	lowerIndex := int(0.025 * float64(samples))
	upperIndex := int(0.975 * float64(samples))
	if upperIndex >= len(means) {
		upperIndex = len(means) - 1
	}
	return beamPairedComparisonCI{Lower: roundSignedMetric(means[lowerIndex]), Upper: roundSignedMetric(means[upperIndex])}
}

func writeBeamPairedComparisonJSON(jsonOut, markdownOut string, report beamPairedComparisonReport) error {
	jsonOut = strings.TrimSpace(jsonOut)
	markdownOut = strings.TrimSpace(markdownOut)
	if jsonOut == "" && markdownOut != "" {
		return nil
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("goncho-bench: encode BEAM paired comparison: %w", err)
	}
	raw = append(raw, '\n')
	if jsonOut == "" || jsonOut == "-" {
		_, err = os.Stdout.Write(raw)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(jsonOut), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create BEAM paired comparison JSON dir: %w", err)
	}
	if err := os.WriteFile(jsonOut, raw, 0o644); err != nil {
		return fmt.Errorf("goncho-bench: write BEAM paired comparison JSON: %w", err)
	}
	return nil
}

func writeBeamPairedComparisonMarkdown(path, jsonPath string, report beamPairedComparisonReport) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create BEAM paired comparison Markdown dir: %w", err)
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
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func sortedBeamPairedAbilities(byAbility map[string]beamPairedComparisonStats) []string {
	keys := make([]string, 0, len(byAbility))
	for key := range byAbility {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
