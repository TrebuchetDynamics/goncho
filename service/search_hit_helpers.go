package goncho

func conclusionsFromSearchHits(hits []SearchHit) []string {
	conclusions := make([]string, 0, len(hits))
	for _, hit := range hits {
		if hit.Source == "conclusion" {
			conclusions = append(conclusions, hit.Content)
		}
	}
	return conclusions
}
