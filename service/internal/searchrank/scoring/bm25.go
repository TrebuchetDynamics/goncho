package scoring

import "math"

func BM25(queryTokens map[string]struct{}, tf map[string]int, df map[string]int, docCount, docLength int, avgLength float64) float64 {
	if docCount == 0 || docLength == 0 || avgLength <= 0 {
		return 0
	}
	stats := bm25CorpusStats{docCount: docCount, docLength: docLength, avgLength: avgLength}
	score := 0.0
	for token := range queryTokens {
		score += stats.termScore(tf[token], df[token])
	}
	return score
}

type bm25CorpusStats struct {
	docCount  int
	docLength int
	avgLength float64
}

func (stats bm25CorpusStats) termScore(freq, rawDocFreq int) float64 {
	const k1 = 1.2
	const b = 0.75
	docFreq, ok := stats.documentFrequency(rawDocFreq)
	if freq == 0 || !ok {
		return 0
	}
	idf := math.Log(1 + (float64(stats.docCount)-float64(docFreq)+0.5)/(float64(docFreq)+0.5))
	denom := float64(freq) + k1*(1-b+b*(float64(stats.docLength)/stats.avgLength))
	return idf * (float64(freq) * (k1 + 1) / denom)
}

func (stats bm25CorpusStats) documentFrequency(value int) (int, bool) {
	if value <= 0 {
		return 0, false
	}
	if value > stats.docCount {
		return stats.docCount, true
	}
	return value, true
}
