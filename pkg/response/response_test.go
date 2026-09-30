package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestSuccessResponse(t *testing.T) {
	e := echo.New()
	for _, tc := range []struct {
		name    string
		data    interface{}
		hasData bool
	}{
		{name: "with data", data: map[string]string{"id": "1"}, hasData: true},
		{name: "without data", hasData: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
			if err := EchoResponse(ctx, SuccessResponse(http.StatusCreated, "created", tc.data)); err != nil {
				t.Fatalf("error=%v", err)
			}
			var body map[string]interface{}
			_ = json.Unmarshal(rec.Body.Bytes(), &body)
			_, hasData := body["data"]
			if rec.Code != http.StatusCreated || body["status"] != "success" || hasData != tc.hasData {
				t.Fatalf("body=%#v", body)
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	e := echo.New()
	for _, tc := range []struct {
		name      string
		errors    map[string]string
		hasErrors bool
	}{
		{name: "with errors", errors: map[string]string{"email": "invalid"}, hasErrors: false},
		{name: "without errors", hasErrors: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
			if err := EchoResponse(ctx, ErrorResponse(http.StatusBadRequest, "bad request", tc.errors)); err != nil {
				t.Fatalf("error=%v", err)
			}
			var body map[string]interface{}
			_ = json.Unmarshal(rec.Body.Bytes(), &body)
			_, hasErrors := body["errors"]
			if rec.Code != http.StatusBadRequest || body["status"] != "failed" || hasErrors != tc.hasErrors {
				t.Fatalf("body=%#v", body)
			}
		})
	}
}
