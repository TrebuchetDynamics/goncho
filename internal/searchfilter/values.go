package searchfilter

import (
	"fmt"
	"reflect"
	"strings"
)

func filterValues(value any, op Operator) ([]string, error) {
	if op == OpIn {
		return filterListValues(value)
	}
	return []string{filterScalar(value)}, nil
}

func filterListValues(value any) ([]string, error) {
	items, ok := listElements(value)
	if !ok {
		return nil, fmt.Errorf("in operator value must be a list")
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, filterScalar(item))
	}
	return out, nil
}

func listElements(value any) ([]any, bool) {
	if items, ok := value.([]any); ok {
		return items, true
	}
	if value == nil {
		return nil, false
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out = append(out, rv.Index(i).Interface())
		}
		return out, true
	default:
		return nil, false
	}
}

func filterScalar(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}
