package gonchohttp

import (
	"slices"
	"strings"
)

type RouteStatus string

const (
	RouteServiceBacked RouteStatus = "service_backed"
	RouteRowBacked     RouteStatus = "row_backed"
)

type Route struct {
	Method      string      `json:"method"`
	Path        string      `json:"path"`
	OperationID string      `json:"operation_id"`
	Tag         string      `json:"tag"`
	Status      RouteStatus `json:"status"`
	Target      string      `json:"target,omitempty"`
	Residual    string      `json:"residual,omitempty"`
	SourceRefs  []string    `json:"source_refs,omitempty"`
}

func OpenAPIRouteManifest() []Route {
	routes := []Route{
		route("POST", "/v3/workspaces", "get_or_create_workspace_v3_workspaces_post", "workspaces", RouteRowBacked, "", "workspace route binding, body validation, pagination, and hosted auth policy remain row-backed", "routers/workspaces.py"),
		route("POST", "/v3/workspaces/list", "get_all_workspaces_v3_workspaces_list_post", "workspaces", RouteRowBacked, "", "workspace list pagination and filter semantics remain row-backed", "routers/workspaces.py"),
		route("PUT", "/v3/workspaces/{workspace_id}", "update_workspace_v3_workspaces__workspace_id__put", "workspaces", RouteRowBacked, "", "workspace update configuration semantics remain row-backed", "routers/workspaces.py"),
		route("DELETE", "/v3/workspaces/{workspace_id}", "delete_workspace_v3_workspaces__workspace_id__delete", "workspaces", RouteServiceBacked, "internal/goncho.Service.DeleteWorkspace", "HTTP status codes, auth, and route response shape remain row-backed", "routers/workspaces.py", "crud/workspace.py"),
		route("POST", "/v3/workspaces/{workspace_id}/search", "search_workspace_v3_workspaces__workspace_id__search_post", "workspaces", RouteServiceBacked, "internal/goncho.Service.Search", "workspace-level HTTP search pagination and validation remain row-backed", "routers/workspaces.py"),
		route("GET", "/v3/workspaces/{workspace_id}/queue/status", "get_queue_status_v3_workspaces__workspace_id__queue_status_get", "workspaces", RouteServiceBacked, "internal/goncho.ReadQueueStatus", "OpenAPI route binding and queue page shape remain row-backed", "routers/workspaces.py"),
		route("POST", "/v3/workspaces/{workspace_id}/schedule_dream", "schedule_dream_v3_workspaces__workspace_id__schedule_dream_post", "workspaces", RouteServiceBacked, "internal/goncho.Service.ScheduleDream", "OpenAPI route binding and dream worker execution remain row-backed", "routers/workspaces.py"),

		route("POST", "/v3/workspaces/{workspace_id}/peers/list", "get_peers_v3_workspaces__workspace_id__peers_list_post", "peers", RouteRowBacked, "", "peer list pagination and filters remain row-backed", "routers/peers.py"),
		route("POST", "/v3/workspaces/{workspace_id}/peers", "get_or_create_peer_v3_workspaces__workspace_id__peers_post", "peers", RouteRowBacked, "", "peer creation route shape and metadata semantics remain row-backed", "routers/peers.py"),
		route("PUT", "/v3/workspaces/{workspace_id}/peers/{peer_id}", "update_peer_v3_workspaces__workspace_id__peers__peer_id__put", "peers", RouteRowBacked, "", "peer update route semantics remain row-backed", "routers/peers.py"),
		route("POST", "/v3/workspaces/{workspace_id}/peers/{peer_id}/sessions", "get_sessions_for_peer_v3_workspaces__workspace_id__peers__peer_id__sessions_post", "peers", RouteRowBacked, "", "peer session listing and pagination remain row-backed", "routers/peers.py"),
		route("POST", "/v3/workspaces/{workspace_id}/peers/{peer_id}/chat", "chat_v3_workspaces__workspace_id__peers__peer_id__chat_post", "peers", RouteServiceBacked, "internal/goncho.Service.Chat", "streaming route binding, auth, and provider execution remain row-backed", "routers/peers.py"),
		route("POST", "/v3/workspaces/{workspace_id}/peers/{peer_id}/representation", "get_representation_v3_workspaces__workspace_id__peers__peer_id__representation_post", "peers", RouteServiceBacked, "internal/goncho.Service.Context", "representation-specific route shape remains row-backed", "routers/peers.py"),
		route("GET", "/v3/workspaces/{workspace_id}/peers/{peer_id}/card", "get_peer_card_v3_workspaces__workspace_id__peers__peer_id__card_get", "peers", RouteServiceBacked, "internal/goncho.Service.Profile", "OpenAPI route binding and empty-card response contract remain row-backed", "routers/peers.py"),
		route("PUT", "/v3/workspaces/{workspace_id}/peers/{peer_id}/card", "set_peer_card_v3_workspaces__workspace_id__peers__peer_id__card_put", "peers", RouteServiceBacked, "internal/goncho.Service.SetProfile", "OpenAPI route binding and validation remain row-backed", "routers/peers.py"),
		route("GET", "/v3/workspaces/{workspace_id}/peers/{peer_id}/context", "get_peer_context_v3_workspaces__workspace_id__peers__peer_id__context_get", "peers", RouteServiceBacked, "internal/goncho.Service.Context", "query parameter validation and HTTP response shape remain row-backed", "routers/peers.py"),
		route("POST", "/v3/workspaces/{workspace_id}/peers/{peer_id}/search", "search_peer_v3_workspaces__workspace_id__peers__peer_id__search_post", "peers", RouteServiceBacked, "internal/goncho.Service.Search", "HTTP route binding and pagination remain row-backed", "routers/peers.py"),

		route("POST", "/v3/workspaces/{workspace_id}/sessions/list", "get_sessions_v3_workspaces__workspace_id__sessions_list_post", "sessions", RouteRowBacked, "", "session list filters and pagination remain row-backed", "routers/sessions.py"),
		route("POST", "/v3/workspaces/{workspace_id}/sessions", "get_or_create_session_v3_workspaces__workspace_id__sessions_post", "sessions", RouteRowBacked, "", "session creation/update route shape remains row-backed", "routers/sessions.py"),
		route("PUT", "/v3/workspaces/{workspace_id}/sessions/{session_id}", "update_session_v3_workspaces__workspace_id__sessions__session_id__put", "sessions", RouteRowBacked, "", "session update metadata and configuration remain row-backed", "routers/sessions.py"),
		route("DELETE", "/v3/workspaces/{workspace_id}/sessions/{session_id}", "delete_session_v3_workspaces__workspace_id__sessions__session_id__delete", "sessions", RouteServiceBacked, "internal/goncho.Service.DeleteSession", "HTTP status code and route response shape remain row-backed", "routers/sessions.py", "crud/session.py"),
		route("POST", "/v3/workspaces/{workspace_id}/sessions/{session_id}/clone", "clone_session_v3_workspaces__workspace_id__sessions__session_id__clone_post", "sessions", RouteRowBacked, "", "session clone semantics remain row-backed", "routers/sessions.py"),
		route("POST", "/v3/workspaces/{workspace_id}/sessions/{session_id}/peers", "add_peers_to_session_v3_workspaces__workspace_id__sessions__session_id__peers_post", "sessions", RouteRowBacked, "", "session-peer add route remains row-backed", "routers/sessions.py"),
		route("PUT", "/v3/workspaces/{workspace_id}/sessions/{session_id}/peers", "set_session_peers_v3_workspaces__workspace_id__sessions__session_id__peers_put", "sessions", RouteRowBacked, "", "session-peer replacement route remains row-backed", "routers/sessions.py"),
		route("DELETE", "/v3/workspaces/{workspace_id}/sessions/{session_id}/peers", "remove_peers_from_session_v3_workspaces__workspace_id__sessions__session_id__peers_delete", "sessions", RouteRowBacked, "", "session-peer removal route remains row-backed", "routers/sessions.py"),
		route("GET", "/v3/workspaces/{workspace_id}/sessions/{session_id}/peers", "get_session_peers_v3_workspaces__workspace_id__sessions__session_id__peers_get", "sessions", RouteRowBacked, "", "session-peer listing route remains row-backed", "routers/sessions.py"),
		route("GET", "/v3/workspaces/{workspace_id}/sessions/{session_id}/peers/{peer_id}/config", "get_peer_config_v3_workspaces__workspace_id__sessions__session_id__peers__peer_id__config_get", "sessions", RouteRowBacked, "", "session peer config read route remains row-backed", "routers/sessions.py"),
		route("PUT", "/v3/workspaces/{workspace_id}/sessions/{session_id}/peers/{peer_id}/config", "set_peer_config_v3_workspaces__workspace_id__sessions__session_id__peers__peer_id__config_put", "sessions", RouteRowBacked, "", "session peer config write route remains row-backed", "routers/sessions.py"),
		route("GET", "/v3/workspaces/{workspace_id}/sessions/{session_id}/context", "get_session_context_v3_workspaces__workspace_id__sessions__session_id__context_get", "sessions", RouteServiceBacked, "internal/goncho.Service.Context", "HTTP query shape and session-scoped validation remain row-backed", "routers/sessions.py"),
		route("GET", "/v3/workspaces/{workspace_id}/sessions/{session_id}/summaries", "get_session_summaries_v3_workspaces__workspace_id__sessions__session_id__summaries_get", "sessions", RouteServiceBacked, "internal/goncho session summary store", "HTTP response shape remains row-backed", "routers/sessions.py"),
		route("POST", "/v3/workspaces/{workspace_id}/sessions/{session_id}/search", "search_session_v3_workspaces__workspace_id__sessions__session_id__search_post", "sessions", RouteServiceBacked, "internal/goncho.Service.Search", "HTTP route binding and pagination remain row-backed", "routers/sessions.py"),

		route("POST", "/v3/workspaces/{workspace_id}/sessions/{session_id}/messages", "create_messages_for_session_v3_workspaces__workspace_id__sessions__session_id__messages_post", "messages", RouteServiceBacked, "internal/goncho.Service.CreateMessages", "HTTP route binding, body aliases, and pagination remain row-backed", "routers/messages.py", "crud/message.py"),
		route("POST", "/v3/workspaces/{workspace_id}/sessions/{session_id}/messages/upload", "create_messages_with_file_v3_workspaces__workspace_id__sessions__session_id__messages_upload_post", "messages", RouteRowBacked, "", "multipart file upload route remains row-backed", "routers/messages.py"),
		route("POST", "/v3/workspaces/{workspace_id}/sessions/{session_id}/messages/list", "get_messages_v3_workspaces__workspace_id__sessions__session_id__messages_list_post", "messages", RouteServiceBacked, "internal/goncho lifecycle message store", "HTTP pagination and filter route remain row-backed", "routers/messages.py"),
		route("GET", "/v3/workspaces/{workspace_id}/sessions/{session_id}/messages/{message_id}", "get_message_v3_workspaces__workspace_id__sessions__session_id__messages__message_id__get", "messages", RouteRowBacked, "", "message get-by-id route remains row-backed", "routers/messages.py"),
		route("PUT", "/v3/workspaces/{workspace_id}/sessions/{session_id}/messages/{message_id}", "update_message_v3_workspaces__workspace_id__sessions__session_id__messages__message_id__put", "messages", RouteRowBacked, "", "message update route remains row-backed", "routers/messages.py"),

		route("POST", "/v3/workspaces/{workspace_id}/conclusions", "create_conclusions_v3_workspaces__workspace_id__conclusions_post", "conclusions", RouteServiceBacked, "internal/goncho.Service.Conclude", "bulk conclusion object route remains row-backed", "routers/conclusions.py"),
		route("POST", "/v3/workspaces/{workspace_id}/conclusions/list", "list_conclusions_v3_workspaces__workspace_id__conclusions_list_post", "conclusions", RouteServiceBacked, "internal/goncho.Service.Search", "pagination and object response shape remain row-backed", "routers/conclusions.py"),
		route("POST", "/v3/workspaces/{workspace_id}/conclusions/query", "query_conclusions_v3_workspaces__workspace_id__conclusions_query_post", "conclusions", RouteServiceBacked, "internal/goncho.Service.Search", "query route body and top_k validation remain row-backed", "routers/conclusions.py"),
		route("DELETE", "/v3/workspaces/{workspace_id}/conclusions/{conclusion_id}", "delete_conclusion_v3_workspaces__workspace_id__conclusions__conclusion_id__delete", "conclusions", RouteServiceBacked, "internal/goncho.Service.Conclude(DeleteID)", "HTTP status code and auth remain row-backed", "routers/conclusions.py"),

		route("POST", "/v3/keys", "create_key_v3_keys_post", "keys", RouteServiceBacked, "internal/goncho.CreateScopedKey", "HTTP admin-auth dependency and status-code mapping remain row-backed", "routers/keys.py", "security.py"),
		route("POST", "/v3/workspaces/{workspace_id}/webhooks", "get_or_create_webhook_endpoint_v3_workspaces__workspace_id__webhooks_post", "webhooks", RouteServiceBacked, "internal/goncho.Service.GetOrCreateWebhookEndpoint", "HTTP route binding and auth dependency remain row-backed", "routers/webhooks.py", "crud/webhook.py"),
		route("GET", "/v3/workspaces/{workspace_id}/webhooks", "list_webhook_endpoints_v3_workspaces__workspace_id__webhooks_get", "webhooks", RouteServiceBacked, "internal/goncho.Service.ListWebhookEndpoints", "HTTP pagination envelope remains row-backed", "routers/webhooks.py", "crud/webhook.py"),
		route("DELETE", "/v3/workspaces/{workspace_id}/webhooks/{endpoint_id}", "delete_webhook_endpoint_v3_workspaces__workspace_id__webhooks__endpoint_id__delete", "webhooks", RouteServiceBacked, "internal/goncho.Service.DeleteWebhookEndpoint", "HTTP 204/error mapping remains row-backed", "routers/webhooks.py", "crud/webhook.py"),
		route("GET", "/v3/workspaces/{workspace_id}/webhooks/test", "test_emit_v3_workspaces__workspace_id__webhooks_test_get", "webhooks", RouteServiceBacked, "internal/goncho.NewTestWebhookEvent", "queue-backed publish and delivery workers remain row-backed", "routers/webhooks.py", "webhooks/events.py"),
	}
	return cloneRoutes(routes)
}

func FindRoute(method, path string) (Route, bool) {
	keyMethod := strings.ToUpper(strings.TrimSpace(method))
	keyPath := strings.TrimSpace(path)
	for _, r := range OpenAPIRouteManifest() {
		if r.Method == keyMethod && r.Path == keyPath {
			return r, true
		}
	}
	return Route{}, false
}

func RowBackedRoutes() []Route {
	var out []Route
	for _, r := range OpenAPIRouteManifest() {
		if r.Status == RouteRowBacked {
			out = append(out, r)
		}
	}
	return cloneRoutes(out)
}

func route(method, path, operationID, tag string, status RouteStatus, target, residual string, refs ...string) Route {
	out := Route{
		Method:      strings.ToUpper(method),
		Path:        path,
		OperationID: operationID,
		Tag:         tag,
		Status:      status,
		Target:      target,
		Residual:    residual,
	}
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref != "" {
			out.SourceRefs = append(out.SourceRefs, "../honcho/src/"+ref)
		}
	}
	return out
}

func cloneRoutes(in []Route) []Route {
	out := make([]Route, len(in))
	for i, r := range in {
		out[i] = r
		out[i].SourceRefs = slices.Clone(r.SourceRefs)
	}
	return out
}
