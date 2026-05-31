package shared

// ScoreTally accumulates BEAM score totals with the shared rounded-average convention.
type ScoreTally struct {
	total float64
	count int
}

// Add records one score in the tally.
func (t *ScoreTally) Add(score float64) {
	t.total += score
	t.count++
}

// Count returns the number of scores recorded in the tally.
func (t ScoreTally) Count() int {
	return t.count
}

// Average returns the shared rounded BEAM average score, or zero for an empty tally.
func (t ScoreTally) Average() float64 {
	if t.count == 0 {
		return 0
	}
	return RoundMetric(t.total / float64(t.count))
}
