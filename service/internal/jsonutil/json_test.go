package jsonutil

import "testing"

func TestStableIndentedUsesTwoSpacesAndTrailingNewline(t *testing.T) {
	got, err := StableIndented(map[string]any{"alpha": []int{1, 2}})
	if err != nil {
		t.Fatalf("StableIndented() error = %v", err)
	}
	want := "{\n  \"alpha\": [\n    1,\n    2\n  ]\n}\n"
	if string(got) != want {
		t.Fatalf("StableIndented() = %q, want %q", string(got), want)
	}
}
