package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestDocumentationRoutes(t *testing.T) {
	e := echo.New()
	SetupRoutes(e, &CheckHandler{}, NewAuthHandler(&fakeAuthService{}))

	tests := []struct {
		path     string
		wantCode int
		contains string
	}{
		{path: "/swagger", wantCode: http.StatusFound, contains: "/swagger/index.html"},
		{path: "/swagger/", wantCode: http.StatusFound, contains: "/swagger/index.html"},
		{path: "/swagger/index.html", wantCode: http.StatusOK, contains: "SwaggerUIBundle"},
		{path: "/swagger/doc.json", wantCode: http.StatusOK, contains: "/api/v1/auth/login"},
	}
	for _, tc := range tests {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != tc.wantCode || !strings.Contains(rec.Body.String()+rec.Header().Get("Location"), tc.contains) {
			t.Fatalf("GET %s: code=%d body=%s", tc.path, rec.Code, rec.Body.String())
		}
	}
}

func TestSetupRoutesRegistersEveryAPIEndpoint(t *testing.T) {
	e := echo.New()
	SetupRoutes(e, &CheckHandler{}, NewAuthHandler(&fakeAuthService{}))

	want := map[string]bool{
		http.MethodGet + " /health":                      false,
		http.MethodGet + " /ready":                       false,
		http.MethodPost + " /api/v1/auth/register":       false,
		http.MethodPost + " /api/v1/auth/login":          false,
		http.MethodGet + " /api/v1/auth/google/login":    false,
		http.MethodGet + " /api/v1/auth/google/callback": false,
		http.MethodPost + " /api/v1/auth/logout":         false,
	}
	for _, route := range e.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("route not registered: %s", route)
		}
	}
}
