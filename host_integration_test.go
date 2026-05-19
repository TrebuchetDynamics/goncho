package goncho

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestHostIntegrationMappingSupportsDocumentedSessionStrategies(t *testing.T) {
	tests := []struct {
		name    string
		input   HostIntegrationInput
		wantKey string
	}{
		{
			name: "opencode per-directory",
			input: HostIntegrationInput{
				Host:             "opencode",
				PeerName:         "alice",
				SessionStrategy:  "per-directory",
				WorkingDirectory: "/work/acme/frontend",
				RecallMode:       "hybrid",
			},
			wantKey: "opencode:dir:/work/acme/frontend",
		},
		{
			name: "opencode per-repo",
			input: HostIntegrationInput{
				Host:            "opencode",
				PeerName:        "alice",
				SessionStrategy: "per-repo",
				Repository:      "github.com/acme/gormes",
				RecallMode:      "hybrid",
			},
			wantKey: "opencode:repo:github.com/acme/gormes",
		},
		{
			name: "opencode per-session",
			input: HostIntegrationInput{
				Host:            "opencode",
				PeerName:        "alice",
				SessionStrategy: "per-session",
				HostSessionID:   "oc-session-7",
				RecallMode:      "hybrid",
			},
			wantKey: "opencode:session:oc-session-7",
		},
		{
			name: "opencode chat-instance",
			input: HostIntegrationInput{
				Host:            "opencode",
				PeerName:        "alice",
				SessionStrategy: "chat-instance",
				ChatInstanceID:  "chat-42",
				RecallMode:      "hybrid",
			},
			wantKey: "opencode:chat:chat-42",
		},
		{
			name: "opencode global",
			input: HostIntegrationInput{
				Host:            "opencode",
				PeerName:        "alice",
				SessionStrategy: "global",
				RecallMode:      "hybrid",
			},
			wantKey: "opencode:global",
		},
		{
			name: "sillytavern chat-instance",
			input: HostIntegrationInput{
				Host:            "sillytavern",
				PeerName:        "alice-rp",
				SessionStrategy: "chat-instance",
				ChatInstanceID:  "st-chat-9",
				RecallMode:      "context-only",
			},
			wantKey: "sillytavern:chat:st-chat-9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapHostIntegration(tt.input)
			if len(got.Unsupported) != 0 {
				t.Fatalf("Unsupported = %+v, want none", got.Unsupported)
			}
			if got.SessionKey != tt.wantKey {
				t.Fatalf("SessionKey = %q, want %q", got.SessionKey, tt.wantKey)
			}
			if got.UserPeerID != tt.input.PeerName {
				t.Fatalf("UserPeerID = %q, want %q", got.UserPeerID, tt.input.PeerName)
			}
			if got.WorkspaceID == "" {
				t.Fatal("WorkspaceID must be populated from host defaults or input")
			}
		})
	}
}

func TestHostIntegrationRecallModesMapContextToolsAndHybrid(t *testing.T) {
	tests := []struct {
		mode       string
		wantPrompt bool
		wantTools  bool
		wantMode   string
	}{
		{mode: "context", wantPrompt: true, wantTools: false, wantMode: "context"},
		{mode: "context-only", wantPrompt: true, wantTools: false, wantMode: "context"},
		{mode: "tools", wantPrompt: false, wantTools: true, wantMode: "tools"},
		{mode: "tool-only", wantPrompt: false, wantTools: true, wantMode: "tools"},
		{mode: "hybrid", wantPrompt: true, wantTools: true, wantMode: "hybrid"},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := MapHostIntegration(HostIntegrationInput{
				Host:             "opencode",
				PeerName:         "alice",
				SessionStrategy:  "per-directory",
				WorkingDirectory: "/work/acme/frontend",
				RecallMode:       tt.mode,
			})
			if len(got.Unsupported) != 0 {
				t.Fatalf("Unsupported = %+v, want none", got.Unsupported)
			}
			if got.RecallMode != tt.wantMode {
				t.Fatalf("RecallMode = %q, want %q", got.RecallMode, tt.wantMode)
			}
			if got.InjectContext != tt.wantPrompt || got.ExposeTools != tt.wantTools {
				t.Fatalf("recall behavior = context:%v tools:%v, want context:%v tools:%v",
					got.InjectContext, got.ExposeTools, tt.wantPrompt, tt.wantTools)
			}
		})
	}
}

func TestHostConfigPatchScopesWritesToSelectedHost(t *testing.T) {
	doc := HostConfigDocument{
		APIKey:   "hch-shared",
		BaseURL:  "http://127.0.0.1:8000",
		PeerName: "alice",
		Hosts: map[string]HostRuntimeConfig{
			"opencode": {
				Workspace:       "opencode",
				AIPeer:          "opencode",
				RecallMode:      "hybrid",
				SessionStrategy: "per-directory",
			},
			"sillytavern": {
				Workspace:  "sillytavern",
				PeerName:   "alice-rp",
				RecallMode: "context",
			},
		},
	}

	updated, err := ApplyHostConfigPatch(doc, "opencode", HostConfigPatch{
		Workspace:  stringPtr("team-acme"),
		RecallMode: stringPtr("tools"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if updated.Hosts["opencode"].Workspace != "team-acme" {
		t.Fatalf("opencode workspace = %q, want team-acme", updated.Hosts["opencode"].Workspace)
	}
	if updated.Hosts["opencode"].RecallMode != "tools" {
		t.Fatalf("opencode recall = %q, want tools", updated.Hosts["opencode"].RecallMode)
	}
	if updated.Hosts["sillytavern"] != doc.Hosts["sillytavern"] {
		t.Fatalf("sillytavern host config changed: got %+v want %+v", updated.Hosts["sillytavern"], doc.Hosts["sillytavern"])
	}
	if doc.Hosts["opencode"].Workspace != "opencode" {
		t.Fatalf("original document mutated: opencode workspace = %q", doc.Hosts["opencode"].Workspace)
	}
}

func TestHostConfigDocumentJSONUsesHonchoSharedKeys(t *testing.T) {
	raw, err := json.Marshal(HostConfigDocument{
		APIKey:   "hch-shared",
		BaseURL:  "http://127.0.0.1:8000",
		PeerName: "alice",
		Hosts: map[string]HostRuntimeConfig{
			"opencode": {
				Workspace:       "opencode",
				AIPeer:          "opencode",
				RecallMode:      "hybrid",
				SessionStrategy: "per-directory",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	text := string(raw)
	for _, key := range []string{`"apiKey"`, `"baseUrl"`, `"peerName"`, `"hosts"`, `"aiPeer"`, `"recallMode"`, `"sessionStrategy"`} {
		if !strings.Contains(text, key) {
			t.Fatalf("config json %s missing key %s", text, key)
		}
	}
	for _, key := range []string{`"APIKey"`, `"BaseURL"`, `"PeerName"`, `"AIPeer"`, `"RecallMode"`, `"SessionStrategy"`} {
		if strings.Contains(text, key) {
			t.Fatalf("config json %s leaked Go field key %s", text, key)
		}
	}
}

func TestHostIntegrationMappingReportsUnsupportedConfig(t *testing.T) {
	got := MapHostIntegration(HostIntegrationInput{
		Host:            "opencode",
		PeerName:        "alice",
		SessionStrategy: "per-repo",
		RecallMode:      "always-on",
	})

	if len(got.Unsupported) != 2 {
		t.Fatalf("Unsupported = %+v, want session_strategy and recall_mode diagnostics", got.Unsupported)
	}
	if !hasUnsupportedField(got.Unsupported, "session_strategy") {
		t.Fatalf("missing session_strategy diagnostic: %+v", got.Unsupported)
	}
	if !hasUnsupportedField(got.Unsupported, "recall_mode") {
		t.Fatalf("missing recall_mode diagnostic: %+v", got.Unsupported)
	}
	if got.SessionKey != "" {
		t.Fatalf("SessionKey = %q, want empty when required per-repo input is missing", got.SessionKey)
	}
}

func TestHonchoExternalCompatibilityKeepsGonchoInternalName(t *testing.T) {
	got := HonchoExternalCompatibility()
	if got.InternalService != "goncho" {
		t.Fatalf("InternalService = %q, want goncho", got.InternalService)
	}
	for _, name := range got.ExternalToolNames {
		if !strings.HasPrefix(name, "honcho_") {
			t.Fatalf("external tool name %q does not preserve honcho_ prefix", name)
		}
		if strings.HasPrefix(name, "goncho_") {
			t.Fatalf("external tool name %q leaked internal goncho prefix", name)
		}
	}
	for _, name := range []string{"honcho_profile", "honcho_search", "honcho_context", "honcho_conclude"} {
		if !slices.Contains(got.ExternalToolNames, name) {
			t.Fatalf("ExternalToolNames = %v, missing %s", got.ExternalToolNames, name)
		}
	}
}

func hasUnsupportedField(items []UnsupportedHostMapping, field string) bool {
	for _, item := range items {
		if item.Field == field && item.Reason != "" {
			return true
		}
	}
	return false
}

func stringPtr(value string) *string {
	return &value
}
