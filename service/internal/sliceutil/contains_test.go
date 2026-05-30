package sliceutil

import "testing"

func TestContainsReportsExactComparableMatch(t *testing.T) {
	if !Contains([]string{"operator", "system"}, "system") {
		t.Fatal("Contains did not find existing string")
	}
	if Contains([]string{"operator", "system"}, "developer") {
		t.Fatal("Contains found absent string")
	}
}

func TestContainsFuncReportsPredicateMatch(t *testing.T) {
	type warning struct{ code string }
	warnings := []warning{{code: "token_budget"}, {code: "semantic_unavailable"}}
	if !ContainsFunc(warnings, func(w warning) bool { return w.code == "semantic_unavailable" }) {
		t.Fatal("ContainsFunc did not find matching struct")
	}
	if ContainsFunc(warnings, func(w warning) bool { return w.code == "other" }) {
		t.Fatal("ContainsFunc found absent struct")
	}
}
