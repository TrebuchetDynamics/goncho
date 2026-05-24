package goncho

import "testing"

func TestHostIntegrationPublicFacadeMapsAndPatchesConfig(t *testing.T) {
	mapped := MapHostIntegration(HostIntegrationInput{
		Host:             "opencode",
		PeerName:         "alice",
		SessionStrategy:  "per-directory",
		WorkingDirectory: "/work/acme/frontend",
		RecallMode:       "hybrid",
	})
	if len(mapped.Unsupported) != 0 {
		t.Fatalf("Unsupported = %+v, want none", mapped.Unsupported)
	}
	if mapped.SessionKey != "opencode:dir:/work/acme/frontend" || mapped.UserPeerID != "alice" {
		t.Fatalf("mapping = %+v, want public facade session and peer mapping", mapped)
	}

	updated, err := ApplyHostConfigPatch(HostConfigDocument{
		Hosts: map[string]HostRuntimeConfig{
			"opencode": {Workspace: "opencode", RecallMode: "hybrid"},
		},
	}, "opencode", HostConfigPatch{Workspace: stringPtr("team-acme")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Hosts["opencode"].Workspace != "team-acme" {
		t.Fatalf("patched workspace = %q, want team-acme", updated.Hosts["opencode"].Workspace)
	}

	compat := HonchoExternalCompatibility()
	if compat.InternalService != "goncho" || len(compat.ExternalToolNames) == 0 {
		t.Fatalf("compatibility = %+v, want public Honcho facade over Goncho", compat)
	}
}

func stringPtr(value string) *string {
	return &value
}
