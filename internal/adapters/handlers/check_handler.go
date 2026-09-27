package handlers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type CheckHandler struct {
	postgresPing func(context.Context) error
	redisPing    func(context.Context) error
}

func NewCheckHandler(db *gorm.DB, rdb *redis.Client) *CheckHandler {
	return &CheckHandler{
		postgresPing: func(ctx context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.PingContext(ctx)
		},
		redisPing: func(ctx context.Context) error {
			return rdb.Ping(ctx).Err()
		},
	}
}

// Health godoc
// @Summary Liveness check
// @Tags System
// @Produce json
// @Success 200 {object} HealthSuccessResponse
// @Router /health [get]
func (h *CheckHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "I'm healthy"})
}

// Ready godoc
// @Summary Readiness check PostgreSQL dan Redis
// @Tags System
// @Produce json
// @Success 200 {object} ReadySuccessResponse
// @Failure 503 {object} ReadinessErrorResponse
// @Router /ready [get]
func (h *CheckHandler) Ready(c echo.Context) error {
	status := map[string]string{
		"postgres": "up",
		"redis":    "up",
	}
	isReady := true
	ctx := c.Request().Context()

	if err := h.postgresPing(ctx); err != nil {
		status["postgres"] = "down"
		isReady = false
	}

	if err := h.redisPing(ctx); err != nil {
		status["redis"] = "down"
		isReady = false
	}

	if !isReady {
		return c.JSON(http.StatusServiceUnavailable, status)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "I'm ready"})
}
