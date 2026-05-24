package goncho

import "testing"

func TestPluginRuntimePublicFacadeResolvesConfigSessionAndPeers(t *testing.T) {
	cfg := ResolvePluginConfig(PluginConfigInput{
		Host: "hermes",
		Raw: map[string]any{
			"baseUrl":           "http://localhost:8000",
			"peerName":          "eri",
			"sessionPeerPrefix": true,
		},
	})
	if cfg.APIKey != LocalHonchoAPIKeySentinel || cfg.APIKeySource != "base_url" || !cfg.Enabled {
		t.Fatalf("config auth = key:%q source:%q enabled:%v, want local base-url sentinel", cfg.APIKey, cfg.APIKeySource, cfg.Enabled)
	}
	if !cfg.HasEvidence(GonchoConfigLocalBaseURL) {
		t.Fatalf("Evidence = %+v, want %s", cfg.Evidence, GonchoConfigLocalBaseURL)
	}

	session := ResolvePluginSessionName(cfg, SessionNameInput{CWD: "/work/project", Title: "My Project"})
	if session != "eri-My-Project" {
		t.Fatalf("session = %q, want peer-prefixed sanitized title", session)
	}

	peers := ResolvePluginPeerNames(PluginPeerInput{Config: cfg, RuntimeUserPeerName: "86701400", SessionKey: "telegram:86701400"})
	if peers.UserPeerID != "86701400" || peers.AssistantPeerID != "hermes" {
		t.Fatalf("peers = %+v, want runtime user and host assistant IDs", peers)
	}
}
