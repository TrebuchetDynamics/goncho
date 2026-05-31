package values

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/searchfilter/contracts"
)

// FilterValues converts a raw filter value into the string domain used by the
// current storage indexes for the supplied operator.
func FilterValues(value any, op contracts.Operator) ([]string, error) {
	if op == contracts.OpIn {
		return ListValues(value)
	}
	return []string{Scalar(value)}, nil
}

// ListValues converts a raw list/array filter value into scalar strings.
func ListValues(value any) ([]string, error) {
	items, ok := Elements(value)
	if !ok {
		return nil, fmt.Errorf("in operator value must be a list")
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, Scalar(item))
	}
	return out, nil
}

// Elements returns JSON-like list elements from []any or any Go slice/array.
func Elements(value any) ([]any, bool) {
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

// Scalar converts a filter scalar into the string domain used by the current
// storage indexes. JSON null / Go nil is not a literal ID or source; treat it
// like a blank value so equality filters fail closed instead of searching for
// fmt's "<nil>" sentinel text.
func Scalar(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

// NormalizeEqualityValues trims, optionally lowercases, and de-duplicates values
// while preserving first-seen order.
func NormalizeEqualityValues(raw []string, lower bool) []string {
	out := make([]string, 0, len(raw))
	for _, value := range raw {
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
