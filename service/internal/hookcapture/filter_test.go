package hookcapture

import (
	"strings"
	"testing"
)

func TestFilterRedactsSecretShapesAndMetadataKeys(t *testing.T) {
	payload := Payload{
		Content:  "Authorization: Bearer secret-token",
		Input:    `{"api_token":"secret-json"}`,
		Output:   "OPENAI_API_KEY=sk-live-secret",
		Metadata: map[string]string{"api_token": "metadata-secret", "safe": "kept"},
	}

	got := Filter(payload, 1024)
	stored := got.Payload.Content + got.Payload.Input + got.Payload.Output + got.Payload.Metadata["api_token"]
	for _, leaked := range []string{"secret-token", "secret-json", "sk-live-secret", "metadata-secret"} {
		if strings.Contains(stored, leaked) {
			t.Fatalf("filtered payload leaked %q in %q", leaked, stored)
		}
	}
	for _, want := range []string{"[REDACTED:authorization]", "[REDACTED:json_secret]", "[REDACTED:env_secret]", "[REDACTED:metadata_secret]"} {
		if !strings.Contains(stored, want) {
			t.Fatalf("filtered payload missing %q in %q", want, stored)
		}
	}
	if !got.Redacted || got.RedactionCount != 4 {
		t.Fatalf("result redaction = (%v, %d), want true and 4", got.Redacted, got.RedactionCount)
	}
	if got.Payload.Metadata["safe"] != "kept" {
		t.Fatalf("safe metadata = %q, want kept", got.Payload.Metadata["safe"])
	}
}

func TestFilterTruncatesAfterValidUTF8(t *testing.T) {
	got := Filter(Payload{Content: string([]byte{'a', 0xff, 'b'}) + strings.Repeat("x", 16)}, 8)
	if !got.Truncated {
		t.Fatal("Truncated = false, want true")
	}
	if strings.ContainsRune(got.Payload.Content, rune(0xff)) {
		t.Fatalf("content contains invalid byte replacement source: %q", got.Payload.Content)
	}
	if len([]byte(got.Payload.Content)) > 8 {
		t.Fatalf("content bytes = %d, want <= 8", len([]byte(got.Payload.Content)))
	}
}
