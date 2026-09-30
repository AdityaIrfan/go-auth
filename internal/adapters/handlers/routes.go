package handlers

import (
	"net/http"
	"os"

	_ "kda-auth-service/docs"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func SetupRoutes(e *echo.Echo, checkHandler *CheckHandler, authHandler *AuthHandler, eventHandler *EventHandler) {
	redirectToSwagger := func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/swagger/index.html")
	}
	e.GET("/swagger", redirectToSwagger)
	e.GET("/swagger/", redirectToSwagger)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	e.GET("/health", checkHandler.Health)
	e.GET("/ready", checkHandler.Ready)

	v1 := e.Group("/api/v1")

	// Public Routes
	v1.POST("/auth/register", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)
	v1.GET("/auth/google/login", authHandler.GoogleSSOLogin)
	v1.GET("/auth/google/callback", authHandler.GoogleSSOCallback)
	v1.POST("/auth/refresh", authHandler.RefreshToken)

	// Protected Routes (Butuh Token)
	jwtSecret := os.Getenv("JWT_SECRET")
	v1Protected := v1.Group("")
	v1Protected.Use(JWTMiddleware(jwtSecret))

	v1Protected.POST("/auth/logout", authHandler.Logout)

	v1Protected.POST("/events", eventHandler.Create)
	v1Protected.GET("/events", eventHandler.List)
	v1Protected.PUT("/events/:id", eventHandler.Update)
	v1Protected.DELETE("/events/:id", eventHandler.Delete)
}
