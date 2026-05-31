package goncho

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"

func conclusionsFromSearchHits(hits []SearchHit) []string {
	return sliceutil.FilterMap(hits, func(hit SearchHit) (string, bool) {
		return hit.Content, hit.Source == "conclusion"
	})
}
