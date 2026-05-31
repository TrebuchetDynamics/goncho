package textmatch

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

// IntersectionSize returns the count of tokens present in both sets.
func IntersectionSize(a, b map[string]struct{}) int {
	shared := 0
	for token := range a {
		if _, ok := b[token]; ok {
			shared++
		}
	}
	return shared
}

// Coverage returns the fraction of wanted tokens present in got.
func Coverage(want, got map[string]struct{}) float64 {
	if len(want) == 0 || len(got) == 0 {
		return 0
	}
	return float64(IntersectionSize(want, got)) / float64(len(want))
}

// Jaccard returns intersection/union similarity for two token sets.
func Jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection := IntersectionSize(a, b)
	union := len(a)
	for token := range b {
		if _, ok := a[token]; !ok {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// JaccardSignificantWords returns Jaccard similarity over lower-cased word tokens
// after stripping common punctuation and dropping tokens shorter than three bytes.
func JaccardSignificantWords(a, b string) float64 {
	return Jaccard(SignificantWordSet(a), SignificantWordSet(b))
}

// SignificantWordSet tokenizes text for lightweight natural-language similarity.
func SignificantWordSet(s string) map[string]struct{} {
	return textutil.Set(strings.Fields(s), func(w string) string {
		w = strings.Trim(strings.ToLower(w), ".,;:!?\"'()[]{}")
		if len(w) <= 2 {
			return ""
		}
		return w
	})
}

// OverlapCoefficient returns intersection/min(len(a), len(b)) for two token sets.
func OverlapCoefficient(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	shared := IntersectionSize(a, b)
	denom := len(a)
	if len(b) < denom {
		denom = len(b)
	}
	if denom == 0 {
		return 0
	}
	return float64(shared) / float64(denom)
}
