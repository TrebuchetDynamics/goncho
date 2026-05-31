package compiler

import (
	"slices"
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/searchfilter/contracts"
	searchvalues "github.com/TrebuchetDynamics/goncho/internal/searchfilter/values"
)

func Compile(expr contracts.Expression, peer string) (contracts.Compiled, error) {
	switch expr.Kind {
	case "", contracts.KindAll:
		return contracts.Compiled{}, nil
	case contracts.KindAnd:
		var out contracts.Compiled
		for _, child := range expr.Children {
			compiled, err := Compile(child, peer)
			if err != nil {
				return contracts.Compiled{}, err
			}
			out = mergeCompiledSearchFilters(out, compiled)
		}
		return out, nil
	case contracts.KindOr:
		return contracts.Compiled{}, contracts.NewUnsupportedFilter("", "OR", "OR filters are parsed but not enforceable by the current search index")
	case contracts.KindNot:
		return contracts.Compiled{}, contracts.NewUnsupportedFilter("", "NOT", "NOT filters are parsed but not enforceable by the current search index")
	case contracts.KindComparison:
		return compileComparisonFilter(expr, peer)
	default:
		return contracts.Compiled{}, contracts.NewUnsupportedFilter(expr.Field, "", "unknown filter expression")
	}
}

func compileComparisonFilter(expr contracts.Expression, peer string) (contracts.Compiled, error) {
	switch expr.Field {
	case "session_id":
		valueSet, err := compileEqualityValueSet(expr, false)
		if err != nil {
			return contracts.Compiled{}, err
		}
		if valueSet.DenyAll {
			return contracts.Compiled{DenyAll: true}, nil
		}
		return contracts.Compiled{SessionIDs: valueSet.Values}, nil
	case "source":
		valueSet, err := compileEqualityValueSet(expr, true)
		if err != nil {
			return contracts.Compiled{}, err
		}
		if valueSet.DenyAll {
			return contracts.Compiled{DenyAll: true}, nil
		}
		return contracts.Compiled{Sources: valueSet.Values}, nil
	case "peer_id":
		valueSet, err := compileEqualityValueSet(expr, false)
		if err != nil {
			return contracts.Compiled{}, err
		}
		if valueSet.DenyAll {
			return contracts.Compiled{DenyAll: true}, nil
		}
		if peerFilterMatches(valueSet.Values, peer) {
			return contracts.Compiled{}, nil
		}
		return contracts.Compiled{DenyAll: true}, nil
	case "created_at", "content":
		return contracts.Compiled{}, contracts.NewUnsupportedFilter(expr.Field, string(expr.Operator), "field is parsed but not enforceable by the current search index")
	default:
		if strings.HasPrefix(expr.Field, "metadata.") {
			return contracts.Compiled{}, contracts.NewUnsupportedFilter(expr.Field, string(expr.Operator), "metadata filters require a metadata index")
		}
		return contracts.Compiled{}, contracts.NewUnsupportedFilter(expr.Field, string(expr.Operator), "unknown filter field")
	}
}

type equalityValueSet struct {
	Values  []string
	DenyAll bool
}

func compileEqualityValueSet(expr contracts.Expression, lower bool) (equalityValueSet, error) {
	values, err := compileEqualityFilterValues(expr, lower)
	if err != nil {
		return equalityValueSet{}, err
	}
	return equalityValueSet{Values: values, DenyAll: len(values) == 0}, nil
}

func compileEqualityFilterValues(expr contracts.Expression, lower bool) ([]string, error) {
	if !isEqualityOperator(expr.Operator) {
		return nil, contracts.NewUnsupportedFilter(expr.Field, string(expr.Operator), expr.Field+" only supports equality, in, and wildcard filters")
	}
	return searchvalues.NormalizeEqualityValues(expr.Values, lower), nil
}

func isEqualityOperator(op contracts.Operator) bool {
	return op == contracts.OpEQ || op == contracts.OpIn
}

func peerFilterMatches(values []string, peer string) bool {
	peer = strings.TrimSpace(peer)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "*" || value == peer {
			return true
		}
	}
	return false
}

func mergeCompiledSearchFilters(a, b contracts.Compiled) contracts.Compiled {
	if a.DenyAll || b.DenyAll {
		return contracts.Compiled{DenyAll: true}
	}
	sessionIDs, denySessions := intersectFilterValues(a.SessionIDs, b.SessionIDs)
	sources, denySources := intersectFilterValues(a.Sources, b.Sources)
	if denySessions || denySources {
		return contracts.Compiled{DenyAll: true}
	}
	return contracts.Compiled{
		SessionIDs: sessionIDs,
		Sources:    sources,
	}
}

func intersectFilterValues(a, b []string) ([]string, bool) {
	if len(a) == 0 {
		return append([]string(nil), b...), false
	}
	if len(b) == 0 {
		return append([]string(nil), a...), false
	}
	if slices.Contains(a, "*") {
		return append([]string(nil), b...), false
	}
	if slices.Contains(b, "*") {
		return append([]string(nil), a...), false
	}
	out := make([]string, 0, min(len(a), len(b)))
	for _, left := range a {
		if slices.Contains(b, left) && !slices.Contains(out, left) {
			out = append(out, left)
		}
	}
	return out, len(out) == 0
}

func MergeSources(paramsSources, filterSources []string) (sources []string, denyAll bool) {
	merged, denyAll := intersectFilterValues(searchvalues.NormalizeEqualityValues(paramsSources, true), searchvalues.NormalizeEqualityValues(filterSources, true))
	if denyAll {
		return nil, true
	}
	if slices.Contains(merged, "*") {
		return nil, false
	}
	return merged, false
}

func HasWildcard(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "*" {
			return true
		}
	}
	return false
}
