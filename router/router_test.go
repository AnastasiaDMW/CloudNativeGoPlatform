package router

import (
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouterRoutesAndMetrics(t *testing.T) {
	r := Router()

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/healthz"},
		{"GET", "/readyz"},
		{"GET", "/api/user/123"},
		{"OPTIONS", "/api/user/123"},
		{"GET", "/api/user"},
		{"OPTIONS", "/api/user"},
		{"POST", "/api/newuser"},
		{"OPTIONS", "/api/newuser"},
		{"PUT", "/api/user/123"},
		{"OPTIONS", "/api/user/123"},
		{"DELETE", "/api/user/123"},
		{"OPTIONS", "/api/user/123"},
		{"GET", "/metrics"},
	}

	for _, route := range routes {
		req := httptest.NewRequest(route.method, route.path, nil)

		if !r.Match(req, &mux.RouteMatch{}) {
			t.Errorf("route not found: %s %s", route.method, route.path)
		}
	}

	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRec := httptest.NewRecorder()
	r.ServeHTTP(healthRec, healthReq)

	if healthRec.Code != http.StatusOK {
		t.Errorf("healthz: expected status %d, got %d",
			http.StatusOK, healthRec.Code)
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	r.ServeHTTP(metricsRec, metricsReq)

	if metricsRec.Code != http.StatusOK {
		t.Errorf("metrics: expected status %d, got %d",
			http.StatusOK, metricsRec.Code)
	}

	if !strings.Contains(metricsRec.Body.String(), "http_requests_total") {
		t.Error("metrics response does not contain http_requests_total")
	}
}
