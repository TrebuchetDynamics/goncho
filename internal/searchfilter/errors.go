package searchfilter

import "strings"

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

func unsupportedFilter(field, operator, reason string) *UnsupportedFilterError {
	return &UnsupportedFilterError{
		Code:     "unsupported_filter",
		Field:    strings.Trim(field, "."),
		Operator: operator,
		Reason:   reason,
	}
}
