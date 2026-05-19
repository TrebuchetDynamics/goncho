package goncho

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestContractProfileResultJSONShape(t *testing.T) {
	raw, err := json.Marshal(ProfileResult{
		WorkspaceID: "default",
		Peer:        "telegram:6586915095",
		Card:        []string{"Likes exact reports"},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := `{"workspace_id":"default","peer":"telegram:6586915095","card":["Likes exact reports"]}`
	if string(raw) != want {
		t.Fatalf("profile json = %s, want %s", raw, want)
	}
}

func TestContractContextResultIncludesStableFields(t *testing.T) {
	raw, err := json.Marshal(ContextResult{
		WorkspaceID:    "default",
		Peer:           "telegram:6586915095",
		SessionKey:     "telegram:6586915095",
		PeerCard:       []string{"Blind", "Prefers exact outputs"},
		Representation: "The user prefers exact outputs.",
	})
	if err != nil {
		t.Fatal(err)
	}

	text := string(raw)
	if !strings.Contains(text, `"workspace_id":"default"`) {
		t.Fatalf("missing workspace_id in %s", raw)
	}
	if !strings.Contains(text, `"representation":"The user prefers exact outputs."`) {
		t.Fatalf("missing representation in %s", raw)
	}
	if !strings.Contains(text, `"session_key":"telegram:6586915095"`) {
		t.Fatalf("missing session_key in %s", raw)
	}
}

func TestContractSearchParamsOptionalScopeAndSourcesJSONShape(t *testing.T) {
	raw, err := json.Marshal(SearchParams{
		Peer:    "user-juan",
		Query:   "Atlas",
		Scope:   "user",
		Sources: []string{"discord"},
	})
	if err != nil {
		t.Fatal(err)
	}

	text := string(raw)
	if !strings.Contains(text, `"scope":"user"`) {
		t.Fatalf("missing scope in %s", raw)
	}
	if !strings.Contains(text, `"sources":["discord"]`) {
		t.Fatalf("missing sources in %s", raw)
	}
}
