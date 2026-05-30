package locomo

import "strings"

const NotFoundRank = 999999

type ComparisonRow struct {
	Question      string
	GoldMemoryIDs []string
	AGoldBestRank int
	BGoldBestRank int
	Winner        string
	DeltaBucket   string
}

func NormalizedRank(rank int) int {
	if rank <= 0 {
		return NotFoundRank
	}
	return rank
}

func CompareWinner(aRank, bRank int) string {
	if aRank == NotFoundRank && bRank == NotFoundRank {
		return "both_miss"
	}
	if aRank < bRank {
		return "a"
	}
	if bRank < aRank {
		return "b"
	}
	return "tie"
}

func ClassifyDeltaBucket(row ComparisonRow) string {
	switch row.Winner {
	case "a":
		if row.BGoldBestRank == NotFoundRank {
			return "a_only_hit"
		}
		return "a_rank_better"
	case "b":
		if row.AGoldBestRank == NotFoundRank {
			return "b_only_hit"
		}
		return "b_rank_better"
	case "tie":
		return "same_rank"
	case "both_miss":
		return "both_miss"
	default:
		return "unknown"
	}
}

func ClassifyComparison(row ComparisonRow) string {
	q := strings.ToLower(row.Question)
	if len(row.GoldMemoryIDs) > 1 {
		return "gold_ambiguity"
	}
	switch row.DeltaBucket {
	case "a_only_hit":
		return "b_candidate_missing"
	case "a_rank_better":
		return "b_rank_regression"
	case "b_only_hit":
		return "b_candidate_improvement"
	case "b_rank_better":
		return "b_rank_improvement"
	}
	if containsAny(q, []string{"who ", "said", "told", "mentioned", "according to"}) {
		return "speaker_attribution"
	}
	if containsAny(q, []string{"now", "current", "currently", "latest", "recent", "before", "after", "when", "how long"}) {
		return "temporal_evolution"
	}
	if containsAny(q, []string{"replace", "changed", "migrated", "used to", "formerly", "instead"}) {
		return "contradiction_handling"
	}
	if containsAny(q, []string{"which", "what", "where", "when", "how many", "how much"}) {
		return "entity_exactness"
	}
	if row.Winner == "both_miss" {
		return "unknown"
	}
	return "lexical_grounding"
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
