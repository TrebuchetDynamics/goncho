package grammar

import (
	"slices"
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/searchfilter/contracts"
	searchvalues "github.com/TrebuchetDynamics/goncho/internal/searchfilter/values"
)

func Parse(raw map[string]any) (contracts.Expression, error) {
	if len(raw) == 0 {
		return contracts.Expression{Kind: contracts.KindAll}, nil
	}
	return parseFilterMap(raw, nil)
}

func parseFilterMap(raw map[string]any, path []string) (contracts.Expression, error) {
	if len(raw) == 0 {
		return contracts.Expression{Kind: contracts.KindAll}, nil
	}

	children := make([]contracts.Expression, 0, len(raw))
	for _, key := range sortedMapKeys(raw) {
		value := raw[key]
		switch key {
		case "AND", "OR", "NOT":
			child, err := parseLogicalFilter(key, value, path)
			if err != nil {
				return contracts.Expression{}, err
			}
			children = append(children, child)
		case "metadata":
			child, err := parseMetadataFilter(value)
			if err != nil {
				return contracts.Expression{}, err
			}
			children = append(children, child)
		default:
			if len(path) == 0 && !isSupportedTopLevelFilterField(key) {
				return contracts.Expression{}, contracts.NewUnsupportedFilter(key, "", "unknown filter field")
			}
			fieldPath := appendPath(path, key)
			child, err := parseFieldCondition(strings.Join(fieldPath, "."), value)
			if err != nil {
				return contracts.Expression{}, err
			}
			children = append(children, child)
		}
	}
	return collapseImplicitAnd(children), nil
}

func parseLogicalFilter(key string, value any, path []string) (contracts.Expression, error) {
	items, ok := searchvalues.Elements(value)
	if !ok {
		return contracts.Expression{}, contracts.NewUnsupportedFilter(strings.Join(path, "."), key, "logical filter value must be a list")
	}
	children := make([]contracts.Expression, 0, len(items))
	for _, item := range items {
		childMap, ok := item.(map[string]any)
		if !ok {
			return contracts.Expression{}, contracts.NewUnsupportedFilter(strings.Join(path, "."), key, "logical filter child must be an object")
		}
		child, err := parseFilterMap(childMap, path)
		if err != nil {
			return contracts.Expression{}, err
		}
		children = append(children, child)
	}

	switch key {
	case "AND":
		return contracts.Expression{Kind: contracts.KindAnd, Children: children}, nil
	case "OR":
		return contracts.Expression{Kind: contracts.KindOr, Children: children}, nil
	case "NOT":
		return contracts.Expression{Kind: contracts.KindNot, Children: children}, nil
	default:
		return contracts.Expression{}, contracts.NewUnsupportedFilter(strings.Join(path, "."), key, "unknown logical operator")
	}
}

func parseMetadataFilter(value any) (contracts.Expression, error) {
	raw, ok := value.(map[string]any)
	if !ok {
		return contracts.Expression{}, contracts.NewUnsupportedFilter("metadata", "", "metadata filter must be an object")
	}
	return parseMetadataMap(raw, []string{"metadata"})
}

func parseMetadataMap(raw map[string]any, path []string) (contracts.Expression, error) {
	children := make([]contracts.Expression, 0, len(raw))
	for _, key := range sortedMapKeys(raw) {
		value := raw[key]
		fieldPath := appendPath(path, key)
		if nested, ok := value.(map[string]any); ok {
			classification := classifyOperatorMap(nested)
			if classification.Empty {
				return contracts.Expression{}, contracts.NewUnsupportedFilter(strings.Join(fieldPath, "."), "", "operator map must not be empty")
			}
			if classification.Mixed {
				return contracts.Expression{}, contracts.NewUnsupportedFilter(strings.Join(fieldPath, "."), classification.UnknownOperator, "unknown filter operator")
			}
			if !classification.Valid {
				child, err := parseMetadataMap(nested, fieldPath)
				if err != nil {
					return contracts.Expression{}, err
				}
				children = append(children, child)
				continue
			}
		}
		child, err := parseFieldCondition(strings.Join(fieldPath, "."), value)
		if err != nil {
			return contracts.Expression{}, err
		}
		children = append(children, child)
	}
	return collapseImplicitAnd(children), nil
}

func parseFieldCondition(field string, value any) (contracts.Expression, error) {
	if rawOps, ok := value.(map[string]any); ok {
		return parseOperatorConditions(field, rawOps)
	}

	values, err := searchvalues.FilterValues(value, contracts.OpEQ)
	if err != nil {
		return contracts.Expression{}, contracts.NewUnsupportedFilter(field, string(contracts.OpEQ), err.Error())
	}
	return contracts.Expression{
		Kind:     contracts.KindComparison,
		Field:    field,
		Operator: contracts.OpEQ,
		Values:   values,
	}, nil
}

var supportedFilterOperators = map[string]contracts.Operator{
	string(contracts.OpEQ):        contracts.OpEQ,
	string(contracts.OpGT):        contracts.OpGT,
	string(contracts.OpGTE):       contracts.OpGTE,
	string(contracts.OpLT):        contracts.OpLT,
	string(contracts.OpLTE):       contracts.OpLTE,
	string(contracts.OpNE):        contracts.OpNE,
	string(contracts.OpIn):        contracts.OpIn,
	string(contracts.OpContains):  contracts.OpContains,
	string(contracts.OpIContains): contracts.OpIContains,
}

func parseOperatorConditions(field string, rawOps map[string]any) (contracts.Expression, error) {
	if len(rawOps) == 0 {
		return contracts.Expression{}, contracts.NewUnsupportedFilter(field, "", "operator map must not be empty")
	}
	children := make([]contracts.Expression, 0, len(rawOps))
	for _, rawOp := range sortedMapKeys(rawOps) {
		rawValue := rawOps[rawOp]
		op, ok := parseFilterOperator(rawOp)
		if !ok {
			return contracts.Expression{}, contracts.NewUnsupportedFilter(field, rawOp, "unknown filter operator")
		}
		values, err := searchvalues.FilterValues(rawValue, op)
		if err != nil {
			return contracts.Expression{}, contracts.NewUnsupportedFilter(field, rawOp, err.Error())
		}
		children = append(children, contracts.Expression{
			Kind:     contracts.KindComparison,
			Field:    field,
			Operator: op,
			Values:   values,
		})
	}
	return collapseImplicitAnd(children), nil
}

func parseFilterOperator(op string) (contracts.Operator, bool) {
	parsed, ok := supportedFilterOperators[op]
	return parsed, ok
}

func collapseImplicitAnd(children []contracts.Expression) contracts.Expression {
	if len(children) == 0 {
		return contracts.Expression{Kind: contracts.KindAll}
	}
	if len(children) == 1 {
		return children[0]
	}
	return contracts.Expression{Kind: contracts.KindAnd, Children: children}
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

func FlattenComparisons(expr contracts.Expression) []contracts.Expression {
	if expr.Kind == contracts.KindComparison {
		return []contracts.Expression{expr}
	}
	var out []contracts.Expression
	for _, child := range expr.Children {
		out = append(out, FlattenComparisons(child)...)
	}
	return out
}
