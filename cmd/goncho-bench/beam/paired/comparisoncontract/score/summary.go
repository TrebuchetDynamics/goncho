package score

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"

const (
	WinnerCandidate = "candidate"
	WinnerBaseline  = "baseline"
	WinnerTie       = "tie"
)

type Summary struct {
	PairedCount       int
	BaselineAvgScore  float64
	CandidateAvgScore float64
	ScoreDelta        float64
	BaselineWins      int
	CandidateWins     int
	Ties              int
	baselineTally     shared.ScoreTally
	candidateTally    shared.ScoreTally
}

func (s *Summary) Add(baselineScore, candidateScore float64, winner string) {
	s.PairedCount++
	s.baselineTally.Add(baselineScore)
	s.candidateTally.Add(candidateScore)
	s.BaselineAvgScore = s.baselineTally.Average()
	s.CandidateAvgScore = s.candidateTally.Average()
	s.ScoreDelta = shared.RoundSignedMetric(s.CandidateAvgScore - s.BaselineAvgScore)
	s.AddWinner(winner)
}

func (s *Summary) AddWinner(winner string) {
	switch winner {
	case WinnerCandidate:
		s.CandidateWins++
	case WinnerBaseline:
		s.BaselineWins++
	default:
		s.Ties++
	}
}

func Winner(baseScore, candidateScore float64) string {
	switch {
	case candidateScore > baseScore:
		return WinnerCandidate
	case baseScore > candidateScore:
		return WinnerBaseline
	default:
		return WinnerTie
	}
}
