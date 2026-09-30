package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"kda-auth-service/internal/core/domain"
	"kda-auth-service/internal/core/ports"
	"kda-auth-service/pkg/config"
	jwtPkg "kda-auth-service/pkg/jwt"
	"kda-auth-service/pkg/response"
	"kda-auth-service/pkg/utils"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

type AuthHandler struct {
	service ports.AuthService
}

func NewAuthHandler(s ports.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

// Register godoc
// @Summary Register dengan email dan password
// @Description Membuat user baru menggunakan email dan password.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body domain.RegisterReq true "Registration payload"
// @Success 201 {object} RegisterSuccessResponse
// @Failure 400 {object} ValidationErrorResponse
// @Failure 422 {object} DuplicateEmailErrorResponse
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req domain.RegisterReq
	if err := c.Bind(&req); err != nil {
		return response.EchoResponseInvalidRequestBody(c, nil)
	}
	if err := c.Validate(req); err != nil {
		errs := utils.FormatValidationError(err)
		return response.EchoResponseInvalidRequestBody(c, errs)
	}

	res := h.service.Register(c.Request().Context(), req)
	return response.EchoResponse(c, res)
}

// Login godoc
// @Summary Login dengan email dan password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body domain.LoginReq true "Login payload"
// @Success 200 {object} TokenSuccessResponse
// @Failure 400 {object} ValidationErrorResponse
// @Failure 401 {object} InvalidCredentialsErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req domain.LoginReq
	if err := c.Bind(&req); err != nil {
		return response.EchoResponseInvalidRequestBody(c, nil)
	}
	if err := c.Validate(req); err != nil {
		errs := utils.FormatValidationError(err)
		return response.EchoResponseInvalidRequestBody(c, errs)
	}

	res := h.service.Login(c.Request().Context(), req)
	return response.EchoResponse(c, res)
}

func (h *AuthHandler) GoogleSSOLogin(c echo.Context) error {
	oauthState := generateStateOauthCookie()
	fmt.Println("clientId: ", config.GoogleOauthConfig.ClientID)
	fmt.Println("clientSecret: ", config.GoogleOauthConfig.ClientSecret)
	fmt.Println("redirect: ", config.GoogleOauthConfig.RedirectURL)
	url := config.GoogleOauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func generateStateOauthCookie() string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	return state
}

func (h *AuthHandler) GoogleSSOCallback(c echo.Context) error {
	token, err := config.GoogleOauthConfig.Exchange(context.Background(), c.FormValue("code"))
	if err != nil {
		log.Error().Msg("ERROR OAUTH CONFIG EXCHANGE: " + err.Error())
		return response.EchoResponse(c, response.ErrorResponse(http.StatusUnauthorized, "login failed", nil))
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Error().Msg("ERROR TOKEN EXTRA: missing id_token in google response")
		return response.EchoResponse(c, response.ErrorResponse(http.StatusUnauthorized, "login failed", nil))
	}

	res := h.service.GoogleSSO(c.Request().Context(), domain.GoogleSSOReq{
		IDToken: idToken,
	})
	return response.EchoResponse(c, res)
}

// Logout godoc
// @Summary Logout dan revoke access token
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} LogoutSuccessResponse
// @Failure 401 {object} JWTErrorResponse
// @Failure 422 {object} LogoutErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	authHeader := c.Request().Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")

	res := h.service.Logout(c.Request().Context(), token)
	return response.EchoResponse(c, res)
}

func (h *AuthHandler) RefreshToken(c echo.Context) error {
	var req domain.RefreshTokenReq
	if err := c.Bind(&req); err != nil {
		return response.EchoResponseInvalidRequestBody(c, nil)
	}
	if err := c.Validate(req); err != nil {
		errs := utils.FormatValidationError(err)
		return response.EchoResponseInvalidRequestBody(c, errs)
	}

	userID, err := jwtPkg.UnpackRefreshToken(req.RefreshToken)
	if err != nil {
		log.Error().Msg("ERROR: failed to unpack refresh token")
		return response.EchoResponse(c, response.ErrorResponse(http.StatusUnprocessableEntity, "refresh token failed", nil))
	}

	res := h.service.RefreshToken(c.Request().Context(), userID)
	return response.EchoResponse(c, res)
}
