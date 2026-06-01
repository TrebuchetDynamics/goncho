package goncho

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/vectorcalc"
)

// HashTextEmbeddingProvider is a deterministic local embedding provider used by
// the built-in vector-index maintenance commands. It performs no network calls
// and is intentionally simple; hosts can still inject higher-quality providers
// through the VectorStore seam.
type HashTextEmbeddingProvider struct {
	Dimensions int
}

func (p HashTextEmbeddingProvider) EmbedText(ctx context.Context, text string) ([]float64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dims := p.Dimensions
	if dims <= 0 {
		dims = 128
	}
	vec := make([]float64, dims)
	for _, token := range strings.Fields(strings.ToLower(text)) {
		token = strings.Trim(token, ".,;:!?()[]{}\"'`")
		if token == "" {
			continue
		}
		bucket := int(fnv32(token) % uint32(dims))
		vec[bucket]++
	}
	if err := vectorcalc.ValidateEmbedding(vec); err != nil {
		return nil, err
	}
	return vec, nil
}

func fnv32(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
