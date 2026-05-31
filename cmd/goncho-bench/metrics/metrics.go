package metrics

import "math"

// Round returns v rounded to Goncho bench's canonical four decimal places.
func Round(v float64) float64 {
	return math.Round(v*10000) / 10000
}

// RoundSigned preserves symmetric rounding for signed deltas.
func RoundSigned(v float64) float64 {
	if v < 0 {
		return -Round(-v)
	}
	return Round(v)
}
