package recalldiag

import (
	"fmt"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

type ScoreBreakdown struct {
	KeywordScore     float64
	SemanticScore    float64
	GraphScore       float64
	FactScore        float64
	RecencyScore     float64
	ImportanceScore  float64
	ScopeScore       float64
	RRFScore         float64
	DiversityPenalty float64
}

func FormatScores(scores ScoreBreakdown) string {
	return fmt.Sprintf("scores: keyword=%.6f semantic=%.6f graph=%.6f fact=%.6f recency=%.6f importance=%.6f scope=%.6f rrf=%.6f diversity_penalty=%.6f",
		scores.KeywordScore,
		scores.SemanticScore,
		scores.GraphScore,
		scores.FactScore,
		scores.RecencyScore,
		scores.ImportanceScore,
		scores.ScopeScore,
		scores.RRFScore,
		scores.DiversityPenalty,
	)
}

func PreviewContent(content string) string {
	content = textutil.CollapseWhitespace(content)
	if len(content) <= 96 {
		return content
	}
	return content[:93] + "..."
}
