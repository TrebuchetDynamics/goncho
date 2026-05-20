package gonchohttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	goncho "github.com/TrebuchetDynamics/goncho"
)

// NewServiceHandler exposes a small local HTTP adapter over service-backed
// Honcho-compatible routes. It is intended for embedded/local smoke tests, not
// hosted auth, pagination, or provider orchestration.
func NewServiceHandler(svc *goncho.Service) http.Handler {
	return serviceHandler{svc: svc}
}

type serviceHandler struct {
	svc *goncho.Service
}

func (h serviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeHTTPError(w, http.StatusInternalServerError, "goncho http: nil service")
		return
	}
	parts := splitPath(r.URL.Path)
	if len(parts) < 3 || parts[0] != "v3" || parts[1] != "workspaces" {
		writeHTTPError(w, http.StatusNotFound, "goncho http: route not found")
		return
	}

	workspaceID := parts[2]
	_ = workspaceID // The bound service owns workspace selection for this local adapter.
	rest := parts[3:]

	switch {
	case r.Method == http.MethodPut && len(rest) == 3 && rest[0] == "peers" && rest[2] == "card":
		h.handleSetPeerCard(w, r, rest[1])
	case r.Method == http.MethodGet && len(rest) == 3 && rest[0] == "peers" && rest[2] == "context":
		h.handlePeerContext(w, r, rest[1])
	case r.Method == http.MethodPost && len(rest) == 3 && rest[0] == "peers" && rest[2] == "search":
		h.handlePeerSearch(w, r, rest[1])
	case r.Method == http.MethodPost && len(rest) == 3 && rest[0] == "peers" && rest[2] == "chat":
		h.handlePeerChat(w, r, rest[1])
	case r.Method == http.MethodPost && len(rest) == 3 && rest[0] == "sessions" && rest[2] == "messages":
		h.handleCreateMessages(w, r, rest[1])
	case r.Method == http.MethodPost && len(rest) == 1 && rest[0] == "conclusions":
		h.handleConclude(w, r)
	default:
		writeHTTPError(w, http.StatusNotFound, "goncho http: route not found")
	}
}

func (h serviceHandler) handleSetPeerCard(w http.ResponseWriter, r *http.Request, peer string) {
	var body struct {
		ProfileID string   `json:"profile_id"`
		Card      []string `json:"card"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	profileID := strings.TrimSpace(body.ProfileID)
	if profileID != "" {
		if err := h.svc.SetProfileInNamespace(r.Context(), goncho.MemoryNamespace{ProfileID: profileID, PeerID: peer, Scope: goncho.MemoryScopeProfile}, body.Card); err != nil {
			writeHTTPError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := h.svc.ProfileInNamespace(r.Context(), goncho.MemoryNamespace{ProfileID: profileID, PeerID: peer, Scope: goncho.MemoryScopeProfile})
		if err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if err := h.svc.SetProfile(r.Context(), peer, body.Card); err != nil {
		writeHTTPError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.Profile(r.Context(), peer)
	if err != nil {
		writeHTTPError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h serviceHandler) handleCreateMessages(w http.ResponseWriter, r *http.Request, sessionKey string) {
	var body struct {
		Messages []goncho.CreateMessage `json:"messages"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	result, err := h.svc.CreateMessages(r.Context(), goncho.CreateMessagesParams{
		SessionKey: sessionKey,
		Messages:   body.Messages,
	})
	if err != nil {
		writeHTTPError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h serviceHandler) handleConclude(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProfileID  string `json:"profile_id"`
		Peer       string `json:"peer"`
		PeerID     string `json:"peer_id"`
		Conclusion string `json:"conclusion"`
		SessionKey string `json:"session_key"`
		Scope      string `json:"scope"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	peer := firstNonEmpty(body.Peer, body.PeerID)
	result, err := h.svc.Conclude(r.Context(), goncho.ConcludeParams{
		ProfileID:  body.ProfileID,
		Peer:       peer,
		Conclusion: body.Conclusion,
		SessionKey: body.SessionKey,
		Scope:      body.Scope,
	})
	if err != nil {
		writeHTTPError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h serviceHandler) handlePeerContext(w http.ResponseWriter, r *http.Request, peer string) {
	query := r.URL.Query()
	result, err := h.svc.Context(r.Context(), goncho.ContextParams{
		ProfileID:  query.Get("profile_id"),
		Peer:       peer,
		Query:      firstNonEmpty(query.Get("query"), query.Get("search_query")),
		SessionKey: firstNonEmpty(query.Get("session_key"), query.Get("session_id")),
		Scope:      query.Get("scope"),
	})
	if err != nil {
		writeHTTPError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h serviceHandler) handlePeerSearch(w http.ResponseWriter, r *http.Request, peer string) {
	var body goncho.SearchParams
	if !decodeJSONBody(w, r, &body) {
		return
	}
	body.Peer = peer
	result, err := h.svc.Search(r.Context(), body)
	if err != nil {
		writeHTTPError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h serviceHandler) handlePeerChat(w http.ResponseWriter, r *http.Request, peer string) {
	var body goncho.ChatParams
	if !decodeJSONBody(w, r, &body) {
		return
	}
	result, err := h.svc.Chat(r.Context(), peer, body)
	if err != nil {
		writeHTTPError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, out any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		writeHTTPError(w, http.StatusBadRequest, fmt.Sprintf("goncho http: decode json: %v", err))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeHTTPError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
