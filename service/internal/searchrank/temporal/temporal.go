package temporal

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/searchrank/signals"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

type Direction int

const (
	DirectionNone Direction = iota
	DirectionNewer
	DirectionOlder
)

type Features struct {
	Direction Direction
	Markers   []string
	Temporal  bool
}

func Intent(query string) Features {
	q := normalizeQuery(query)
	features := Features{Markers: Markers(q), Temporal: Query(q)}
	if textutil.ContainsAnySubstring(q, []string{"first", "earliest", "initial", "original", "started first"}) {
		features.Direction = DirectionOlder
		return features
	}
	if textutil.ContainsAnySubstring(q, []string{"latest", "current", "currently", "most recently"}) {
		features.Direction = DirectionNewer
		return features
	}
	return features
}

func Query(query string) bool {
	needles := []string{"when", "first", "earliest", "initial", "original", "latest", "current", "currently", "recent", "today", "yesterday", "tomorrow", "last ", "this ", "past ", "how many days", "how many weeks", "how many months", "how many years", "how long", "order of"}
	return textutil.ContainsAnySubstring(normalizeQuery(query), needles)
}

func Markers(query string) []string {
	q := normalizeQuery(query)
	markers := []string{}
	for _, candidate := range markerCandidates() {
		if containsMarker(q, candidate) {
			markers = append(markers, candidate)
		}
	}
	return markers
}

func containsMarker(query, marker string) bool {
	if marker == "" {
		return false
	}
	for offset := 0; offset <= len(query); {
		idx := strings.Index(query[offset:], marker)
		if idx < 0 {
			return false
		}
		start := offset + idx
		end := start + len(marker)
		if markerBoundary(query, start-1) && markerBoundary(query, end) {
			return true
		}
		offset = start + 1
	}
	return false
}

func markerBoundary(value string, index int) bool {
	if index < 0 || index >= len(value) {
		return true
	}
	ch := value[index]
	return !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9'))
}

func normalizeQuery(query string) string {
	return strings.ToLower(query)
}

func markerCandidates() []string {
	return []string{
		"today", "yesterday", "tomorrow", "most recently", "this weekend", "this week", "this month", "this year", "past few months", "past three months", "last week", "last month", "last year", "last friday", "last saturday", "last sunday",
		"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december",
	}
}

func RerankBonus(features Features, content string, index, total int, score, maxScore float64) float64 {
	if total < 2 || maxScore <= 0 {
		return 0
	}
	if features.Temporal && signals.GenericAssistantAnswer(content) && signals.PersonalSignalCount(content) < 12 && score >= maxScore*0.70 {
		return -maxScore * 0.30
	}
	if features.Direction == DirectionNone || score < maxScore*0.78 {
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
	case DirectionNewer:
		// Newer/current phrasing is common in distractors (for example "new products"),
		// so only exact query temporal marker matches contribute positive evidence.
	case DirectionOlder:
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
		if features.Direction == DirectionOlder {
			position = float64(index) / float64(total-1)
		}
	}
	return maxScore * 0.08 * alignment * (0.5 + 0.5*position)
}
