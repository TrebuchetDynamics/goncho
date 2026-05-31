package searchrank

import "github.com/TrebuchetDynamics/goncho/service/internal/searchrank/temporal"

type TemporalDirection = temporal.Direction

const (
	TemporalNone  = temporal.DirectionNone
	TemporalNewer = temporal.DirectionNewer
	TemporalOlder = temporal.DirectionOlder
)

type TemporalFeatures = temporal.Features

func TemporalIntent(query string) TemporalFeatures {
	return temporal.Intent(query)
}

func TemporalQuery(query string) bool {
	return temporal.Query(query)
}

func TemporalMarkers(query string) []string {
	return temporal.Markers(query)
}

func TemporalRerankBonus(features TemporalFeatures, content string, index, total int, score, maxScore float64) float64 {
	return temporal.RerankBonus(features, content, index, total, score, maxScore)
}
