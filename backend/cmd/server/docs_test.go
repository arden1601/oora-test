package main

// Tests for the API documentation. They keep the OpenAPI file honest: every
// route in it must exist on the server, and every route we serve must be in it.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Every route the server offers. A new route must be added here and to
// backend/api/openapi.json, or TestEveryRouteIsDocumented fails.
var servedRoutes = []string{
	"GET /healthz",
	"POST /login",
	"POST /logout",
	"GET /me",
	"GET /bookings",
	"GET /patients",
	"GET /medical-record",
	"GET /revenue",
	"GET /sales-entry",
	"POST /sales-entry",
	"GET /admin/staff",
	"GET /admin/staff/{id}/access",
	"POST /admin/staff/{id}/deactivate",
	"POST /admin/grants",
	"DELETE /admin/grants/{id}",
}

func TestDocsPageIsServed(t *testing.T) {
	rec := httptest.NewRecorder()
	newMux(nil).ServeHTTP(rec, httptest.NewRequest("GET", "/docs", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /docs: got %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "/openapi.json") {
		t.Fatal("the docs page does not load /openapi.json")
	}
}

func TestEveryRouteIsDocumented(t *testing.T) {
	documented := documentedRoutes(t)
	for _, route := range servedRoutes {
		if !documented[route] {
			t.Errorf("%s is served but missing from openapi.json", route)
		}
	}
}

func TestEveryDocumentedRouteIsServed(t *testing.T) {
	served := map[string]bool{}
	for _, route := range servedRoutes {
		served[route] = true
	}
	mux := newMux(nil)
	for route := range documentedRoutes(t) {
		if !served[route] {
			t.Errorf("%s is in openapi.json but not in the list of served routes", route)
			continue
		}
		// Ask the router itself which pattern would handle this request.
		method, path, _ := strings.Cut(route, " ")
		path = strings.ReplaceAll(path, "{id}", "00000000-0000-0000-0000-000000000000")
		_, pattern := mux.Handler(httptest.NewRequest(method, path, nil))
		if pattern != route {
			t.Errorf("%s is documented, but the server routes it to %q", route, pattern)
		}
	}
}

// documentedRoutes reads the served OpenAPI file and lists its "METHOD /path" pairs.
func documentedRoutes(t *testing.T) map[string]bool {
	t.Helper()
	rec := httptest.NewRecorder()
	newMux(nil).ServeHTTP(rec, httptest.NewRequest("GET", "/openapi.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /openapi.json: got %d, want 200", rec.Code)
	}
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("openapi.json is not valid JSON: %v", err)
	}
	if len(spec.Paths) == 0 {
		t.Fatal("openapi.json lists no paths")
	}
	routes := map[string]bool{}
	for path, methods := range spec.Paths {
		for method := range methods {
			routes[strings.ToUpper(method)+" "+path] = true
		}
	}
	return routes
}
