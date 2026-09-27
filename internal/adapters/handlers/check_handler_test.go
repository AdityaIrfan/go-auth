package handlers

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestNewCheckHandler(t *testing.T) {
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("gorm.Open(): %v", err)
	}
	mock.ExpectPing()

	client := redis.NewClient(&redis.Options{
		Addr:       "unused:6379",
		MaxRetries: -1,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("redis unavailable")
		},
	})
	defer client.Close()
	handler := NewCheckHandler(db, client)
	if handler == nil || handler.postgresPing == nil || handler.redisPing == nil {
		t.Fatal("constructor did not configure dependency checks")
	}
	e := echo.New()
	rec := httptest.NewRecorder()
	ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/ready", nil), rec)
	if err := handler.Ready(ctx); err != nil || rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), `"redis":"down"`) {
		t.Fatalf("Ready() code=%d body=%s err=%v", rec.Code, rec.Body.String(), err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestCheckHandlerHealth(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/health", nil), rec)
	if err := (&CheckHandler{}).Health(ctx); err != nil || rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "I'm healthy") {
		t.Fatalf("Health() code=%d body=%s err=%v", rec.Code, rec.Body.String(), err)
	}
}

func TestCheckHandlerReady(t *testing.T) {
	tests := []struct {
		name        string
		postgresErr error
		redisErr    error
		wantCode    int
		wantBody    string
	}{
		{name: "all dependencies up", wantCode: http.StatusOK, wantBody: "I'm ready"},
		{name: "postgres down", postgresErr: errors.New("down"), wantCode: http.StatusServiceUnavailable, wantBody: `"postgres":"down"`},
		{name: "redis down", redisErr: errors.New("down"), wantCode: http.StatusServiceUnavailable, wantBody: `"redis":"down"`},
		{name: "all dependencies down", postgresErr: errors.New("down"), redisErr: errors.New("down"), wantCode: http.StatusServiceUnavailable, wantBody: `"postgres":"down"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &CheckHandler{
				postgresPing: func(context.Context) error { return tc.postgresErr },
				redisPing:    func(context.Context) error { return tc.redisErr },
			}
			e := echo.New()
			rec := httptest.NewRecorder()
			ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/ready", nil), rec)
			if err := h.Ready(ctx); err != nil || rec.Code != tc.wantCode || !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("Ready() code=%d body=%s err=%v", rec.Code, rec.Body.String(), err)
			}
		})
	}
}
