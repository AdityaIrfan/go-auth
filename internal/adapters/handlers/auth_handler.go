package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"kda-auth-service/internal/core/domain"
	"kda-auth-service/internal/core/ports"
	"kda-auth-service/pkg/config"
	"kda-auth-service/pkg/response"
	"kda-auth-service/pkg/utils"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
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
		return response.ErrorResponse(c, http.StatusBadRequest, "invalid request body", nil)
	}
	if err := c.Validate(req); err != nil {
		errs := utils.FormatValidationError(err)
		return response.ErrorResponse(c, http.StatusBadRequest, "invalid request body", errs)
	}

	err := h.service.Register(c.Request().Context(), req)
	if err != nil {
		log.Println("REGISTER FAILED: ", err.Error())
		return response.ErrorResponse(c, http.StatusUnauthorized, "register failed", nil)
	}

	return response.SuccessResponse(c, http.StatusCreated, "registered successfully", nil)
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
		return response.ErrorResponse(c, http.StatusBadRequest, "invalid request body", nil)
	}
	if err := c.Validate(req); err != nil {
		errs := utils.FormatValidationError(err)
		return response.ErrorResponse(c, http.StatusBadRequest, "invalid request body", errs)
	}

	resp, err := h.service.Login(c.Request().Context(), req)
	if err != nil {
		log.Println("LOGIN FAILED: ", err.Error())
		return response.ErrorResponse(c, http.StatusUnauthorized, "login failed", nil)
	}

	return response.SuccessResponse(c, http.StatusOK, "logged in successfully", resp)
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
		fmt.Println("ERROR OAUTH CONFIG EXCHANGE: ", err.Error())
		return response.ErrorResponse(c, http.StatusUnauthorized, "login failed", nil)
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Println("ERROR TOKEN EXTRA: missing id_token in google response")
		return response.ErrorResponse(c, http.StatusUnauthorized, "login failed", nil)
	}

	resp, err := h.service.GoogleSSO(c.Request().Context(), domain.GoogleSSOReq{
		IDToken: idToken,
	})
	if err != nil {
		fmt.Println("ERROR LOGIN OAUTH2: ", err.Error())
		return response.ErrorResponse(c, http.StatusUnauthorized, "login failed", nil)
	}

	return response.SuccessResponse(c, http.StatusOK, "logged in successfully", resp)
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

	err := h.service.Logout(c.Request().Context(), token)
	if err != nil {
		fmt.Println("ERROR LOGOUT: ", err.Error())
		return response.ErrorResponse(c, http.StatusUnauthorized, "logout failed", nil)
	}

	return response.SuccessResponse(c, http.StatusOK, "logged out successfully", nil)
}
