package vectorcalc

import (
	"math"
	"testing"
)

func TestValidateEmbedding(t *testing.T) {
	if err := ValidateEmbedding([]float64{1, 0.5}); err != nil {
		t.Fatalf("valid vector returned error: %v", err)
	}
	if err := ValidateEmbedding(nil); err == nil {
		t.Fatal("empty vector returned nil error")
	}
	if err := ValidateEmbedding([]float64{math.NaN()}); err == nil {
		t.Fatal("NaN vector returned nil error")
	}
	if err := ValidateEmbedding([]float64{math.Inf(1)}); err == nil {
		t.Fatal("Inf vector returned nil error")
	}
}

func TestCosineSimilarity(t *testing.T) {
	if got := CosineSimilarity([]float64{1, 0}, []float64{1, 0}); got != 1 {
		t.Fatalf("identical cosine = %v, want 1", got)
	}
	if got := CosineSimilarity([]float64{1, 0}, []float64{0, 1}); got != 0 {
		t.Fatalf("orthogonal cosine = %v, want 0", got)
	}
	if got := CosineSimilarity([]float64{1}, []float64{1, 2}); got != 0 {
		t.Fatalf("mismatched cosine = %v, want 0", got)
	}
	if got := CosineSimilarity([]float64{0, 0}, []float64{1, 0}); got != 0 {
		t.Fatalf("zero-vector cosine = %v, want 0", got)
	}
}
