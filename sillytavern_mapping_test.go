package goncho

import (
	"slices"
	"strings"
	"testing"
)

func TestSillyTavernPeerModesMapSharedAndPersonaScopedUserPeers(t *testing.T) {
	shared := MapSillyTavernIntegration(SillyTavernIntegrationInput{
		PeerMode:       "Single peer for all personas",
		PeerName:       "alice",
		PersonaName:    "Scholar Persona",
		SessionNaming:  "auto",
		ChatInstanceID: "chat-1",
		EnrichmentMode: "context only",
	})
	separate := MapSillyTavernIntegration(SillyTavernIntegrationInput{
		PeerMode:       "Separate peer per persona",
		PeerName:       "alice",
		PersonaName:    "Scholar Persona",
		SessionNaming:  "auto",
		ChatInstanceID: "chat-2",
		EnrichmentMode: "context only",
	})

	if len(shared.Unsupported) != 0 {
		t.Fatalf("shared Unsupported = %+v, want none", shared.Unsupported)
	}
	if len(separate.Unsupported) != 0 {
		t.Fatalf("separate Unsupported = %+v, want none", separate.Unsupported)
	}
	if shared.WorkspaceID != "sillytavern" || separate.WorkspaceID != "sillytavern" {
		t.Fatalf("workspace IDs = %q/%q, want both sillytavern", shared.WorkspaceID, separate.WorkspaceID)
	}
	if shared.UserPeerID != "alice" {
		t.Fatalf("shared UserPeerID = %q, want alice", shared.UserPeerID)
	}
	if separate.UserPeerID != "alice:persona:scholar-persona" {
		t.Fatalf("separate UserPeerID = %q, want persona-scoped peer", separate.UserPeerID)
	}
	if shared.UserPeerID == separate.UserPeerID {
		t.Fatalf("peer modes collapsed to one peer %q", shared.UserPeerID)
	}
}

func TestSillyTavernSessionNamingModesAndResetEvidence(t *testing.T) {
	tests := []struct {
		name            string
		input           SillyTavernIntegrationInput
		wantSession     string
		wantOrphaned    string
		wantUnsupported string
	}{
		{
			name: "auto per-chat",
			input: SillyTavernIntegrationInput{
				PeerName:       "alice",
				SessionNaming:  "auto",
				ChatInstanceID: "chat-hash-42",
				EnrichmentMode: "context only",
			},
			wantSession: "sillytavern:chat:chat-hash-42",
		},
		{
			name: "per-character",
			input: SillyTavernIntegrationInput{
				PeerName:       "alice",
				SessionNaming:  "per character",
				CharacterName:  "Mira the Cartographer",
				EnrichmentMode: "context only",
			},
			wantSession: "sillytavern:session:character:mira-the-cartographer",
		},
		{
			name: "custom",
			input: SillyTavernIntegrationInput{
				PeerName:          "alice",
				SessionNaming:     "custom",
				CustomSessionName: "night-market-arc",
				EnrichmentMode:    "context only",
			},
			wantSession: "sillytavern:session:custom:night-market-arc",
		},
		{
			name: "existing session stays frozen",
			input: SillyTavernIntegrationInput{
				PeerName:           "alice",
				SessionNaming:      "per character",
				CharacterName:      "Renamed Character",
				ExistingSessionKey: "sillytavern:chat:original-chat",
				EnrichmentMode:     "context only",
			},
			wantSession: "sillytavern:chat:original-chat",
		},
		{
			name: "reset orphans active session and starts new chat",
			input: SillyTavernIntegrationInput{
				PeerName:           "alice",
				SessionNaming:      "auto",
				ChatInstanceID:     "chat-hash-43",
				ExistingSessionKey: "sillytavern:chat:chat-hash-42",
				ResetActiveSession: true,
				EnrichmentMode:     "context only",
			},
			wantSession:  "sillytavern:chat:chat-hash-43",
			wantOrphaned: "sillytavern:chat:chat-hash-42",
		},
		{
			name: "missing new-chat identifier degrades",
			input: SillyTavernIntegrationInput{
				PeerName:       "alice",
				SessionNaming:  "auto",
				EnrichmentMode: "context only",
			},
			wantUnsupported: "session_strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapSillyTavernIntegration(tt.input)
			if tt.wantUnsupported != "" {
				if !hasUnsupportedField(got.Unsupported, tt.wantUnsupported) {
					t.Fatalf("Unsupported = %+v, want %s diagnostic", got.Unsupported, tt.wantUnsupported)
				}
				return
			}
			if len(got.Unsupported) != 0 {
				t.Fatalf("Unsupported = %+v, want none", got.Unsupported)
			}
			if got.SessionKey != tt.wantSession {
				t.Fatalf("SessionKey = %q, want %q", got.SessionKey, tt.wantSession)
			}
			if got.OrphanedSessionKey != tt.wantOrphaned {
				t.Fatalf("OrphanedSessionKey = %q, want %q", got.OrphanedSessionKey, tt.wantOrphaned)
			}
		})
	}
}

func TestSillyTavernGroupChatMapsOnePeerPerCharacterAndLazyAddsSpeakers(t *testing.T) {
	got := MapSillyTavernIntegration(SillyTavernIntegrationInput{
		PeerName:                 "alice",
		SessionNaming:            "auto",
		ChatInstanceID:           "group-chat-7",
		GroupCharacterNames:      []string{"Mira", "Jules"},
		ExistingCharacterPeerIDs: []string{"sillytavern:character:mira"},
		MessageCharacterName:     "Kade",
		EnrichmentMode:           "context only",
	})

	if len(got.Unsupported) != 0 {
		t.Fatalf("Unsupported = %+v, want none", got.Unsupported)
	}
	wantPeers := []string{
		"sillytavern:character:mira",
		"sillytavern:character:jules",
		"sillytavern:character:kade",
	}
	if !slices.Equal(got.CharacterPeerIDs, wantPeers) {
		t.Fatalf("CharacterPeerIDs = %v, want %v", got.CharacterPeerIDs, wantPeers)
	}
	wantLazy := []string{
		"sillytavern:character:jules",
		"sillytavern:character:kade",
	}
	if !slices.Equal(got.LazyAddedCharacterPeerIDs, wantLazy) {
		t.Fatalf("LazyAddedCharacterPeerIDs = %v, want %v", got.LazyAddedCharacterPeerIDs, wantLazy)
	}
	for _, peerID := range got.CharacterPeerIDs {
		if strings.Contains(peerID, "group") {
			t.Fatalf("character peer %q collapsed group identity into peer ID", peerID)
		}
	}
}

func TestSillyTavernEnrichmentModesAndUnsupportedPanelEvidence(t *testing.T) {
	tests := []struct {
		mode          string
		wantContext   bool
		wantReasoning bool
		wantTools     bool
	}{
		{mode: "Context only", wantContext: true},
		{mode: "Reasoning", wantContext: true, wantReasoning: true},
		{mode: "Tool call", wantContext: true, wantTools: true},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := MapSillyTavernIntegration(SillyTavernIntegrationInput{
				PeerName:       "alice",
				SessionNaming:  "auto",
				ChatInstanceID: "chat-1",
				EnrichmentMode: tt.mode,
			})
			if len(got.Unsupported) != 0 {
				t.Fatalf("Unsupported = %+v, want none", got.Unsupported)
			}
			if got.InjectContext != tt.wantContext || got.UseReasoning != tt.wantReasoning || got.ExposeTools != tt.wantTools {
				t.Fatalf("mode %q mapped to context:%v reasoning:%v tools:%v, want context:%v reasoning:%v tools:%v",
					tt.mode, got.InjectContext, got.UseReasoning, got.ExposeTools,
					tt.wantContext, tt.wantReasoning, tt.wantTools)
			}
			if tt.wantReasoning && got.ReasoningToolName != "honcho_chat" {
				t.Fatalf("ReasoningToolName = %q, want honcho_chat", got.ReasoningToolName)
			}
			for _, name := range got.ExternalToolNames {
				if !strings.HasPrefix(name, "honcho_") {
					t.Fatalf("external tool name %q does not preserve honcho_ prefix", name)
				}
				if strings.HasPrefix(name, "goncho_") {
					t.Fatalf("external tool name %q leaked internal goncho prefix", name)
				}
			}
			if !slices.Contains(got.ExternalToolNames, "honcho_chat") {
				t.Fatalf("ExternalToolNames = %v, want honcho_chat for reasoning mode compatibility", got.ExternalToolNames)
			}
		})
	}

	degraded := MapSillyTavernIntegration(SillyTavernIntegrationInput{
		PeerName:              "alice",
		SessionNaming:         "auto",
		ChatInstanceID:        "chat-1",
		EnrichmentMode:        "ambient",
		UnsupportedPanelKnobs: []string{"injection_position", "prompt_template"},
	})
	for _, field := range []string{"enrichment_mode", "sillytavern_panel_knob"} {
		if !hasUnsupportedField(degraded.Unsupported, field) {
			t.Fatalf("Unsupported = %+v, want %s diagnostic", degraded.Unsupported, field)
		}
	}
}
