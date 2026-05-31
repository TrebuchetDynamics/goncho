package convertpolicy

import "testing"

func TestStableIDSegmentNormalizesConversationID(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{in: "  Conv 42/A  ", want: "conv-42-a"},
		{in: "!!!", want: "conversation"},
		{in: "Already--Split", want: "already-split"},
	} {
		if got := StableIDSegment(tc.in); got != tc.want {
			t.Fatalf("StableIDSegment(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPythonLiteralToJSONishConvertsPythonBarewordsAndQuotes(t *testing.T) {
	got := PythonLiteralToJSONish(`{'ok': True, "missing": None, 'bad': False, 'quote': 'a"b'}`)
	want := `{"ok": true, "missing": null, "bad": false, "quote": "a\"b"}`
	if got != want {
		t.Fatalf("PythonLiteralToJSONish() = %q, want %q", got, want)
	}
}
