package searchfilter

import (
	"slices"
	"strings"
)

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
	for _, key := range sortedMapKeys(raw) {
		value := raw[key]
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
	items, ok := listElements(value)
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
	for _, key := range sortedMapKeys(raw) {
		value := raw[key]
		fieldPath := appendPath(path, key)
		if nested, ok := value.(map[string]any); ok {
			classification := classifyOperatorMap(nested)
			if classification.Empty {
				return Expression{}, unsupportedFilter(strings.Join(fieldPath, "."), "", "operator map must not be empty")
			}
			if classification.Mixed {
				return Expression{}, unsupportedFilter(strings.Join(fieldPath, "."), classification.UnknownOperator, "unknown filter operator")
			}
			if !classification.Valid {
				child, err := parseMetadataMap(nested, fieldPath)
				if err != nil {
					return Expression{}, err
				}
				children = append(children, child)
				continue
			}
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
		return parseOperatorConditions(field, rawOps)
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

var supportedFilterOperators = map[string]Operator{
	string(OpEQ):        OpEQ,
	string(OpGT):        OpGT,
	string(OpGTE):       OpGTE,
	string(OpLT):        OpLT,
	string(OpLTE):       OpLTE,
	string(OpNE):        OpNE,
	string(OpIn):        OpIn,
	string(OpContains):  OpContains,
	string(OpIContains): OpIContains,
}

func parseOperatorConditions(field string, rawOps map[string]any) (Expression, error) {
	if len(rawOps) == 0 {
		return Expression{}, unsupportedFilter(field, "", "operator map must not be empty")
	}
	children := make([]Expression, 0, len(rawOps))
	for _, rawOp := range sortedMapKeys(rawOps) {
		rawValue := rawOps[rawOp]
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

func parseFilterOperator(op string) (Operator, bool) {
	parsed, ok := supportedFilterOperators[op]
	return parsed, ok
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

type operatorMapClassification struct {
	Empty           bool
	Valid           bool
	Mixed           bool
	UnknownOperator string
}

func isOperatorMap(raw map[string]any) bool {
	return classifyOperatorMap(raw).Valid
}

func classifyOperatorMap(raw map[string]any) operatorMapClassification {
	if len(raw) == 0 {
		return operatorMapClassification{Empty: true}
	}

	knownOperators := 0
	unknownOperator := ""
	for _, key := range sortedMapKeys(raw) {
		if _, ok := parseFilterOperator(key); ok {
			knownOperators++
			continue
		}
		if unknownOperator == "" {
			unknownOperator = key
		}
	}
	if knownOperators == len(raw) {
		return operatorMapClassification{Valid: true}
	}
	if knownOperators > 0 {
		return operatorMapClassification{Mixed: true, UnknownOperator: unknownOperator}
	}
	return operatorMapClassification{}
}

func sortedMapKeys(raw map[string]any) []string {
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
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
