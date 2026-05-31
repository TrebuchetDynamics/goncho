package searchfilter

import "github.com/TrebuchetDynamics/goncho/internal/searchfilter/grammar"

func Parse(raw map[string]any) (Expression, error) {
	return grammar.Parse(raw)
}

func FlattenComparisons(expr Expression) []Expression {
	return grammar.FlattenComparisons(expr)
}
