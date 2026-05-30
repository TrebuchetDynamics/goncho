package goncho

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

// SearchReranker is an optional host-owned reranking seam. Implementations may
// call a local cross-encoder, remote provider, or deterministic test fake.
// Goncho degrades to the original search order on reranker errors.
type SearchReranker interface {
	RerankSearch(ctx context.Context, query string, candidates []SearchRerankCandidate) ([]SearchRerankScore, error)
}

// SearchRerankCandidate is the privacy-minimal candidate shape passed to an
// optional SearchReranker.
type SearchRerankCandidate struct {
	ID      string `json:"id"`
	Source  string `json:"source,omitempty"`
	Content string `json:"content"`
}

// SearchRerankScore is one scored reranker response row keyed by candidate ID.
type SearchRerankScore struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

func applySearchReranker(ctx context.Context, reranker SearchReranker, query string, hits []SearchHit) []SearchHit {
	if reranker == nil || strings.TrimSpace(query) == "" || len(hits) < 2 {
		return hits
	}
	candidates := make([]SearchRerankCandidate, 0, len(hits))
	for _, hit := range hits {
		id := searchHitRerankID(hit)
		if id == "" || strings.TrimSpace(hit.Content) == "" {
			continue
		}
		candidates = append(candidates, SearchRerankCandidate{ID: id, Source: hit.Source, Content: hit.Content})
	}
	if len(candidates) < 2 {
		return hits
	}
	scored, err := reranker.RerankSearch(ctx, query, candidates)
	if err != nil || len(scored) == 0 {
		return hits
	}
	scores := map[string]float64{}
	for _, score := range scored {
		if id := strings.TrimSpace(score.ID); id != "" {
			scores[id] = score.Score
		}
	}
	if len(scores) == 0 {
		return hits
	}
	out := sliceutil.Clone(hits)
	sort.SliceStable(out, func(i, j int) bool {
		left, leftOK := scores[searchHitRerankID(out[i])]
		right, rightOK := scores[searchHitRerankID(out[j])]
		if leftOK != rightOK {
			return leftOK
		}
		if left == right {
			return false
		}
		return left > right
	})
	return out
}

func searchHitRerankID(hit SearchHit) string {
	if hit.ID > 0 {
		return strconv.FormatInt(hit.ID, 10)
	}
	for _, evidence := range hit.Provenance {
		if strings.TrimSpace(evidence.ID) != "" {
			return strings.TrimSpace(evidence.ID)
		}
	}
	return "content:" + strings.TrimSpace(hit.Content)
}
