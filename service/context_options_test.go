package goncho

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestContractContextParamsRepresentationOptionsJSONShape(t *testing.T) {
	limitToSession := true
	searchTopK := 10
	searchMaxDistance := 0.75
	includeMostFrequent := true
	maxConclusions := 25

	raw, err := json.Marshal(ContextParams{
		Peer:                "user-juan",
		Query:               "Atlas",
		SessionKey:          "sess-telegram",
		PeerTarget:          "user-juan",
		PeerPerspective:     "assistant",
		LimitToSession:      &limitToSession,
		SearchTopK:          &searchTopK,
		SearchMaxDistance:   &searchMaxDistance,
		IncludeMostFrequent: &includeMostFrequent,
		MaxConclusions:      &maxConclusions,
	})
	if err != nil {
		t.Fatal(err)
	}

	text := string(raw)
	for _, want := range []string{
		`"peer_target":"user-juan"`,
		`"peer_perspective":"assistant"`,
		`"limit_to_session":true`,
		`"search_top_k":10`,
		`"search_max_distance":0.75`,
		`"include_most_frequent":true`,
		`"max_conclusions":25`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ContextParams JSON missing %s in %s", want, raw)
		}
	}
}

func TestService_ContextOmittedRepresentationOptionsPreservesSameChatDefault(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	seedContextOptionConclusions(t, ctx, svc, "telegram:6586915095")

	got, err := svc.Context(ctx, ContextParams{
		Peer:       "telegram:6586915095",
		Query:      "codename",
		MaxTokens:  400,
		SessionKey: "telegram:6586915095",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Contains(got.Conclusions, "same-chat codename orchid") {
		t.Fatalf("Conclusions = %#v, want same-chat conclusion", got.Conclusions)
	}
	if slices.Contains(got.Conclusions, "other-chat codename orchid") {
		t.Fatalf("Conclusions leaked other-chat result: %#v", got.Conclusions)
	}
	if len(got.Unavailable) != 0 {
		t.Fatalf("Unavailable = %#v, want no degraded evidence for omitted options", got.Unavailable)
	}
}

func TestService_ContextLimitToSessionCannotWidenThroughUserScope(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	seedContextOptionConclusions(t, ctx, svc, "user-juan")
	limitToSession := true

	got, err := svc.Context(ctx, ContextParams{
		Peer:           "user-juan",
		Query:          "codename",
		MaxTokens:      400,
		Scope:          "user",
		LimitToSession: &limitToSession,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Conclusions) != 0 {
		t.Fatalf("Conclusions = %#v, want no widened recall without a session_key", got.Conclusions)
	}
	requireUnavailableFields(t, got, "limit_to_session")
}

func TestService_ContextUnsupportedRepresentationOptionsReturnUnavailableEvidence(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	searchTopK := 10
	searchMaxDistance := 0.8
	includeMostFrequent := true
	maxConclusions := 25

	got, err := svc.Context(ctx, ContextParams{
		Peer:                "telegram:6586915095",
		Query:               "coding preferences",
		MaxTokens:           400,
		SessionKey:          "telegram:6586915095",
		PeerTarget:          "telegram:6586915095",
		PeerPerspective:     "assistant",
		SearchTopK:          &searchTopK,
		SearchMaxDistance:   &searchMaxDistance,
		IncludeMostFrequent: &includeMostFrequent,
		MaxConclusions:      &maxConclusions,
	})
	if err != nil {
		t.Fatal(err)
	}

	requireUnavailableFields(t, got,
		"peer_target",
		"peer_perspective",
		"search_top_k",
		"search_max_distance",
		"include_most_frequent",
		"max_conclusions",
	)
}

func requireUnavailableFields(t *testing.T, got ContextResult, wantFields ...string) {
	t.Helper()

	seen := make(map[string]ContextUnavailableEvidence, len(got.Unavailable))
	for _, item := range got.Unavailable {
		seen[item.Field] = item
	}

	for _, field := range wantFields {
		item, ok := seen[field]
		if !ok {
			t.Fatalf("Unavailable = %#v, missing field %q", got.Unavailable, field)
		}
		if item.Reason == "" {
			t.Fatalf("Unavailable[%s] has empty reason: %#v", field, item)
		}
	}
}

func seedContextOptionConclusions(t *testing.T, ctx context.Context, svc *Service, peer string) {
	t.Helper()

	for _, item := range []struct {
		sessionKey string
		conclusion string
	}{
		{
			sessionKey: "telegram:6586915095",
			conclusion: "same-chat codename orchid",
		},
		{
			sessionKey: "discord:channel-9",
			conclusion: "other-chat codename orchid",
		},
	} {
		if _, err := svc.Conclude(ctx, ConcludeParams{
			Peer:       peer,
			Conclusion: item.conclusion,
			SessionKey: item.sessionKey,
		}); err != nil {
			t.Fatalf("seed conclusion %q: %v", item.conclusion, err)
		}
	}
}
