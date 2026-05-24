package goncho

import "testing"

func TestSillyTavernPublicFacadeMapsPanelControls(t *testing.T) {
	got := MapSillyTavernIntegration(SillyTavernIntegrationInput{
		PeerMode:       "Separate peer per persona",
		PeerName:       "alice",
		PersonaName:    "Scholar Persona",
		SessionNaming:  "auto",
		ChatInstanceID: "chat-1",
		EnrichmentMode: "reasoning",
	})
	if len(got.Unsupported) != 0 {
		t.Fatalf("Unsupported = %+v, want none", got.Unsupported)
	}
	if got.WorkspaceID != "sillytavern" || got.UserPeerID != "alice:persona:scholar-persona" || got.SessionKey != "sillytavern:chat:chat-1" {
		t.Fatalf("mapping = %+v, want public SillyTavern facade mapping", got)
	}
	if !got.InjectContext || !got.UseReasoning || got.ReasoningToolName != "honcho_chat" {
		t.Fatalf("enrichment = context:%v reasoning:%v tool:%q, want reasoning via honcho_chat", got.InjectContext, got.UseReasoning, got.ReasoningToolName)
	}
}
