package serviceartifact

import (
	"testing"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestBuildSummaryGroupsScoresByScaleAndAbility(t *testing.T) {
	started := time.Date(2026, 5, 30, 1, 2, 3, 0, time.UTC)
	report := goncho.RecallBenchmarkReport{
		Service:       "svc",
		CorpusVersion: "corpus",
		CaseCount:     3,
		Cases: []goncho.RecallBenchmarkCaseReport{
			{ID: "a", Scale: "10K", Ability: "rw"},
			{ID: "b", Scale: "10K", Ability: "RW"},
			{ID: "c", Scale: "1M", Ability: "abs"},
		},
	}
	scores := map[string]float64{"a": 1, "b": 0.5, "c": 0}

	summary := BuildSummary(report, SummaryOptions{
		ConfigID:    "cfg",
		RunStarted:  started,
		JudgeModel:  "judge",
		Description: "desc",
		PureRecall:  false,
		Score: func(c goncho.RecallBenchmarkCaseReport) float64 {
			return scores[c.ID]
		},
	})

	if summary.Date != "2026-05-30T01:02:03Z" {
		t.Fatalf("Date = %q, want RFC3339 run start", summary.Date)
	}
	if summary.Metadata.ConfigID != "cfg" || summary.Metadata.JudgeModel != "judge" || summary.Metadata.Description != "desc" || summary.Metadata.PureRecall {
		t.Fatalf("metadata = %#v, want supplied summary metadata", summary.Metadata)
	}
	if got := summary.AbilitySummary["10K"]["RW"]; got.AvgScore != 0.75 || got.Count != 2 {
		t.Fatalf("10K/RW = %#v, want avg .75 count 2", got)
	}
	if got := summary.AbilitySummary["10K"]["OVERALL"]; got.AvgScore != 0.75 || got.Count != 2 {
		t.Fatalf("10K/OVERALL = %#v, want avg .75 count 2", got)
	}
	if got := summary.AbilitySummary["1M"]["ABS"]; got.AvgScore != 0 || got.Count != 1 {
		t.Fatalf("1M/ABS = %#v, want avg 0 count 1", got)
	}
}
