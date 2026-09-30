package ports

import (
	"context"
	"errors"
	"kda-auth-service/internal/core/domain"
	"kda-auth-service/pkg/response"
	"time"

	"github.com/google/uuid"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

type TokenCacheRepository interface {
	Store(ctx context.Context, userID uuid.UUID, token string, ttlSeconds int) error
	Revoke(ctx context.Context, token string) error
	Validate(ctx context.Context, token string) (bool, error)
}

type AuthService interface {
	Register(ctx context.Context, req domain.RegisterReq) response.Response
	Login(ctx context.Context, req domain.LoginReq) response.Response
	GoogleSSO(ctx context.Context, req domain.GoogleSSOReq) response.Response
	Logout(ctx context.Context, token string) response.Response
	RefreshToken(ctx context.Context, userID uuid.UUID) response.Response
}

type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Event, error)
	GetByID(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) (*domain.Event, error)
	GetByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]domain.Event, error)
	Update(ctx context.Context, event *domain.Event) error
	Delete(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) error
}

type EventService interface {
	CreateEvent(ctx context.Context, userID uuid.UUID, req domain.CreateEventReq) response.Response
	ListEvents(ctx context.Context, userID uuid.UUID, req domain.ListEventReq) response.Response
	UpdateEvent(ctx context.Context, eventID uuid.UUID, userID uuid.UUID, req domain.UpdateEventReq) response.Response
	DeleteEvent(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) response.Response
}
