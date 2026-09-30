package repositories

import (
	"context"
	"kda-auth-service/internal/core/domain"
	"kda-auth-service/internal/core/ports"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type pgEventRepository struct {
	db *gorm.DB
}

func NewPGEventRepository(db *gorm.DB) ports.EventRepository {
	return &pgEventRepository{db: db}
}

func (r *pgEventRepository) Create(ctx context.Context, event *domain.Event) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *pgEventRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Event, error) {
	var events []domain.Event
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("start_time ASC").Find(&events).Error
	return events, err
}

func (r *pgEventRepository) GetByID(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) (*domain.Event, error) {
	var event domain.Event
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", eventID, userID).First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *pgEventRepository) GetByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]domain.Event, error) {
	var events []domain.Event

	db := r.db.WithContext(ctx).Where("user_id = ?", userID)

	if !startDate.IsZero() {
		db = db.Where("start_time >= ?", startDate)
	}

	if !endDate.IsZero() {
		db = db.Where("end_time <= ?", endDate)
	}

	err := db.Order("start_time ASC").Find(&events).Error

	return events, err
}

func (r *pgEventRepository) Update(ctx context.Context, event *domain.Event) error {
	return r.db.WithContext(ctx).Save(event).Error
}

func (r *pgEventRepository) Delete(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ? AND user_id = ?", eventID, userID).Delete(&domain.Event{}).Error
}
