package handlers

import (
	customjwt "kda-auth-service/pkg/jwt"
	"kda-auth-service/pkg/response"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(secret),
		ContextKey: "token",
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(customjwt.CustomClaims)
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return response.EchoResponse(c, response.ErrorResponse(http.StatusUnauthorized, "invalid or expired token", nil))
		},
	})
}
