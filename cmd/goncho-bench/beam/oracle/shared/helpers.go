package shared

import "math"

func RoundMetric(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func TopN(values []string, n int) []string {
	if n > len(values) {
		n = len(values)
	}
	return append([]string(nil), values[:n]...)
}
