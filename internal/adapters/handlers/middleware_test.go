package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	customjwt "kda-auth-service/pkg/jwt"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestJWTMiddleware(t *testing.T) {
	const secret = "test-secret"
	validClaims := customjwt.CustomClaims{UserID: uuid.New(), RegisteredClaims: jwtlib.RegisteredClaims{ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour))}}
	validToken, _ := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, validClaims).SignedString([]byte(secret))
	expiredClaims := customjwt.CustomClaims{UserID: uuid.New(), RegisteredClaims: jwtlib.RegisteredClaims{ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(-time.Hour))}}
	expiredToken, _ := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, expiredClaims).SignedString([]byte(secret))

	for _, tc := range []struct {
		name  string
		token string
		want  int
	}{
		{name: "valid", token: validToken, want: http.StatusNoContent},
		{name: "missing", want: http.StatusUnauthorized},
		{name: "invalid signature", token: validToken + "x", want: http.StatusUnauthorized},
		{name: "expired", token: expiredToken, want: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			e.GET("/protected", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) }, JWTMiddleware(secret))
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.token != "" {
				req.Header.Set(echo.HeaderAuthorization, "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
			}
			if tc.want == http.StatusUnauthorized && !strings.Contains(rec.Body.String(), "invalid or expired token") {
				t.Fatalf("body=%s", rec.Body.String())
			}
		})
	}
}
