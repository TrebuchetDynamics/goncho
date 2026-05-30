package textutil

import "testing"

func TestTrimSpaceAndQuotes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "plain", value: " alpha ", want: "alpha"},
		{name: "ascii quotes", value: " \"alpha\" ", want: "alpha"},
		{name: "backticks", value: "`alpha`", want: "alpha"},
		{name: "curly quotes", value: " “alpha” ", want: "alpha"},
		{name: "apostrophe boundary", value: "'alpha'", want: "alpha"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := TrimSpaceAndQuotes(tc.value); got != tc.want {
				t.Fatalf("TrimSpaceAndQuotes(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}
