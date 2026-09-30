package handlers

import (
	"kda-auth-service/internal/core/domain"
	"kda-auth-service/internal/core/ports"
	jwtPkg "kda-auth-service/pkg/jwt"
	"kda-auth-service/pkg/response"
	"kda-auth-service/pkg/utils"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type EventHandler struct {
	service ports.EventService
}

func NewEventHandler(s ports.EventService) *EventHandler {
	return &EventHandler{service: s}
}

func (h *EventHandler) Create(c echo.Context) error {
	var req domain.CreateEventReq
	if err := c.Bind(&req); err != nil {
		return response.EchoResponseInvalidRequestBody(c, nil)
	}

	if err := c.Validate(&req); err != nil {
		errs := utils.FormatValidationError(err)
		return response.EchoResponseInvalidRequestBody(c, errs)
	}

	userID := h.extractUserID(c)
	res := h.service.CreateEvent(c.Request().Context(), userID, req)
	return response.EchoResponse(c, res)
}

func (h *EventHandler) List(c echo.Context) error {
	var req domain.ListEventReq
	if err := c.Bind(&req); err != nil {
		return response.EchoResponseInvalidRequestBody(c, nil)
	}

	userID := h.extractUserID(c)
	res := h.service.ListEvents(c.Request().Context(), userID, req)
	return response.EchoResponse(c, res)
}

func (h *EventHandler) Update(c echo.Context) error {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return response.EchoResponse(c, response.ErrorResponse(http.StatusBadRequest, "invalid event id", nil))
	}

	var req domain.UpdateEventReq
	if err := c.Bind(&req); err != nil {
		return response.EchoResponseInvalidRequestBody(c, nil)
	}

	if err := c.Validate(&req); err != nil {
		errs := utils.FormatValidationError(err)
		return response.EchoResponseInvalidRequestBody(c, errs)
	}

	userID := h.extractUserID(c)
	res := h.service.UpdateEvent(c.Request().Context(), eventID, userID, req)
	return response.EchoResponse(c, res)
}

func (h *EventHandler) Delete(c echo.Context) error {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return response.EchoResponse(c, response.ErrorResponse(http.StatusBadRequest, "invalid event id", nil))
	}

	userID := h.extractUserID(c)
	res := h.service.DeleteEvent(c.Request().Context(), eventID, userID)
	return response.EchoResponse(c, res)
}

func (h *EventHandler) extractUserID(c echo.Context) uuid.UUID {
	userToken := c.Get("token").(*jwt.Token)
	claims := userToken.Claims.(*jwtPkg.CustomClaims)
	return claims.UserID
}
