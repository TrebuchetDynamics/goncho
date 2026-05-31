package queryexpand

import (
	"slices"
	"testing"
)

func TestExpandUsesReciprocalSynonyms(t *testing.T) {
	got := Expand("authentication")
	for _, want := range []string{"auth", "login", "signin"} {
		if !contains(got.Terms, want) {
			t.Fatalf("Expand(authentication) terms = %+v, want reciprocal synonym %q", got.Terms, want)
		}
	}
}

func TestSynonymLexiconIsReciprocal(t *testing.T) {
	for term, aliases := range synonyms {
		for _, alias := range aliases {
			if !contains(synonyms[alias], term) {
				t.Fatalf("synonym lexicon drift: %q lists %q, but reverse entry is missing", term, alias)
			}
		}
	}
}

func TestExpandReturnsDeterministicSortedTerms(t *testing.T) {
	got := Expand("authentication")
	want := []string{"auth", "login", "signin"}
	if !slices.Equal(got.Terms, want) {
		t.Fatalf("Expand(authentication) terms = %v, want deterministic sorted terms %v", got.Terms, want)
	}
}

func TestExpandUsesTokenNormalizedSynonymKeys(t *testing.T) {
	got := Expand("credentials")
	for _, want := range []string{"auth", "login", "signin"} {
		if !contains(got.Terms, want) {
			t.Fatalf("Expand(credentials) terms = %+v, want token-normalized synonym %q", got.Terms, want)
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
