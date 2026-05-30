package sourcefilter

import "testing"

func TestAllowsEmptyAndWildcardPermitAll(t *testing.T) {
	if !Allows(nil, "conclusion", false) {
		t.Fatalf("empty source list should permit all")
	}
	if !Allows([]string{" memory ", " * "}, "conclusion", false) {
		t.Fatalf("wildcard source list should permit all")
	}
}

func TestAllowsMatchesCaseInsensitiveTrimmedSource(t *testing.T) {
	if !Allows([]string{" conclusion "}, "CONCLUSION", false) {
		t.Fatalf("trimmed case-insensitive source should match")
	}
	if Allows([]string{"message"}, "conclusion", false) {
		t.Fatalf("unlisted source should not match")
	}
}

func TestAllowsEmptySourcePolicy(t *testing.T) {
	if !Allows([]string{"conclusion"}, " ", true) {
		t.Fatalf("empty source should match when legacy policy allows it")
	}
	if Allows([]string{"conclusion"}, " ", false) {
		t.Fatalf("empty source should not match when legacy policy denies it")
	}
}
