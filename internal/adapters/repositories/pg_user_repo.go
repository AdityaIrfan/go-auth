package repositories

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"kda-auth-service/internal/core/domain"
	"kda-auth-service/internal/core/ports"
)

type pgUserRepo struct {
	db *gorm.DB
}

func NewPGUserRepository(db *gorm.DB) ports.UserRepository {
	return &pgUserRepo{db: db}
}

func (r *pgUserRepo) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *pgUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ports.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
