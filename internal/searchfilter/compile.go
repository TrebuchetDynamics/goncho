package searchfilter

import (
	"slices"
	"strings"
)

func Compile(expr Expression, peer string) (Compiled, error) {
	switch expr.Kind {
	case "", KindAll:
		return Compiled{}, nil
	case KindAnd:
		var out Compiled
		for _, child := range expr.Children {
			compiled, err := Compile(child, peer)
			if err != nil {
				return Compiled{}, err
			}
			out = mergeCompiledSearchFilters(out, compiled)
		}
		return out, nil
	case KindOr:
		return Compiled{}, unsupportedFilter("", "OR", "OR filters are parsed but not enforceable by the current search index")
	case KindNot:
		return Compiled{}, unsupportedFilter("", "NOT", "NOT filters are parsed but not enforceable by the current search index")
	case KindComparison:
		return compileComparisonFilter(expr, peer)
	default:
		return Compiled{}, unsupportedFilter(expr.Field, "", "unknown filter expression")
	}
}

func compileComparisonFilter(expr Expression, peer string) (Compiled, error) {
	switch expr.Field {
	case "session_id":
		valueSet, err := compileEqualityValueSet(expr, false)
		if err != nil {
			return Compiled{}, err
		}
		if valueSet.DenyAll {
			return Compiled{DenyAll: true}, nil
		}
		return Compiled{SessionIDs: valueSet.Values}, nil
	case "source":
		valueSet, err := compileEqualityValueSet(expr, true)
		if err != nil {
			return Compiled{}, err
		}
		if valueSet.DenyAll {
			return Compiled{DenyAll: true}, nil
		}
		return Compiled{Sources: valueSet.Values}, nil
	case "peer_id":
		valueSet, err := compileEqualityValueSet(expr, false)
		if err != nil {
			return Compiled{}, err
		}
		if valueSet.DenyAll {
			return Compiled{DenyAll: true}, nil
		}
		if peerFilterMatches(valueSet.Values, peer) {
			return Compiled{}, nil
		}
		return Compiled{DenyAll: true}, nil
	case "created_at", "content":
		return Compiled{}, unsupportedFilter(expr.Field, string(expr.Operator), "field is parsed but not enforceable by the current search index")
	default:
		if strings.HasPrefix(expr.Field, "metadata.") {
			return Compiled{}, unsupportedFilter(expr.Field, string(expr.Operator), "metadata filters require a metadata index")
		}
		return Compiled{}, unsupportedFilter(expr.Field, string(expr.Operator), "unknown filter field")
	}
}

type equalityValueSet struct {
	Values  []string
	DenyAll bool
}

func compileEqualityValueSet(expr Expression, lower bool) (equalityValueSet, error) {
	values, err := compileEqualityFilterValues(expr, lower)
	if err != nil {
		return equalityValueSet{}, err
	}
	return equalityValueSet{Values: values, DenyAll: len(values) == 0}, nil
}

func compileEqualityFilterValues(expr Expression, lower bool) ([]string, error) {
	if !isEqualityOperator(expr.Operator) {
		return nil, unsupportedFilter(expr.Field, string(expr.Operator), expr.Field+" only supports equality, in, and wildcard filters")
	}
	return normalizeFilterValues(expr.Values, lower), nil
}

func isEqualityOperator(op Operator) bool {
	return op == OpEQ || op == OpIn
}

func normalizeFilterValues(values []string, lower bool) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if lower {
			value = strings.ToLower(value)
		}
		if value == "" || slices.Contains(out, value) {
			continue
		}
		out = append(out, value)
	}
	return out
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

func mergeCompiledSearchFilters(a, b Compiled) Compiled {
	if a.DenyAll || b.DenyAll {
		return Compiled{DenyAll: true}
	}
	sessionIDs, denySessions := intersectFilterValues(a.SessionIDs, b.SessionIDs)
	sources, denySources := intersectFilterValues(a.Sources, b.Sources)
	if denySessions || denySources {
		return Compiled{DenyAll: true}
	}
	return Compiled{
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

func ParseAndCompile(raw map[string]any, peer string) (Compiled, error) {
	expr, err := Parse(raw)
	if err != nil {
		return Compiled{}, err
	}
	return Compile(expr, peer)
}

func MergeSources(paramsSources, filterSources []string) (sources []string, denyAll bool) {
	merged, denyAll := intersectFilterValues(normalizeFilterValues(paramsSources, true), normalizeFilterValues(filterSources, true))
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
