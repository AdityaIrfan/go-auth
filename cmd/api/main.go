package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"kda-auth-service/internal/adapters/handlers"
	"kda-auth-service/internal/adapters/repositories"
	"kda-auth-service/internal/core/services"
	"kda-auth-service/pkg/config"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// @title KDA Auth Service API
// @version 1.0
// @description Authentication API for email/password and Google ID tokens.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Masukkan token dengan format: Bearer {token}
func main() {
	godotenv.Load()
	pgConn := config.NewPostgresConn()
	redisConn := config.NewRedisConn()
	config.GoogleConfig()

	userRepo := repositories.NewPGUserRepository(pgConn)
	tokenRepo := repositories.NewRedisTokenRepository(redisConn)

	authService := services.NewAuthService(userRepo, tokenRepo)

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	checkHandler := handlers.NewCheckHandler(pgConn, redisConn)
	authHandler := handlers.NewAuthHandler(authService)

	handlers.SetupRoutes(e, checkHandler, authHandler)

	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
