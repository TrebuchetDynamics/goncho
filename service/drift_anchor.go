package goncho

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/limitutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/textmatch"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

type DriftAnchorDetector struct {
	store MemoryToolStore
}

type DriftAnchorCheckParams struct {
	Prompt string `json:"prompt"`
	Limit  int    `json:"limit,omitempty"`
}

type DriftAnchorWarning struct {
	Warn            bool    `json:"warn"`
	Code            string  `json:"code,omitempty"`
	MatchedMemoryID string  `json:"matched_memory_id,omitempty"`
	MatchedContent  string  `json:"matched_content,omitempty"`
	SimilarityScore float64 `json:"similarity_score,omitempty"`
	Recommendation  string  `json:"recommendation,omitempty"`
}

func NewDriftAnchorDetector(store MemoryToolStore) *DriftAnchorDetector {
	return &DriftAnchorDetector{store: store}
}

func (d *DriftAnchorDetector) Check(ctx context.Context, params DriftAnchorCheckParams) (DriftAnchorWarning, error) {
	if d == nil || d.store == nil {
		return DriftAnchorWarning{}, fmt.Errorf("goncho: drift anchor store is required")
	}
	prompt := strings.TrimSpace(params.Prompt)
	if prompt == "" {
		return DriftAnchorWarning{}, fmt.Errorf("goncho: drift anchor prompt is required")
	}
	limit := limitutil.Default(params.Limit, 5)
	entries, err := d.store.Retrieve(ctx, "dead-end", limit)
	if err != nil {
		return DriftAnchorWarning{}, err
	}
	if len(entries) == 0 {
		entries, err = d.store.Retrieve(ctx, "negative", limit)
		if err != nil {
			return DriftAnchorWarning{}, err
		}
	}

	bestScore := 0.0
	var best MemoryToolEntry
	for _, entry := range entries {
		if !isNegativeDriftAnchor(entry) {
			continue
		}
		score := driftAnchorSimilarity(prompt, entry.Content)
		if score > bestScore {
			bestScore = score
			best = entry
		}
	}
	if best.ID == "" || bestScore < 0.30 {
		return DriftAnchorWarning{Warn: false}, nil
	}
	return DriftAnchorWarning{
		Warn:            true,
		Code:            "negative_drift_anchor",
		MatchedMemoryID: best.ID,
		MatchedContent:  best.Content,
		SimilarityScore: bestScore,
		Recommendation:  "verify_live_state_before_repeating_failed_path",
	}, nil
}

func isNegativeDriftAnchor(entry MemoryToolEntry) bool {
	for _, tag := range entry.Tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "negative" || tag == "dead-end" || tag == "drift-anchor" {
			return true
		}
	}
	content := strings.ToLower(entry.Content)
	return strings.Contains(content, "dead end") || strings.Contains(content, "known failure") || strings.Contains(content, "failed path")
}

var driftAnchorTokenPattern = regexp.MustCompile(`[a-z0-9]+`)

func driftAnchorSimilarity(prompt, memory string) float64 {
	return textmatch.OverlapCoefficient(driftAnchorTokenSet(prompt), driftAnchorTokenSet(memory))
}

func driftAnchorTokenSet(value string) map[string]struct{} {
	return textutil.Set(driftAnchorTokenPattern.FindAllString(strings.ToLower(value), -1), func(token string) string {
		if len(token) < 4 || driftAnchorStopword(token) {
			return ""
		}
		return token
	})
}

func driftAnchorStopword(token string) bool {
	switch token {
	case "this", "that", "with", "from", "before", "after", "again", "should", "would", "could", "known":
		return true
	default:
		return false
	}
}
