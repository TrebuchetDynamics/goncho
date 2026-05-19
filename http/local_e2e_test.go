package gonchohttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goncho "github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/goncho/memory"
)

func TestLocalE2E_HTTPServiceLifecycleUsesHonchoCompatibleRoutes(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer func() {
		if err := store.Close(ctx); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	workspace := "http-e2e-workspace"
	peer := "telegram:6586915095"
	session := "http-e2e-session"
	profileFact := "Prefers HTTP local smoke tests."
	userMessage := "Please verify Goncho through local HTTP routes."
	conclusion := "Goncho HTTP local E2E uses in-process httptest only."

	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: workspace, ObserverPeerID: "assistant"}, nil)
	handler := NewServiceHandler(svc)

	putJSON(t, handler, "/v3/workspaces/"+workspace+"/peers/"+peer+"/card", map[string]any{
		"card": []string{profileFact},
	}, http.StatusOK)

	messages := postJSON[goncho.CreateMessagesResult](t, handler, "/v3/workspaces/"+workspace+"/sessions/"+session+"/messages", map[string]any{
		"messages": []map[string]any{
			{"peer_id": peer, "role": "user", "content": userMessage},
			{"peer_id": "assistant", "role": "assistant", "content": "I will verify the HTTP route lifecycle."},
		},
	}, http.StatusOK)
	if len(messages.Messages) != 2 || messages.Messages[0].Sequence != 1 || messages.Messages[1].Sequence != 2 {
		t.Fatalf("messages result = %+v, want two sequenced messages", messages)
	}

	postJSON[goncho.ConcludeResult](t, handler, "/v3/workspaces/"+workspace+"/conclusions", map[string]any{
		"peer_id":     peer,
		"conclusion":  conclusion,
		"session_key": session,
	}, http.StatusOK)

	contextResult := getJSON[goncho.ContextResult](t, handler, "/v3/workspaces/"+workspace+"/peers/"+peer+"/context?query=httptest&session_id="+session, http.StatusOK)
	if !containsString(contextResult.PeerCard, profileFact) {
		t.Fatalf("context peer card = %#v, want %q", contextResult.PeerCard, profileFact)
	}
	if !containsString(contextResult.Conclusions, conclusion) {
		t.Fatalf("context conclusions = %#v, want %q", contextResult.Conclusions, conclusion)
	}
	if !messageSlicesContain(contextResult.RecentMessages, userMessage) {
		t.Fatalf("context recent messages = %#v, want %q", contextResult.RecentMessages, userMessage)
	}

	searchResult := postJSON[goncho.SearchResultSet](t, handler, "/v3/workspaces/"+workspace+"/peers/"+peer+"/search", map[string]any{
		"query":       "httptest",
		"session_key": session,
	}, http.StatusOK)
	if !searchHitsContainSourceContent(searchResult.Results, "conclusion", conclusion) {
		t.Fatalf("search results = %#v, want conclusion %q", searchResult.Results, conclusion)
	}

	chatResult := postJSON[goncho.ChatResult](t, handler, "/v3/workspaces/"+workspace+"/peers/"+peer+"/chat", map[string]any{
		"query":      "How should I test Goncho HTTP locally?",
		"session_id": session,
	}, http.StatusOK)
	for _, want := range []string{"Query: How should I test Goncho HTTP locally?", "Reasoning level: low", conclusion} {
		if !strings.Contains(chatResult.Content, want) {
			t.Fatalf("chat content missing %q in %q", want, chatResult.Content)
		}
	}
}

func putJSON(t *testing.T, handler http.Handler, path string, body any, wantStatus int) {
	t.Helper()
	_ = requestJSON[map[string]any](t, handler, http.MethodPut, path, body, wantStatus)
}

func postJSON[T any](t *testing.T, handler http.Handler, path string, body any, wantStatus int) T {
	t.Helper()
	return requestJSON[T](t, handler, http.MethodPost, path, body, wantStatus)
}

func getJSON[T any](t *testing.T, handler http.Handler, path string, wantStatus int) T {
	t.Helper()
	return requestJSON[T](t, handler, http.MethodGet, path, nil, wantStatus)
}

func requestJSON[T any](t *testing.T, handler http.Handler, method, path string, body any, wantStatus int) T {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d; body: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	var out T
	if rec.Body.Len() == 0 {
		return out
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response for %s %s: %v\n%s", method, path, err, rec.Body.String())
	}
	return out
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func messageSlicesContain(messages []goncho.MessageSlice, content string) bool {
	for _, message := range messages {
		if message.Content == content {
			return true
		}
	}
	return false
}

func searchHitsContainSourceContent(hits []goncho.SearchHit, source, content string) bool {
	for _, hit := range hits {
		if hit.Source == source && hit.Content == content {
			return true
		}
	}
	return false
}
