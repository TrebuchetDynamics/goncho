package goncho

import (
	"fmt"
	"slices"
	"strings"
)

const (
	defaultSearchLimit = 10
	maxSearchLimit     = 100
)

type filterKind string

const (
	filterKindAll        filterKind = "all"
	filterKindAnd        filterKind = "and"
	filterKindOr         filterKind = "or"
	filterKindNot        filterKind = "not"
	filterKindComparison filterKind = "comparison"
)

type filterOperator string

const (
	filterOpEQ        filterOperator = "eq"
	filterOpGT        filterOperator = "gt"
	filterOpGTE       filterOperator = "gte"
	filterOpLT        filterOperator = "lt"
	filterOpLTE       filterOperator = "lte"
	filterOpNE        filterOperator = "ne"
	filterOpIn        filterOperator = "in"
	filterOpContains  filterOperator = "contains"
	filterOpIContains filterOperator = "icontains"
)

type filterExpression struct {
	Kind     filterKind
	Children []filterExpression
	Field    string
	Operator filterOperator
	Values   []string
}

// UnsupportedFilterError is returned before search when a Honcho-shaped filter
// cannot be enforced by the current Goncho storage model.
type UnsupportedFilterError struct {
	Code     string `json:"code"`
	Field    string `json:"field,omitempty"`
	Operator string `json:"operator,omitempty"`
	Reason   string `json:"reason"`
}

func (e *UnsupportedFilterError) Error() string {
	if e == nil {
		return ""
	}
	parts := []string{"goncho: unsupported_filter"}
	if e.Field != "" {
		parts = append(parts, "field="+e.Field)
	}
	if e.Operator != "" {
		parts = append(parts, "operator="+e.Operator)
	}
	if e.Reason != "" {
		parts = append(parts, e.Reason)
	}
	return strings.Join(parts, ": ")
}

type compiledSearchFilter struct {
	SessionIDs []string
	Sources    []string
	DenyAll    bool
}

func normalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return defaultSearchLimit
	}
	if limit > maxSearchLimit {
		return maxSearchLimit
	}
	return limit
}

func parseSearchFilter(raw map[string]any) (filterExpression, error) {
	if len(raw) == 0 {
		return filterExpression{Kind: filterKindAll}, nil
	}
	return parseFilterMap(raw, nil)
}

func parseFilterMap(raw map[string]any, path []string) (filterExpression, error) {
	if len(raw) == 0 {
		return filterExpression{Kind: filterKindAll}, nil
	}

	children := make([]filterExpression, 0, len(raw))
	for key, value := range raw {
		switch key {
		case "AND", "OR", "NOT":
			child, err := parseLogicalFilter(key, value, path)
			if err != nil {
				return filterExpression{}, err
			}
			children = append(children, child)
		case "metadata":
			child, err := parseMetadataFilter(value)
			if err != nil {
				return filterExpression{}, err
			}
			children = append(children, child)
		default:
			if len(path) == 0 && !isSupportedTopLevelFilterField(key) {
				return filterExpression{}, unsupportedFilter(key, "", "unknown filter field")
			}
			fieldPath := appendPath(path, key)
			child, err := parseFieldCondition(strings.Join(fieldPath, "."), value)
			if err != nil {
				return filterExpression{}, err
			}
			children = append(children, child)
		}
	}
	return collapseImplicitAnd(children), nil
}

func parseLogicalFilter(key string, value any, path []string) (filterExpression, error) {
	items, ok := value.([]any)
	if !ok {
		return filterExpression{}, unsupportedFilter(strings.Join(path, "."), key, "logical filter value must be a list")
	}
	children := make([]filterExpression, 0, len(items))
	for _, item := range items {
		childMap, ok := item.(map[string]any)
		if !ok {
			return filterExpression{}, unsupportedFilter(strings.Join(path, "."), key, "logical filter child must be an object")
		}
		child, err := parseFilterMap(childMap, path)
		if err != nil {
			return filterExpression{}, err
		}
		children = append(children, child)
	}

	switch key {
	case "AND":
		return filterExpression{Kind: filterKindAnd, Children: children}, nil
	case "OR":
		return filterExpression{Kind: filterKindOr, Children: children}, nil
	case "NOT":
		return filterExpression{Kind: filterKindNot, Children: children}, nil
	default:
		return filterExpression{}, unsupportedFilter(strings.Join(path, "."), key, "unknown logical operator")
	}
}

func parseMetadataFilter(value any) (filterExpression, error) {
	raw, ok := value.(map[string]any)
	if !ok {
		return filterExpression{}, unsupportedFilter("metadata", "", "metadata filter must be an object")
	}
	return parseMetadataMap(raw, []string{"metadata"})
}

func parseMetadataMap(raw map[string]any, path []string) (filterExpression, error) {
	children := make([]filterExpression, 0, len(raw))
	for key, value := range raw {
		fieldPath := appendPath(path, key)
		if nested, ok := value.(map[string]any); ok && !isOperatorMap(nested) {
			child, err := parseMetadataMap(nested, fieldPath)
			if err != nil {
				return filterExpression{}, err
			}
			children = append(children, child)
			continue
		}
		child, err := parseFieldCondition(strings.Join(fieldPath, "."), value)
		if err != nil {
			return filterExpression{}, err
		}
		children = append(children, child)
	}
	return collapseImplicitAnd(children), nil
}

func parseFieldCondition(field string, value any) (filterExpression, error) {
	if rawOps, ok := value.(map[string]any); ok {
		children := make([]filterExpression, 0, len(rawOps))
		for rawOp, rawValue := range rawOps {
			op, ok := parseFilterOperator(rawOp)
			if !ok {
				return filterExpression{}, unsupportedFilter(field, rawOp, "unknown filter operator")
			}
			values, err := filterValues(rawValue, op)
			if err != nil {
				return filterExpression{}, unsupportedFilter(field, rawOp, err.Error())
			}
			children = append(children, filterExpression{
				Kind:     filterKindComparison,
				Field:    field,
				Operator: op,
				Values:   values,
			})
		}
		return collapseImplicitAnd(children), nil
	}

	values, err := filterValues(value, filterOpEQ)
	if err != nil {
		return filterExpression{}, unsupportedFilter(field, string(filterOpEQ), err.Error())
	}
	return filterExpression{
		Kind:     filterKindComparison,
		Field:    field,
		Operator: filterOpEQ,
		Values:   values,
	}, nil
}

func parseFilterOperator(op string) (filterOperator, bool) {
	switch op {
	case string(filterOpGT):
		return filterOpGT, true
	case string(filterOpGTE):
		return filterOpGTE, true
	case string(filterOpLT):
		return filterOpLT, true
	case string(filterOpLTE):
		return filterOpLTE, true
	case string(filterOpNE):
		return filterOpNE, true
	case string(filterOpIn):
		return filterOpIn, true
	case string(filterOpContains):
		return filterOpContains, true
	case string(filterOpIContains):
		return filterOpIContains, true
	default:
		return "", false
	}
}

func filterValues(value any, op filterOperator) ([]string, error) {
	if op == filterOpIn {
		items, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("in operator value must be a list")
		}
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, filterScalar(item))
		}
		return out, nil
	}
	return []string{filterScalar(value)}, nil
}

func filterScalar(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func collapseImplicitAnd(children []filterExpression) filterExpression {
	if len(children) == 0 {
		return filterExpression{Kind: filterKindAll}
	}
	if len(children) == 1 {
		return children[0]
	}
	return filterExpression{Kind: filterKindAnd, Children: children}
}

func appendPath(path []string, key string) []string {
	out := make([]string, 0, len(path)+1)
	out = append(out, path...)
	out = append(out, key)
	return out
}

func isSupportedTopLevelFilterField(field string) bool {
	switch field {
	case "session_id", "peer_id", "source", "created_at", "content":
		return true
	default:
		return false
	}
}

func isOperatorMap(raw map[string]any) bool {
	if len(raw) == 0 {
		return false
	}
	for key := range raw {
		if _, ok := parseFilterOperator(key); !ok {
			return false
		}
	}
	return true
}

func unsupportedFilter(field, operator, reason string) *UnsupportedFilterError {
	return &UnsupportedFilterError{
		Code:     "unsupported_filter",
		Field:    strings.Trim(field, "."),
		Operator: operator,
		Reason:   reason,
	}
}

func flattenComparisons(expr filterExpression) []filterExpression {
	if expr.Kind == filterKindComparison {
		return []filterExpression{expr}
	}
	var out []filterExpression
	for _, child := range expr.Children {
		out = append(out, flattenComparisons(child)...)
	}
	return out
}

func compileSearchFilter(expr filterExpression, peer string) (compiledSearchFilter, error) {
	switch expr.Kind {
	case "", filterKindAll:
		return compiledSearchFilter{}, nil
	case filterKindAnd:
		var out compiledSearchFilter
		for _, child := range expr.Children {
			compiled, err := compileSearchFilter(child, peer)
			if err != nil {
				return compiledSearchFilter{}, err
			}
			out = mergeCompiledSearchFilters(out, compiled)
		}
		return out, nil
	case filterKindOr:
		return compiledSearchFilter{}, unsupportedFilter("", "OR", "OR filters are parsed but not enforceable by the current search index")
	case filterKindNot:
		return compiledSearchFilter{}, unsupportedFilter("", "NOT", "NOT filters are parsed but not enforceable by the current search index")
	case filterKindComparison:
		return compileComparisonFilter(expr, peer)
	default:
		return compiledSearchFilter{}, unsupportedFilter(expr.Field, "", "unknown filter expression")
	}
}

func compileComparisonFilter(expr filterExpression, peer string) (compiledSearchFilter, error) {
	switch expr.Field {
	case "session_id":
		if !isEqualityOperator(expr.Operator) {
			return compiledSearchFilter{}, unsupportedFilter(expr.Field, string(expr.Operator), "session_id only supports equality, in, and wildcard filters")
		}
		return compiledSearchFilter{SessionIDs: normalizeFilterValues(expr.Values, false)}, nil
	case "source":
		if !isEqualityOperator(expr.Operator) {
			return compiledSearchFilter{}, unsupportedFilter(expr.Field, string(expr.Operator), "source only supports equality, in, and wildcard filters")
		}
		return compiledSearchFilter{Sources: normalizeFilterValues(expr.Values, true)}, nil
	case "peer_id":
		if !isEqualityOperator(expr.Operator) {
			return compiledSearchFilter{}, unsupportedFilter(expr.Field, string(expr.Operator), "peer_id only supports equality, in, and wildcard filters")
		}
		if peerFilterMatches(expr.Values, peer) {
			return compiledSearchFilter{}, nil
		}
		return compiledSearchFilter{DenyAll: true}, nil
	case "created_at", "content":
		return compiledSearchFilter{}, unsupportedFilter(expr.Field, string(expr.Operator), "field is parsed but not enforceable by the current search index")
	default:
		if strings.HasPrefix(expr.Field, "metadata.") {
			return compiledSearchFilter{}, unsupportedFilter(expr.Field, string(expr.Operator), "metadata filters require a metadata index")
		}
		return compiledSearchFilter{}, unsupportedFilter(expr.Field, string(expr.Operator), "unknown filter field")
	}
}

func isEqualityOperator(op filterOperator) bool {
	return op == filterOpEQ || op == filterOpIn
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

func mergeCompiledSearchFilters(a, b compiledSearchFilter) compiledSearchFilter {
	if a.DenyAll || b.DenyAll {
		return compiledSearchFilter{DenyAll: true}
	}
	return compiledSearchFilter{
		SessionIDs: intersectFilterValues(a.SessionIDs, b.SessionIDs),
		Sources:    intersectFilterValues(a.Sources, b.Sources),
	}
}

func intersectFilterValues(a, b []string) []string {
	if len(a) == 0 {
		return append([]string(nil), b...)
	}
	if len(b) == 0 {
		return append([]string(nil), a...)
	}
	if slices.Contains(a, "*") {
		return append([]string(nil), b...)
	}
	if slices.Contains(b, "*") {
		return append([]string(nil), a...)
	}
	out := make([]string, 0, min(len(a), len(b)))
	for _, left := range a {
		if slices.Contains(b, left) && !slices.Contains(out, left) {
			out = append(out, left)
		}
	}
	if len(out) == 0 {
		return []string{"__deny_all__"}
	}
	return out
}

func parseAndCompileSearchFilter(raw map[string]any, peer string) (compiledSearchFilter, error) {
	expr, err := parseSearchFilter(raw)
	if err != nil {
		return compiledSearchFilter{}, err
	}
	return compileSearchFilter(expr, peer)
}

func mergeSearchSources(paramsSources, filterSources []string) (sources []string, denyAll bool) {
	merged := intersectFilterValues(normalizeFilterValues(paramsSources, true), normalizeFilterValues(filterSources, true))
	if len(merged) == 1 && merged[0] == "__deny_all__" {
		return nil, true
	}
	if slices.Contains(merged, "*") {
		return nil, false
	}
	return merged, false
}

func filterValuesDenyAll(values []string) bool {
	return len(values) == 1 && values[0] == "__deny_all__"
}

func filterHasWildcard(values []string) bool {
	return slices.Contains(values, "*")
}
