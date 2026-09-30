package services

import (
	"context"
	"kda-auth-service/internal/core/domain"
	"kda-auth-service/internal/core/ports"
	"kda-auth-service/pkg/response"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type eventService struct {
	repo ports.EventRepository
}

func NewEventService(r ports.EventRepository) ports.EventService {
	return &eventService{repo: r}
}

func (s *eventService) CreateEvent(ctx context.Context, userID uuid.UUID, req domain.CreateEventReq) response.Response {
	if req.EndTime.Before(req.StartTime) || req.EndTime.Equal(req.StartTime) {
		return response.ErrorResponse(http.StatusBadRequest, "end_time must be after start_time", nil)
	}

	event := &domain.Event{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, event); err != nil {
		log.Error().Msg("ERROR repo.Create: " + err.Error())
		return response.ErrorResponse(http.StatusUnprocessableEntity, "create event failed", nil)
	}

	return response.SuccessResponse(http.StatusCreated, "create event success", event)
}

func (s *eventService) ListEvents(ctx context.Context, userID uuid.UUID, req domain.ListEventReq) response.Response {
	events, err := s.repo.GetByDateRange(ctx, userID, req.StartTime, req.EndTime)
	if err != nil {
		log.Error().Msg("ERROR repo.GetByUserID : " + err.Error())
		return response.ErrorResponse(http.StatusUnprocessableEntity, "list events failed", nil)
	}

	return response.SuccessResponse(http.StatusOK, "list events success", events)
}

func (s *eventService) UpdateEvent(ctx context.Context, eventID uuid.UUID, userID uuid.UUID, req domain.UpdateEventReq) response.Response {
	event, err := s.repo.GetByID(ctx, eventID, userID)
	if err != nil {
		log.Error().Msg("ERROR repo.GetByID: " + err.Error())
		return response.ErrorResponse(http.StatusNotFound, "event not found", nil)
	}

	if req.EndTime.Before(req.StartTime) {
		return response.ErrorResponse(http.StatusBadRequest, "end_time must be after start_time", nil)
	}

	event.Title = req.Title
	event.Description = req.Description
	event.StartTime = req.StartTime
	event.EndTime = req.EndTime
	event.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, event); err != nil {
		log.Error().Msg("ERROR repo.Update: " + err.Error())
		return response.ErrorResponse(http.StatusUnprocessableEntity, "update event failed", nil)
	}

	return response.SuccessResponse(http.StatusOK, "update event success", event)
}

func (s *eventService) DeleteEvent(ctx context.Context, eventID uuid.UUID, userID uuid.UUID) response.Response {
	_, err := s.repo.GetByID(ctx, eventID, userID)
	if err != nil {
		log.Error().Msg("ERROR repo.GetByID: " + err.Error())
		return response.ErrorResponse(http.StatusNotFound, "event not found", nil)
	}
	if err := s.repo.Delete(ctx, eventID, userID); err != nil {
		log.Error().Msg("ERROR repo.Delete: " + err.Error())
		return response.ErrorResponse(http.StatusUnprocessableEntity, "delete event failed", nil)
	}

	return response.SuccessResponse(http.StatusOK, "delete event success", nil)
}
