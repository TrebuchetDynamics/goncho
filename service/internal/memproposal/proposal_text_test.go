package memproposal

import "testing"

func TestSplitMarkerRecognizesSupportedPrefixes(t *testing.T) {
	prefix, body, ok := SplitMarker(" Remember: Deployment owner is Mira. ")
	if !ok || prefix != "remember" || body != "Deployment owner is Mira." {
		t.Fatalf("SplitMarker() = %q, %q, %v", prefix, body, ok)
	}
	if _, _, ok := SplitMarker("Note: Deployment owner is Mira."); ok {
		t.Fatal("SplitMarker unsupported prefix ok = true, want false")
	}
}

func TestSubjectUsesDurableFactGrammarThenFirstWords(t *testing.T) {
	if got := Subject("Deployment owner is Mira."); got != "Deployment owner" {
		t.Fatalf("Subject(fact) = %q, want Deployment owner", got)
	}
	if got := Subject("release checklist should cite rollback owners before launch"); got != "release checklist should cite rollback" {
		t.Fatalf("Subject(fallback) = %q", got)
	}
}

func TestReviewRiskDetectors(t *testing.T) {
	if !IsLowConfidence("I think the deploy window moved") {
		t.Fatal("IsLowConfidence() = false, want true")
	}
	if !IsPrivacySensitive("API token is sk-live-secret") {
		t.Fatal("IsPrivacySensitive() = false, want true")
	}
}
