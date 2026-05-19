package goncho

import "testing"

func TestGonchoHonchoBaseURLLocalSentinel(t *testing.T) {
	t.Run("api key precedence", func(t *testing.T) {
		root := ResolvePluginConfig(PluginConfigInput{
			Host: "hermes",
			Raw: map[string]any{
				"apiKey": "root-key",
				"hosts": map[string]any{
					"hermes": map[string]any{"apiKey": "host-key"},
				},
			},
		})
		if root.APIKey != "host-key" || root.APIKeySource != "host" {
			t.Fatalf("APIKey/APIKeySource = %q/%q, want host-key/host", root.APIKey, root.APIKeySource)
		}
	})

	t.Run("base url sentinel", func(t *testing.T) {
		for _, raw := range []map[string]any{
			{"baseUrl": "http://localhost:8000"},
			{"base_url": "https://honcho.example.com"},
			{"baseUrl": "localhost:8000"},
			{},
		} {
			input := PluginConfigInput{Host: "hermes", Raw: raw}
			if len(raw) == 0 {
				input.Env = map[string]string{"HONCHO_BASE_URL": "http://10.0.0.5:8000"}
			}
			got := ResolvePluginConfig(input)
			if got.APIKey != LocalHonchoAPIKeySentinel || got.APIKeySource != "base_url" {
				t.Fatalf("ResolvePluginConfig(%v) APIKey/APIKeySource = %q/%q, want local/base_url", raw, got.APIKey, got.APIKeySource)
			}
		}
	})

	t.Run("garbage base url fails closed", func(t *testing.T) {
		for _, raw := range []string{"true", "false", "null", "1", "12345", "localhost"} {
			got := ResolvePluginConfig(PluginConfigInput{
				Host: "hermes",
				Raw:  map[string]any{"baseUrl": raw},
			})
			if got.APIKey != "" {
				t.Fatalf("baseUrl %q APIKey = %q, want empty", raw, got.APIKey)
			}
			if got.Enabled {
				t.Fatalf("baseUrl %q Enabled = true, want false", raw)
			}
			if !got.HasEvidence(GonchoConfigBaseURLInvalid) {
				t.Fatalf("baseUrl %q evidence = %+v, want %s", raw, got.Evidence, GonchoConfigBaseURLInvalid)
			}
		}
	})
}

func TestGonchoWriteFrequencyParsingAndRouting(t *testing.T) {
	root := ResolvePluginConfig(PluginConfigInput{
		Host: "hermes",
		Raw: map[string]any{
			"writeFrequency": "turn",
			"hosts": map[string]any{
				"hermes": map[string]any{"writeFrequency": "session"},
			},
		},
	})
	if root.WriteFrequency.Mode != WriteFrequencySession {
		t.Fatalf("host writeFrequency mode = %q, want %q", root.WriteFrequency.Mode, WriteFrequencySession)
	}

	for _, tc := range []struct {
		raw  any
		mode WriteFrequencyMode
		n    int
	}{
		{raw: "async", mode: WriteFrequencyAsync},
		{raw: "turn", mode: WriteFrequencyTurn},
		{raw: "session", mode: WriteFrequencySession},
		{raw: 5, mode: WriteFrequencyEvery, n: 5},
		{raw: "3", mode: WriteFrequencyEvery, n: 3},
	} {
		got := ParsePluginWriteFrequency(tc.raw)
		if got.Mode != tc.mode || got.Every != tc.n {
			t.Fatalf("ParsePluginWriteFrequency(%v) = %+v, want mode=%s every=%d", tc.raw, got, tc.mode, tc.n)
		}
	}

	session := PluginMemorySession{Key: "s1"}
	flushes := 0
	router := NewPluginWriteRouter(PluginWriteRouterConfig{
		Frequency: ParsePluginWriteFrequency("turn"),
		Flusher: PluginSessionFlusherFunc(func(PluginMemorySession) error {
			flushes++
			return nil
		}),
	})
	if got := router.Save(session); got.Code != GonchoWriteFlushed || flushes != 1 {
		t.Fatalf("turn Save = %+v flushes=%d, want flushed once", got, flushes)
	}

	router = NewPluginWriteRouter(PluginWriteRouterConfig{
		Frequency: ParsePluginWriteFrequency(3),
		Flusher: PluginSessionFlusherFunc(func(PluginMemorySession) error {
			flushes++
			return nil
		}),
	})
	flushes = 0
	for i := 0; i < 2; i++ {
		if got := router.Save(session); got.Code != GonchoWriteDeferred {
			t.Fatalf("turn %d Save code = %q, want %q", i+1, got.Code, GonchoWriteDeferred)
		}
	}
	if got := router.Save(session); got.Code != GonchoWriteFlushed || flushes != 1 {
		t.Fatalf("third Save = %+v flushes=%d, want flushed once", got, flushes)
	}
}

func TestGonchoSessionNameResolution(t *testing.T) {
	cfg := PluginConfig{
		WorkspaceID:       "my-workspace",
		PeerName:          "eri",
		SessionPeerPrefix: true,
		SessionStrategy:   "per-directory",
		Sessions:          map[string]string{"/work/project": "manual-name"},
	}

	if got := ResolvePluginSessionName(cfg, SessionNameInput{CWD: "/work/project", Title: "title"}); got != "manual-name" {
		t.Fatalf("manual override = %q, want manual-name", got)
	}
	if got := ResolvePluginSessionName(cfg, SessionNameInput{CWD: "/work/other", Title: "my project/name!"}); got != "eri-my-project-name" {
		t.Fatalf("title resolution = %q, want eri-my-project-name", got)
	}
	if got := ResolvePluginSessionName(cfg, SessionNameInput{CWD: "/work/dir", Title: "!!! ###"}); got != "eri-dir" {
		t.Fatalf("invalid title fallback = %q, want eri-dir", got)
	}
	if got := ResolvePluginSessionName(PluginConfig{SessionStrategy: "per-session"}, SessionNameInput{CWD: "/work/dir", SessionID: "20260309_175514_9797dd"}); got != "20260309_175514_9797dd" {
		t.Fatalf("per-session = %q, want session id", got)
	}
	if got := ResolvePluginSessionName(PluginConfig{WorkspaceID: "global", SessionStrategy: "global"}, SessionNameInput{CWD: "/work/dir"}); got != "global" {
		t.Fatalf("global = %q, want global", got)
	}
	if got := ResolvePluginSessionName(PluginConfig{}, SessionNameInput{GatewaySessionKey: "agent:main:telegram:dm:8439114563"}); got != "agent-main-telegram-dm-8439114563" {
		t.Fatalf("gateway key = %q, want sanitized gateway key", got)
	}
}

func TestGonchoPinPeerNameResolution(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cfg     PluginConfig
		runtime string
		key     string
		want    string
	}{
		{
			name:    "runtime wins by default",
			cfg:     PluginConfig{PeerName: "Igor", PinPeerName: false, AIPeer: "hermes"},
			runtime: "86701400",
			key:     "telegram:86701400",
			want:    "86701400",
		},
		{
			name:    "pin uses configured peer",
			cfg:     PluginConfig{PeerName: "Igor", PinPeerName: true, AIPeer: "hermes"},
			runtime: "86701400",
			key:     "telegram:86701400",
			want:    "Igor",
		},
		{
			name:    "pin without peer falls back to runtime",
			cfg:     PluginConfig{PinPeerName: true, AIPeer: "hermes"},
			runtime: "86701400",
			key:     "telegram:86701400",
			want:    "86701400",
		},
		{
			name: "session key fallback",
			cfg:  PluginConfig{AIPeer: "hermes"},
			key:  "telegram:123",
			want: "user-telegram-123",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolvePluginPeerNames(PluginPeerInput{
				Config:              tc.cfg,
				RuntimeUserPeerName: tc.runtime,
				SessionKey:          tc.key,
			})
			if got.UserPeerID != tc.want {
				t.Fatalf("UserPeerID = %q, want %q", got.UserPeerID, tc.want)
			}
			if got.AssistantPeerID != "hermes" {
				t.Fatalf("AssistantPeerID = %q, want hermes", got.AssistantPeerID)
			}
		})
	}
}
