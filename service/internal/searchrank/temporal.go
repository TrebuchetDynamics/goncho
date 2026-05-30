package searchrank

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

type TemporalDirection int

const (
	TemporalNone TemporalDirection = iota
	TemporalNewer
	TemporalOlder
)

type TemporalFeatures struct {
	Direction TemporalDirection
	Markers   []string
	Temporal  bool
}

func TemporalIntent(query string) TemporalFeatures {
	q := strings.ToLower(query)
	features := TemporalFeatures{Markers: TemporalMarkers(q), Temporal: TemporalQuery(q)}
	if textutil.ContainsAnySubstring(q, []string{"first", "earliest", "initial", "original", "started first"}) {
		features.Direction = TemporalOlder
		return features
	}
	if textutil.ContainsAnySubstring(q, []string{"latest", "current", "currently", "most recently"}) {
		features.Direction = TemporalNewer
		return features
	}
	return features
}

func TemporalQuery(query string) bool {
	needles := []string{"when", "first", "earliest", "initial", "original", "latest", "current", "currently", "recent", "today", "yesterday", "tomorrow", "last ", "this ", "past ", "how many days", "how many weeks", "how many months", "how many years", "how long", "order of"}
	return textutil.ContainsAnySubstring(query, needles)
}

func TemporalMarkers(query string) []string {
	candidates := []string{
		"today", "yesterday", "tomorrow", "most recently", "this weekend", "this week", "this month", "this year", "past few months", "past three months", "last week", "last month", "last year", "last friday", "last saturday", "last sunday",
		"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december",
	}
	markers := []string{}
	for _, candidate := range candidates {
		if strings.Contains(query, candidate) {
			markers = append(markers, candidate)
		}
	}
	return markers
}

func TemporalRerankBonus(features TemporalFeatures, content string, index, total int, score, maxScore float64) float64 {
	if total < 2 || maxScore <= 0 {
		return 0
	}
	if features.Temporal && GenericAssistantAnswer(content) && PersonalSignalCount(content) < 12 && score >= maxScore*0.70 {
		return -maxScore * 0.30
	}
	if features.Direction == TemporalNone || score < maxScore*0.78 {
		return 0
	}
	contentLower := strings.ToLower(content)
	markerMatches := 0
	for _, marker := range features.Markers {
		if strings.Contains(contentLower, marker) {
			markerMatches++
		}
	}
	alignment := float64(markerMatches)
	switch features.Direction {
	case TemporalNewer:
		// Newer/current phrasing is common in distractors (for example "new products"),
		// so only exact query temporal marker matches contribute positive evidence.
	case TemporalOlder:
		if textutil.ContainsAnySubstring(contentLower, []string{"first", "initial", "original", "earliest", "started", "began"}) {
			alignment += 0.5
		}
	}
	if alignment == 0 {
		return 0
	}
	position := 0.0
	if total > 1 {
		position = float64(total-1-index) / float64(total-1)
		if features.Direction == TemporalOlder {
			position = float64(index) / float64(total-1)
		}
	}
	return maxScore * 0.08 * alignment * (0.5 + 0.5*position)
}
