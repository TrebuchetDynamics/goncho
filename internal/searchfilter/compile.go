package searchfilter

import "github.com/TrebuchetDynamics/goncho/internal/searchfilter/compiler"

func Compile(expr Expression, peer string) (Compiled, error) {
	return compiler.Compile(expr, peer)
}

func ParseAndCompile(raw map[string]any, peer string) (Compiled, error) {
	expr, err := Parse(raw)
	if err != nil {
		return Compiled{}, err
	}
	return Compile(expr, peer)
}

func MergeSources(paramsSources, filterSources []string) (sources []string, denyAll bool) {
	return compiler.MergeSources(paramsSources, filterSources)
}

func HasWildcard(values []string) bool {
	return compiler.HasWildcard(values)
}
