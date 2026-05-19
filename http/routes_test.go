package gonchohttp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestOpenAPIRouteManifestMatchesHonchoV3OpenAPI(t *testing.T) {
	upstream := readUpstreamOpenAPIRoutes(t)
	manifest := OpenAPIRouteManifest()

	got := routeSignatures(manifest)
	want := routeSignatures(upstream)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("route manifest mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestOpenAPIRouteManifestClassifiesEveryRoute(t *testing.T) {
	routes := OpenAPIRouteManifest()
	if len(routes) != 45 {
		t.Fatalf("routes len = %d, want 45", len(routes))
	}
	seen := map[string]bool{}
	for _, r := range routes {
		key := r.Method + " " + r.Path
		if seen[key] {
			t.Fatalf("duplicate route %s", key)
		}
		seen[key] = true
		if r.OperationID == "" || r.Tag == "" {
			t.Fatalf("route %s missing operation/tag: %+v", key, r)
		}
		if r.Status != RouteServiceBacked && r.Status != RouteRowBacked {
			t.Fatalf("route %s status = %q, want classified", key, r.Status)
		}
		if len(r.SourceRefs) == 0 {
			t.Fatalf("route %s missing source refs", key)
		}
		if r.Status == RouteServiceBacked && r.Target == "" {
			t.Fatalf("route %s service-backed without target", key)
		}
		if r.Residual == "" {
			t.Fatalf("route %s missing residual contract", key)
		}
	}
}

func TestOpenAPIRouteManifestKeysWebhooksAndCrudAreServiceBacked(t *testing.T) {
	for _, tc := range []struct {
		method string
		path   string
		target string
	}{
		{"POST", "/v3/keys", "CreateScopedKey"},
		{"POST", "/v3/workspaces/{workspace_id}/webhooks", "GetOrCreateWebhookEndpoint"},
		{"GET", "/v3/workspaces/{workspace_id}/webhooks", "ListWebhookEndpoints"},
		{"DELETE", "/v3/workspaces/{workspace_id}/webhooks/{endpoint_id}", "DeleteWebhookEndpoint"},
		{"GET", "/v3/workspaces/{workspace_id}/webhooks/test", "NewTestWebhookEvent"},
		{"POST", "/v3/workspaces/{workspace_id}/sessions/{session_id}/messages", "CreateMessages"},
		{"DELETE", "/v3/workspaces/{workspace_id}/sessions/{session_id}", "DeleteSession"},
		{"DELETE", "/v3/workspaces/{workspace_id}", "DeleteWorkspace"},
	} {
		route, ok := FindRoute(tc.method, tc.path)
		if !ok {
			t.Fatalf("missing route %s %s", tc.method, tc.path)
		}
		if route.Status != RouteServiceBacked {
			t.Fatalf("%s %s status = %s, want service_backed", tc.method, tc.path, route.Status)
		}
		if !strings.Contains(route.Target, tc.target) {
			t.Fatalf("%s %s target = %q, want %q", tc.method, tc.path, route.Target, tc.target)
		}
	}
}

func TestOpenAPIRouteManifestReturnsDefensiveCopies(t *testing.T) {
	routes := OpenAPIRouteManifest()
	routes[0].Path = "/mutated"
	routes[0].SourceRefs[0] = "mutated"

	again := OpenAPIRouteManifest()
	if again[0].Path == "/mutated" || again[0].SourceRefs[0] == "mutated" {
		t.Fatalf("manifest was mutated through returned slice: %+v", again[0])
	}

	rowBacked := RowBackedRoutes()
	if len(rowBacked) == 0 {
		t.Fatal("expected row-backed routes")
	}
	rowBacked[0].Path = "/mutated-row-backed"
	if later := RowBackedRoutes(); later[0].Path == "/mutated-row-backed" {
		t.Fatalf("row-backed route slice was mutable: %+v", later[0])
	}
}

type openAPIDoc struct {
	Paths map[string]map[string]openAPIOperation `json:"paths"`
}

type openAPIOperation struct {
	OperationID string   `json:"operationId"`
	Tags        []string `json:"tags"`
}

func readUpstreamOpenAPIRoutes(t *testing.T) []Route {
	t.Helper()
	path := filepath.Clean("../../../../honcho/docs/v3/openapi.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("upstream Honcho OpenAPI fixture not present at %s", path)
		}
		t.Fatalf("read upstream OpenAPI: %v", err)
	}
	var doc openAPIDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode upstream OpenAPI: %v", err)
	}
	var routes []Route
	for path, methods := range doc.Paths {
		for method, op := range methods {
			method = strings.ToUpper(method)
			if !isHTTPMethod(method) {
				continue
			}
			tag := ""
			if len(op.Tags) > 0 {
				tag = op.Tags[0]
			}
			routes = append(routes, Route{
				Method:      method,
				Path:        path,
				OperationID: op.OperationID,
				Tag:         tag,
			})
		}
	}
	return routes
}

func routeSignatures(routes []Route) []string {
	out := make([]string, 0, len(routes))
	for _, r := range routes {
		out = append(out, r.Method+"\t"+r.Path+"\t"+r.OperationID+"\t"+r.Tag)
	}
	sort.Strings(out)
	return out
}

func isHTTPMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}
