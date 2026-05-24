package gonchohttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

type HTTPServiceRestartE2EConfig struct {
	DBPath      string
	WorkspaceID string
	ObserverID  string
	PeerID      string
	SessionKey  string
}

type HTTPServiceRestartE2EReport struct {
	SQLiteRestartVerified               bool   `json:"sqlite_restart_verified"`
	NetworkRequired                     bool   `json:"network_required"`
	ExternalProviderRequired            bool   `json:"external_provider_required"`
	MessagesCreated                     int    `json:"messages_created"`
	SearchCountBeforeRestart            int    `json:"search_count_before_restart"`
	SearchCountAfterRestart             int    `json:"search_count_after_restart"`
	ContextHadProfileAfterRestart       bool   `json:"context_had_profile_after_restart"`
	ContextHadConclusionAfterRestart    bool   `json:"context_had_conclusion_after_restart"`
	ContextHadRecentMessageAfterRestart bool   `json:"context_had_recent_message_after_restart"`
	CompletionCondition                 string `json:"completion_condition"`
}

func RunHTTPServiceRestartE2E(ctx context.Context, cfg HTTPServiceRestartE2EConfig) (HTTPServiceRestartE2EReport, error) {
	cfg = cfg.withDefaults()
	store, err := memory.OpenSqlite(cfg.DBPath, 0, nil)
	if err != nil {
		return HTTPServiceRestartE2EReport{}, err
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		_ = store.Close(ctx)
		return HTTPServiceRestartE2EReport{}, err
	}
	service := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: cfg.WorkspaceID, ObserverPeerID: cfg.ObserverID}, nil)
	handler := NewServiceHandler(service)

	profileFact := "HTTP restart E2E keeps profile facts."
	userMessage := "HTTP restart E2E should preserve this recent message."
	conclusion := "HTTP restart E2E keeps conclusions across SQLite reopen."

	putJSONNoTest(handler, "/v3/workspaces/"+cfg.WorkspaceID+"/peers/"+cfg.PeerID+"/card", map[string]any{"card": []string{profileFact}})
	messages, err := requestJSONNoTest[goncho.CreateMessagesResult](handler, http.MethodPost, "/v3/workspaces/"+cfg.WorkspaceID+"/sessions/"+cfg.SessionKey+"/messages", map[string]any{"messages": []map[string]any{{"peer_id": cfg.PeerID, "role": "user", "content": userMessage}, {"peer_id": cfg.ObserverID, "role": "assistant", "content": "HTTP restart E2E assistant response."}}})
	if err != nil {
		_ = store.Close(ctx)
		return HTTPServiceRestartE2EReport{}, err
	}
	if _, err := requestJSONNoTest[goncho.ConcludeResult](handler, http.MethodPost, "/v3/workspaces/"+cfg.WorkspaceID+"/conclusions", map[string]any{"peer_id": cfg.PeerID, "conclusion": conclusion, "session_key": cfg.SessionKey}); err != nil {
		_ = store.Close(ctx)
		return HTTPServiceRestartE2EReport{}, err
	}
	beforeSearch, err := requestJSONNoTest[goncho.SearchResultSet](handler, http.MethodPost, "/v3/workspaces/"+cfg.WorkspaceID+"/peers/"+cfg.PeerID+"/search", map[string]any{"query": "SQLite reopen", "session_key": cfg.SessionKey})
	if err != nil {
		_ = store.Close(ctx)
		return HTTPServiceRestartE2EReport{}, err
	}
	if err := store.Close(ctx); err != nil {
		return HTTPServiceRestartE2EReport{}, err
	}

	reopened, err := memory.OpenSqlite(cfg.DBPath, 0, nil)
	if err != nil {
		return HTTPServiceRestartE2EReport{}, err
	}
	defer reopened.Close(ctx)
	if err := goncho.RunMigrations(reopened.DB()); err != nil {
		return HTTPServiceRestartE2EReport{}, err
	}
	reopenedService := goncho.NewService(reopened.DB(), goncho.Config{WorkspaceID: cfg.WorkspaceID, ObserverPeerID: cfg.ObserverID}, nil)
	reopenedHandler := NewServiceHandler(reopenedService)
	afterSearch, err := requestJSONNoTest[goncho.SearchResultSet](reopenedHandler, http.MethodPost, "/v3/workspaces/"+cfg.WorkspaceID+"/peers/"+cfg.PeerID+"/search", map[string]any{"query": "SQLite reopen", "session_key": cfg.SessionKey})
	if err != nil {
		return HTTPServiceRestartE2EReport{}, err
	}
	afterContext, err := requestJSONNoTest[goncho.ContextResult](reopenedHandler, http.MethodGet, "/v3/workspaces/"+cfg.WorkspaceID+"/peers/"+cfg.PeerID+"/context?query=SQLite%20reopen&session_id="+cfg.SessionKey, nil)
	if err != nil {
		return HTTPServiceRestartE2EReport{}, err
	}

	report := HTTPServiceRestartE2EReport{
		SQLiteRestartVerified:               len(afterSearch.Results) == 1 && gonchoHTTPContainsString(afterContext.PeerCard, profileFact) && gonchoHTTPContainsString(afterContext.Conclusions, conclusion) && gonchoHTTPMessageSlicesContain(afterContext.RecentMessages, userMessage),
		NetworkRequired:                     false,
		ExternalProviderRequired:            false,
		MessagesCreated:                     len(messages.Messages),
		SearchCountBeforeRestart:            len(beforeSearch.Results),
		SearchCountAfterRestart:             len(afterSearch.Results),
		ContextHadProfileAfterRestart:       gonchoHTTPContainsString(afterContext.PeerCard, profileFact),
		ContextHadConclusionAfterRestart:    gonchoHTTPContainsString(afterContext.Conclusions, conclusion),
		ContextHadRecentMessageAfterRestart: gonchoHTTPMessageSlicesContain(afterContext.RecentMessages, userMessage),
		CompletionCondition:                 "go test ./...",
	}
	return report, nil
}

func (cfg HTTPServiceRestartE2EConfig) withDefaults() HTTPServiceRestartE2EConfig {
	if cfg.WorkspaceID == "" {
		cfg.WorkspaceID = "http-restart-workspace"
	}
	if cfg.ObserverID == "" {
		cfg.ObserverID = "assistant"
	}
	if cfg.PeerID == "" {
		cfg.PeerID = "peer-http-restart"
	}
	if cfg.SessionKey == "" {
		cfg.SessionKey = "session-http-restart"
	}
	return cfg
}

func putJSONNoTest(handler http.Handler, path string, body any) error {
	_, err := requestJSONNoTest[map[string]any](handler, http.MethodPut, path, body)
	return err
}

func requestJSONNoTest[T any](handler http.Handler, method, path string, body any) (T, error) {
	var zero T
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			return zero, err
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code > 299 {
		return zero, fmt.Errorf("%s %s status = %d; body: %s", method, path, rec.Code, rec.Body.String())
	}
	if rec.Body.Len() == 0 {
		return zero, nil
	}
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		return zero, fmt.Errorf("decode %s %s response: %w", method, path, err)
	}
	return out, nil
}

func gonchoHTTPContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func gonchoHTTPMessageSlicesContain(messages []goncho.MessageSlice, content string) bool {
	for _, message := range messages {
		if message.Content == content {
			return true
		}
	}
	return false
}
