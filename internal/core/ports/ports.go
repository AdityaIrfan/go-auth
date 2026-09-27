package ports

import (
	"context"
	"errors"
	"kda-auth-service/internal/core/domain"

	"github.com/google/uuid"
)

var ErrUserNotFound = errors.New("user not found")

// Driven Ports (Outbound) - Diimplementasi oleh Adapters (Repositories)
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}

type TokenCacheRepository interface {
	Store(ctx context.Context, userID uuid.UUID, token string, ttlSeconds int) error
	Revoke(ctx context.Context, token string) error
	Validate(ctx context.Context, token string) (bool, error)
}

// Driving Ports (Inbound) - Diimplementasi oleh Services, dipanggil oleh Handlers
type AuthService interface {
	Register(ctx context.Context, req domain.RegisterReq) error
	Login(ctx context.Context, req domain.LoginReq) (*domain.TokenResp, error)
	GoogleSSO(ctx context.Context, req domain.GoogleSSOReq) (*domain.TokenResp, error)
	Logout(ctx context.Context, token string) error
}
