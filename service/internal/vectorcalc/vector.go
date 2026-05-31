package vectorcalc

import (
	"errors"
	"math"
)

// ValidateEmbedding rejects empty or non-finite embedding vectors before they are
// indexed or compared.
func ValidateEmbedding(vector []float64) error {
	if len(vector) == 0 {
		return errors.New("goncho: embedding vector is empty")
	}
	for _, value := range vector {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return errors.New("goncho: embedding vector contains non-finite value")
		}
	}
	return nil
}

// CosineSimilarity returns normalized dot-product similarity. Zero vectors or
// differently sized vectors cannot provide positive semantic evidence.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for idx := range a {
		dot += a[idx] * b[idx]
		normA += a[idx] * a[idx]
		normB += b[idx] * b[idx]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
