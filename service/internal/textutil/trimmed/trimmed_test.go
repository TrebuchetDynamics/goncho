package trimmed

import "testing"

func TestSpaceCaseAndBlankContracts(t *testing.T) {
	if got := Space(" memory \t"); got != "memory" {
		t.Fatalf("Space() = %q, want memory", got)
	}
	if !Blank(" \n\t") || NonBlank(" \n\t") {
		t.Fatalf("blank predicates should share Space policy")
	}
	if got := Lower(" Memory "); got != "memory" {
		t.Fatalf("Lower() = %q, want memory", got)
	}
	if got := Upper(" get "); got != "GET" {
		t.Fatalf("Upper() = %q, want GET", got)
	}
}

func TestFirstNonBlankSharesSpacePolicy(t *testing.T) {
	if got := FirstNonBlank(" ", "\talpha ", "beta"); got != "alpha" {
		t.Fatalf("FirstNonBlank() = %q, want alpha", got)
	}
	if got := FirstNonBlank(" ", "\n"); got != "" {
		t.Fatalf("FirstNonBlank() = %q, want empty", got)
	}
}

func TestComparisonAndOptionalFilterContracts(t *testing.T) {
	if !Equal(" * ", "*") || Equal("Memory", "memory") {
		t.Fatalf("Equal should trim without folding case")
	}
	if !EqualFold(" Memory ", "memory") {
		t.Fatalf("EqualFold should trim and fold case")
	}
	if !Contains([]string{"alpha", " * "}, "*") || Contains([]string{"Memory"}, "memory") {
		t.Fatalf("Contains should trim without folding case")
	}
	if !ContainsEqualFold([]string{"alpha", " Memory "}, "memory") || ContainsEqualFold([]string{"memory"}, "message") {
		t.Fatalf("ContainsEqualFold should trim and fold case")
	}
	if !OptionalMatch("workspace-a", " workspace-a ") || !OptionalMatch("workspace-a", " ") {
		t.Fatalf("OptionalMatch should trim optional filter")
	}
	if !OptionalMatchOrEmpty("", "workspace-a") {
		t.Fatalf("OptionalMatchOrEmpty should admit legacy empty values")
	}
}

func TestContainsMatchUsesEqualerContract(t *testing.T) {
	lastByteMatches := func(value, want string) bool {
		if value == "" || want == "" {
			return false
		}
		return value[len(value)-1:] == want[len(want)-1:]
	}
	if !ContainsMatch([]string{"alpha", "note"}, "scope", lastByteMatches) {
		t.Fatalf("ContainsMatch should use caller-supplied equality policy")
	}
	if ContainsMatch([]string{"alpha"}, "alpha", nil) {
		t.Fatalf("ContainsMatch should reject nil equality policy")
	}
}

func TestBoundaryTrimContracts(t *testing.T) {
	if got := SpaceAndQuotes(" “alpha” "); got != "alpha" {
		t.Fatalf("SpaceAndQuotes() = %q, want alpha", got)
	}
	if got := SentenceBoundary("what?!"); got != "what" {
		t.Fatalf("SentenceBoundary() = %q, want what", got)
	}
	if got := QuestionPunctuation("?what!"); got != "what" {
		t.Fatalf("QuestionPunctuation() = %q, want what", got)
	}
	if got := QuestionPhraseBoundary("? what ."); got != "what" {
		t.Fatalf("QuestionPhraseBoundary() = %q, want what", got)
	}
}
