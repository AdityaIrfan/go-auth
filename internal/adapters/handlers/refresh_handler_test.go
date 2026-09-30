package handlers

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	jwtPkg "kda-auth-service/pkg/jwt"
	"kda-auth-service/pkg/response"
	"net/http"
	"testing"
)

func TestRefreshHandler(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	id := uuid.New()
	pair, err := jwtPkg.GenerateToken(id, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, body string
		code       int
		called     bool
	}{
		{"malformed", "{", 400, false}, {"required", "{}", 400, false}, {"invalid token", `{"refresh_token":"bad"}`, 422, false},
		{"valid refresh", fmt.Sprintf(`{"refresh_token":%q}`, pair.RefreshToken), 200, true},
		{"service failure", fmt.Sprintf(`{"refresh_token":%q}`, pair.RefreshToken), 422, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			h := NewAuthHandler(&fakeAuthService{refreshFn: func(_ context.Context, u uuid.UUID) response.Response {
				called = true
				if u != id {
					t.Fatal("wrong user")
				}
				if tc.code == 422 {
					return response.ErrorResponse(422, "refresh token failed", nil)
				}
				return response.SuccessResponse(200, "refresh token success", map[string]string{"access_token": "new", "refresh_token": "new-refresh"})
			}})
			c, rec := newHandlerContext(http.MethodPost, "/api/v1/auth/refresh", tc.body)
			if err := h.RefreshToken(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tc.code || called != tc.called {
				t.Fatalf("code=%d called=%v body=%s", rec.Code, called, rec.Body.String())
			}
			body := responseBody(t, rec)
			message := "refresh token failed"
			if tc.code == 400 {
				message = "invalid request body"
			}
			if tc.code == 200 {
				message = "refresh token success"
				if body["data"].(map[string]interface{})["refresh_token"] != "new-refresh" {
					t.Fatal("missing new pair")
				}
			}
			if body["message"] != message {
				t.Fatalf("body=%#v", body)
			}
		})
	}
}
