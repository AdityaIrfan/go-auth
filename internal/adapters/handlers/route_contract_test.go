package handlers

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"kda-auth-service/docs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSwaggerMatchesRegisteredRoutes(t *testing.T) {
	e := echo.New()
	SetupRoutes(e, &CheckHandler{}, NewAuthHandler(&fakeAuthService{}), NewEventHandler(&fakeEventService{}))
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal([]byte(docs.SwaggerInfo.ReadDoc()), &spec); err != nil {
		t.Fatal(err)
	}
	actual := map[string]bool{}
	for _, r := range e.Routes() {
		if strings.HasPrefix(r.Path, "/swagger") || r.Method == "echo_route_not_found" {
			continue
		}
		path := strings.ReplaceAll(r.Path, ":id", "{id}")
		method := strings.ToLower(r.Method)
		actual[method+" "+path] = true
		if _, ok := spec.Paths[path][method]; !ok {
			t.Errorf("undocumented runtime route %s %s", method, path)
		}
	}
	for path, methods := range spec.Paths {
		for method := range methods {
			if !actual[method+" "+path] {
				t.Errorf("documented route missing from runtime: %s %s", method, path)
			}
		}
	}
}

func TestProtectedRoutesRejectMissingToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	e := echo.New()
	SetupRoutes(e, &CheckHandler{}, NewAuthHandler(&fakeAuthService{}), NewEventHandler(&fakeEventService{}))
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/auth/logout"}, {http.MethodGet, "/api/v1/events"}, {http.MethodPost, "/api/v1/events"}, {http.MethodPut, "/api/v1/events/bad"}, {http.MethodDelete, "/api/v1/events/bad"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
			body := responseBody(t, rec)
			if rec.Code != 401 || body["status_code"] != float64(401) || body["status"] != "failed" || body["message"] != "invalid or expired token" {
				t.Fatalf("body=%s", rec.Body.String())
			}
		})
	}
}
