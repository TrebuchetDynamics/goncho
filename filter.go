package goncho

import searchfilter "github.com/TrebuchetDynamics/goncho/internal/searchfilter"

type filterKind = searchfilter.Kind

const (
	filterKindAll        = searchfilter.KindAll
	filterKindAnd        = searchfilter.KindAnd
	filterKindOr         = searchfilter.KindOr
	filterKindNot        = searchfilter.KindNot
	filterKindComparison = searchfilter.KindComparison
)

type filterOperator = searchfilter.Operator

const (
	filterOpEQ        = searchfilter.OpEQ
	filterOpGT        = searchfilter.OpGT
	filterOpGTE       = searchfilter.OpGTE
	filterOpLT        = searchfilter.OpLT
	filterOpLTE       = searchfilter.OpLTE
	filterOpNE        = searchfilter.OpNE
	filterOpIn        = searchfilter.OpIn
	filterOpContains  = searchfilter.OpContains
	filterOpIContains = searchfilter.OpIContains
)

type filterExpression = searchfilter.Expression

type UnsupportedFilterError = searchfilter.UnsupportedFilterError

type compiledSearchFilter = searchfilter.Compiled

func normalizeSearchLimit(limit int) int {
	return searchfilter.NormalizeLimit(limit)
}

func parseSearchFilter(raw map[string]any) (filterExpression, error) {
	return searchfilter.Parse(raw)
}

func flattenComparisons(expr filterExpression) []filterExpression {
	return searchfilter.FlattenComparisons(expr)
}

func compileSearchFilter(expr filterExpression, peer string) (compiledSearchFilter, error) {
	return searchfilter.Compile(expr, peer)
}

func parseAndCompileSearchFilter(raw map[string]any, peer string) (compiledSearchFilter, error) {
	return searchfilter.ParseAndCompile(raw, peer)
}

func mergeSearchSources(paramsSources, filterSources []string) (sources []string, denyAll bool) {
	return searchfilter.MergeSources(paramsSources, filterSources)
}

func filterValuesDenyAll(values []string) bool {
	return searchfilter.ValuesDenyAll(values)
}

func filterHasWildcard(values []string) bool {
	return searchfilter.HasWildcard(values)
}
