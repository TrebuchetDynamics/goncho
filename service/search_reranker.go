package goncho

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
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
	if reranker == nil || textutil.IsBlank(query) || len(hits) < 2 {
		return hits
	}
	plans := searchRerankCandidatePlans(hits)
	if len(plans) < 2 {
		return hits
	}
	scored, err := reranker.RerankSearch(ctx, query, searchRerankCandidates(plans))
	if err != nil || len(scored) == 0 {
		return hits
	}
	scoresByHit := searchRerankScoresByHit(plans, scored)
	if len(scoresByHit) == 0 {
		return hits
	}
	type scoredSearchHit struct {
		Hit   SearchHit
		Index int
	}
	out := sliceutil.Map(hits, func(hit SearchHit) scoredSearchHit {
		return scoredSearchHit{Hit: hit}
	})
	for i := range out {
		out[i].Index = i
	}
	sort.SliceStable(out, func(i, j int) bool {
		left, leftOK := scoresByHit[out[i].Index]
		right, rightOK := scoresByHit[out[j].Index]
		if leftOK != rightOK {
			return leftOK
		}
		if left == right {
			return false
		}
		return left > right
	})
	return sliceutil.Map(out, func(item scoredSearchHit) SearchHit {
		return item.Hit
	})
}

func searchRerankScoresByHit(plans []searchRerankCandidatePlan, scored []SearchRerankScore) map[int]float64 {
	scoresByID := searchRerankFiniteScoresByID(scored)
	if len(scoresByID) == 0 {
		return nil
	}
	scoresByHit := map[int]float64{}
	for _, plan := range plans {
		if score, ok := scoresByID[plan.ID]; ok {
			scoresByHit[plan.Index] = score
		}
	}
	return scoresByHit
}

func searchRerankFiniteScoresByID(scored []SearchRerankScore) map[string]float64 {
	scoresByID := map[string]float64{}
	for _, score := range scored {
		id := strings.TrimSpace(score.ID)
		if id == "" || !searchRerankScoreIsFinite(score.Score) {
			continue
		}
		scoresByID[id] = score.Score
	}
	return scoresByID
}

func searchRerankScoreIsFinite(score float64) bool {
	return !math.IsNaN(score) && !math.IsInf(score, 0)
}

type searchRerankCandidatePlan struct {
	Index     int
	ID        string
	Candidate SearchRerankCandidate
}

func searchRerankCandidatePlans(hits []SearchHit) []searchRerankCandidatePlan {
	plans := make([]searchRerankCandidatePlan, 0, len(hits))
	seen := map[string]int{}
	for i, hit := range hits {
		id := uniqueSearchHitRerankID(hit, seen)
		if id == "" || textutil.IsBlank(hit.Content) {
			continue
		}
		plans = append(plans, searchRerankCandidatePlan{
			Index:     i,
			ID:        id,
			Candidate: SearchRerankCandidate{ID: id, Source: hit.Source, Content: hit.Content},
		})
	}
	return plans
}

func searchRerankCandidates(plans []searchRerankCandidatePlan) []SearchRerankCandidate {
	return sliceutil.Map(plans, func(plan searchRerankCandidatePlan) SearchRerankCandidate {
		return plan.Candidate
	})
}

func uniqueSearchHitRerankID(hit SearchHit, seen map[string]int) string {
	id := searchHitRerankID(hit)
	if id == "" {
		return ""
	}
	seen[id]++
	if seen[id] == 1 {
		return id
	}
	return id + "#" + strconv.Itoa(seen[id])
}

func searchHitRerankID(hit SearchHit) string {
	if hit.ID > 0 {
		return strconv.FormatInt(hit.ID, 10)
	}
	for _, evidence := range hit.Provenance {
		if textutil.NonBlank(evidence.ID) {
			return strings.TrimSpace(evidence.ID)
		}
	}
	return "content:" + strings.TrimSpace(hit.Content)
}
