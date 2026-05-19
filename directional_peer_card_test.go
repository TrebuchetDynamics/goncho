package goncho

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestService_DirectionalPeerCardsIsolateObserverObservedPairs(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	if err := svc.SetProfile(ctx, "bob", []string{"Gormes knows Bob"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetProfileForTarget(ctx, "alice", "alice", []string{"Alice self card"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetProfileForTarget(ctx, "alice", "bob", []string{"Alice saw Bob order tea"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetProfileForTarget(ctx, "charlie", "bob", []string{"Charlie saw Bob order coffee"}); err != nil {
		t.Fatal(err)
	}

	defaultBob, err := svc.Profile(ctx, "bob")
	if err != nil {
		t.Fatal(err)
	}
	aliceSelf, err := svc.ProfileForTarget(ctx, "alice", "alice")
	if err != nil {
		t.Fatal(err)
	}
	aliceBob, err := svc.ProfileForTarget(ctx, "alice", "bob")
	if err != nil {
		t.Fatal(err)
	}
	charlieBob, err := svc.ProfileForTarget(ctx, "charlie", "bob")
	if err != nil {
		t.Fatal(err)
	}

	assertCard(t, defaultBob.Card, []string{"Gormes knows Bob"})
	assertCard(t, aliceSelf.Card, []string{"Alice self card"})
	assertCard(t, aliceBob.Card, []string{"Alice saw Bob order tea"})
	assertCard(t, charlieBob.Card, []string{"Charlie saw Bob order coffee"})
}

func TestService_SetProfileForTargetReplacesWholeCardAndCapsAtFortyFacts(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	var tooMany []string
	for i := 1; i <= 45; i++ {
		tooMany = append(tooMany, fmt.Sprintf("Fact %02d", i))
	}
	if err := svc.SetProfileForTarget(ctx, "alice", "bob", tooMany); err != nil {
		t.Fatal(err)
	}

	capped, err := svc.ProfileForTarget(ctx, "alice", "bob")
	if err != nil {
		t.Fatal(err)
	}
	if len(capped.Card) != 40 {
		t.Fatalf("card len = %d, want 40", len(capped.Card))
	}
	if capped.Card[39] != "Fact 40" {
		t.Fatalf("last capped fact = %q, want Fact 40", capped.Card[39])
	}
	if slices.Contains(capped.Card, "Fact 41") {
		t.Fatalf("card contains uncapped fact: %#v", capped.Card)
	}

	if err := svc.SetProfileForTarget(ctx, "alice", "bob", []string{"Replacement fact"}); err != nil {
		t.Fatal(err)
	}
	replaced, err := svc.ProfileForTarget(ctx, "alice", "bob")
	if err != nil {
		t.Fatal(err)
	}
	assertCard(t, replaced.Card, []string{"Replacement fact"})
}

func TestService_ContextReportsDefaultGormesObserverWhenDirectionalRepresentationUnavailable(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	if err := svc.SetProfile(ctx, "bob", []string{"Gormes default Bob card"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetProfileForTarget(ctx, "alice", "bob", []string{"Alice private Bob card"}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Context(ctx, ContextParams{
		Peer:            "bob",
		PeerTarget:      "bob",
		PeerPerspective: "alice",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertCard(t, got.PeerCard, []string{"Gormes default Bob card"})
	if slices.Contains(got.PeerCard, "Alice private Bob card") {
		t.Fatalf("context leaked directional card while representation is unavailable: %#v", got.PeerCard)
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		`"observer_peer_id":"gormes"`,
		`"observed_peer_id":"bob"`,
		`"peer_target"`,
		`"peer_perspective"`,
		"default gormes observer view",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("context JSON missing %s in %s", want, raw)
		}
	}
}

func assertCard(t *testing.T, got, want []string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("card = %#v, want %#v", got, want)
	}
}
