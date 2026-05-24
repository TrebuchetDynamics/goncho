package searchfilter

import (
	"fmt"
	"slices"
	"strings"
)

const (
	defaultSearchLimit = 10
	maxSearchLimit     = 100
)

type Kind string

const (
	KindAll        Kind = "all"
	KindAnd        Kind = "and"
	KindOr         Kind = "or"
	KindNot        Kind = "not"
	KindComparison Kind = "comparison"
)

type Operator string

const (
	OpEQ        Operator = "eq"
	OpGT        Operator = "gt"
	OpGTE       Operator = "gte"
	OpLT        Operator = "lt"
	OpLTE       Operator = "lte"
	OpNE        Operator = "ne"
	OpIn        Operator = "in"
	OpContains  Operator = "contains"
	OpIContains Operator = "icontains"
)

type Expression struct {
	Kind     Kind
	Children []Expression
	Field    string
	Operator Operator
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

type Compiled struct {
	SessionIDs []string
	Sources    []string
	DenyAll    bool
}

func NormalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultSearchLimit
	}
	if limit > maxSearchLimit {
		return maxSearchLimit
	}
	return limit
}

func Parse(raw map[string]any) (Expression, error) {
	if len(raw) == 0 {
		return Expression{Kind: KindAll}, nil
	}
	return parseFilterMap(raw, nil)
}

func parseFilterMap(raw map[string]any, path []string) (Expression, error) {
	if len(raw) == 0 {
		return Expression{Kind: KindAll}, nil
	}

	children := make([]Expression, 0, len(raw))
	for key, value := range raw {
		switch key {
		case "AND", "OR", "NOT":
			child, err := parseLogicalFilter(key, value, path)
			if err != nil {
				return Expression{}, err
			}
			children = append(children, child)
		case "metadata":
			child, err := parseMetadataFilter(value)
			if err != nil {
				return Expression{}, err
			}
			children = append(children, child)
		default:
			if len(path) == 0 && !isSupportedTopLevelFilterField(key) {
				return Expression{}, unsupportedFilter(key, "", "unknown filter field")
			}
			fieldPath := appendPath(path, key)
			child, err := parseFieldCondition(strings.Join(fieldPath, "."), value)
			if err != nil {
				return Expression{}, err
			}
			children = append(children, child)
		}
	}
	return collapseImplicitAnd(children), nil
}

func parseLogicalFilter(key string, value any, path []string) (Expression, error) {
	items, ok := value.([]any)
	if !ok {
		return Expression{}, unsupportedFilter(strings.Join(path, "."), key, "logical filter value must be a list")
	}
	children := make([]Expression, 0, len(items))
	for _, item := range items {
		childMap, ok := item.(map[string]any)
		if !ok {
			return Expression{}, unsupportedFilter(strings.Join(path, "."), key, "logical filter child must be an object")
		}
		child, err := parseFilterMap(childMap, path)
		if err != nil {
			return Expression{}, err
		}
		children = append(children, child)
	}

	switch key {
	case "AND":
		return Expression{Kind: KindAnd, Children: children}, nil
	case "OR":
		return Expression{Kind: KindOr, Children: children}, nil
	case "NOT":
		return Expression{Kind: KindNot, Children: children}, nil
	default:
		return Expression{}, unsupportedFilter(strings.Join(path, "."), key, "unknown logical operator")
	}
}

func parseMetadataFilter(value any) (Expression, error) {
	raw, ok := value.(map[string]any)
	if !ok {
		return Expression{}, unsupportedFilter("metadata", "", "metadata filter must be an object")
	}
	return parseMetadataMap(raw, []string{"metadata"})
}

func parseMetadataMap(raw map[string]any, path []string) (Expression, error) {
	children := make([]Expression, 0, len(raw))
	for key, value := range raw {
		fieldPath := appendPath(path, key)
		if nested, ok := value.(map[string]any); ok && !isOperatorMap(nested) {
			child, err := parseMetadataMap(nested, fieldPath)
			if err != nil {
				return Expression{}, err
			}
			children = append(children, child)
			continue
		}
		child, err := parseFieldCondition(strings.Join(fieldPath, "."), value)
		if err != nil {
			return Expression{}, err
		}
		children = append(children, child)
	}
	return collapseImplicitAnd(children), nil
}

func parseFieldCondition(field string, value any) (Expression, error) {
	if rawOps, ok := value.(map[string]any); ok {
		children := make([]Expression, 0, len(rawOps))
		for rawOp, rawValue := range rawOps {
			op, ok := parseFilterOperator(rawOp)
			if !ok {
				return Expression{}, unsupportedFilter(field, rawOp, "unknown filter operator")
			}
			values, err := filterValues(rawValue, op)
			if err != nil {
				return Expression{}, unsupportedFilter(field, rawOp, err.Error())
			}
			children = append(children, Expression{
				Kind:     KindComparison,
				Field:    field,
				Operator: op,
				Values:   values,
			})
		}
		return collapseImplicitAnd(children), nil
	}

	values, err := filterValues(value, OpEQ)
	if err != nil {
		return Expression{}, unsupportedFilter(field, string(OpEQ), err.Error())
	}
	return Expression{
		Kind:     KindComparison,
		Field:    field,
		Operator: OpEQ,
		Values:   values,
	}, nil
}

func parseFilterOperator(op string) (Operator, bool) {
	switch op {
	case string(OpGT):
		return OpGT, true
	case string(OpGTE):
		return OpGTE, true
	case string(OpLT):
		return OpLT, true
	case string(OpLTE):
		return OpLTE, true
	case string(OpNE):
		return OpNE, true
	case string(OpIn):
		return OpIn, true
	case string(OpContains):
		return OpContains, true
	case string(OpIContains):
		return OpIContains, true
	default:
		return "", false
	}
}

func filterValues(value any, op Operator) ([]string, error) {
	if op == OpIn {
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

func collapseImplicitAnd(children []Expression) Expression {
	if len(children) == 0 {
		return Expression{Kind: KindAll}
	}
	if len(children) == 1 {
		return children[0]
	}
	return Expression{Kind: KindAnd, Children: children}
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

func FlattenComparisons(expr Expression) []Expression {
	if expr.Kind == KindComparison {
		return []Expression{expr}
	}
	var out []Expression
	for _, child := range expr.Children {
		out = append(out, FlattenComparisons(child)...)
	}
	return out
}

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
		if !isEqualityOperator(expr.Operator) {
			return Compiled{}, unsupportedFilter(expr.Field, string(expr.Operator), "session_id only supports equality, in, and wildcard filters")
		}
		return Compiled{SessionIDs: normalizeFilterValues(expr.Values, false)}, nil
	case "source":
		if !isEqualityOperator(expr.Operator) {
			return Compiled{}, unsupportedFilter(expr.Field, string(expr.Operator), "source only supports equality, in, and wildcard filters")
		}
		return Compiled{Sources: normalizeFilterValues(expr.Values, true)}, nil
	case "peer_id":
		if !isEqualityOperator(expr.Operator) {
			return Compiled{}, unsupportedFilter(expr.Field, string(expr.Operator), "peer_id only supports equality, in, and wildcard filters")
		}
		if peerFilterMatches(expr.Values, peer) {
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
	return Compiled{
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

func ParseAndCompile(raw map[string]any, peer string) (Compiled, error) {
	expr, err := Parse(raw)
	if err != nil {
		return Compiled{}, err
	}
	return Compile(expr, peer)
}

func MergeSources(paramsSources, filterSources []string) (sources []string, denyAll bool) {
	merged := intersectFilterValues(normalizeFilterValues(paramsSources, true), normalizeFilterValues(filterSources, true))
	if len(merged) == 1 && merged[0] == "__deny_all__" {
		return nil, true
	}
	if slices.Contains(merged, "*") {
		return nil, false
	}
	return merged, false
}

func ValuesDenyAll(values []string) bool {
	return len(values) == 1 && values[0] == "__deny_all__"
}

func HasWildcard(values []string) bool {
	return slices.Contains(values, "*")
}
