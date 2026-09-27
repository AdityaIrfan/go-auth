package handlers

import (
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	customjwt "kda-auth-service/pkg/jwt"
	"net/http"
)

func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(secret),
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(customjwt.CustomClaims)
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
		},
	})
}
