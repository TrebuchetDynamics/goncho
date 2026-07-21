package gonchohttp

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

// serveMemoryFacadeHTTP exposes Goncho's evidence-backed MemoryFacade through
// the compact add/search/get/update/delete/history shape popularized by mem0.
func (h serviceHandler) serveMemoryFacadeHTTP(w http.ResponseWriter, r *http.Request, parts []string) bool {
	if len(parts) < 2 || parts[0] != "v1" || parts[1] != "memories" {
		return false
	}
	facade := goncho.NewMemoryFacade(h.svc)
	switch {
	case r.Method == http.MethodPost && len(parts) == 2:
		var params goncho.MemoryAddParams
		if !decodeJSONBody(w, r, &params) {
			return true
		}
		if !requireMemoryUserID(w, params.UserID) {
			return true
		}
		params.WorkspaceID = ""
		item, err := facade.Add(r.Context(), params)
		writeMemoryFacadeResult(w, http.StatusCreated, item, err)
	case r.Method == http.MethodGet && len(parts) == 2:
		params := memorySearchQuery(r)
		if !requireMemoryUserID(w, params.UserID) {
			return true
		}
		result, err := facade.Search(r.Context(), params)
		writeMemoryFacadeResult(w, http.StatusOK, result, err)
	case r.Method == http.MethodPost && len(parts) == 3 && parts[2] == "search":
		var params goncho.MemorySearchParams
		if !decodeJSONBody(w, r, &params) {
			return true
		}
		if !requireMemoryUserID(w, params.UserID) {
			return true
		}
		params.WorkspaceID = ""
		result, err := facade.Search(r.Context(), params)
		writeMemoryFacadeResult(w, http.StatusOK, result, err)
	case r.Method == http.MethodGet && len(parts) == 3:
		params := memoryGetQuery(r, parts[2])
		if !requireMemoryUserID(w, params.UserID) {
			return true
		}
		item, err := facade.Get(r.Context(), params)
		writeMemoryFacadeResult(w, http.StatusOK, item, err)
	case r.Method == http.MethodPut && len(parts) == 3:
		var params goncho.MemoryUpdateParams
		if !decodeJSONBody(w, r, &params) {
			return true
		}
		if !requireMemoryUserID(w, params.UserID) {
			return true
		}
		params.ID, params.WorkspaceID = parts[2], ""
		item, err := facade.Update(r.Context(), params)
		writeMemoryFacadeResult(w, http.StatusOK, item, err)
	case r.Method == http.MethodDelete && len(parts) == 3:
		query := r.URL.Query()
		params := goncho.MemoryDeleteParams{ID: parts[2], UserID: query.Get("user_id"), AgentID: query.Get("agent_id"), RunID: query.Get("run_id"), SessionKey: query.Get("session_key"), ProfileID: query.Get("profile_id")}
		if !requireMemoryUserID(w, params.UserID) {
			return true
		}
		item, err := facade.Delete(r.Context(), params)
		writeMemoryFacadeResult(w, http.StatusOK, item, err)
	case r.Method == http.MethodGet && len(parts) == 4 && parts[3] == "history":
		query := r.URL.Query()
		limit, _ := strconv.Atoi(query.Get("limit"))
		params := goncho.MemoryHistoryParams{ID: parts[2], UserID: query.Get("user_id"), ProfileID: query.Get("profile_id"), Limit: limit}
		if !requireMemoryUserID(w, params.UserID) {
			return true
		}
		result, err := facade.History(r.Context(), params)
		writeMemoryFacadeResult(w, http.StatusOK, result, err)
	default:
		writeHTTPError(w, http.StatusNotFound, "goncho http: route not found")
	}
	return true
}

func requireMemoryUserID(w http.ResponseWriter, userID string) bool {
	if strings.TrimSpace(userID) != "" {
		return true
	}
	writeHTTPError(w, http.StatusBadRequest, "goncho http: user_id is required")
	return false
}

func memoryGetQuery(r *http.Request, id string) goncho.MemoryGetParams {
	query := r.URL.Query()
	return goncho.MemoryGetParams{ID: id, UserID: query.Get("user_id"), ProfileID: query.Get("profile_id")}
}

func memorySearchQuery(r *http.Request) goncho.MemorySearchParams {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	return goncho.MemorySearchParams{UserID: query.Get("user_id"), AgentID: query.Get("agent_id"), RunID: query.Get("run_id"), SessionKey: query.Get("session_key"), ProfileID: query.Get("profile_id"), Query: query.Get("query"), Limit: limit}
}

func writeMemoryFacadeResult(w http.ResponseWriter, status int, result any, err error) {
	if err == nil {
		writeJSON(w, status, result)
		return
	}
	if errors.Is(err, goncho.ErrMemoryNotFound) {
		writeHTTPError(w, http.StatusNotFound, err.Error())
		return
	}
	if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "not found") {
		writeHTTPError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeHTTPError(w, http.StatusInternalServerError, err.Error())
}
