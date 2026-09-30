package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"kda-auth-service/internal/core/domain"
	"kda-auth-service/internal/core/ports"
	"kda-auth-service/pkg/jwt"
	"kda-auth-service/pkg/response"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	userRepo           ports.UserRepository
	tokenRepo          ports.TokenCacheRepository
	httpClient         httpDoer
	googleTokenInfoURL string
	generateToken      func(uuid.UUID, int) (jwt.Token, error)
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewAuthService(ur ports.UserRepository, tr ports.TokenCacheRepository) ports.AuthService {
	return &authService{
		userRepo:           ur,
		tokenRepo:          tr,
		httpClient:         http.DefaultClient,
		googleTokenInfoURL: "https://oauth2.googleapis.com/tokeninfo",
		generateToken:      jwt.GenerateToken,
	}
}

func (s *authService) Register(ctx context.Context, req domain.RegisterReq) response.Response {
	registerFailed := response.ErrorResponse(http.StatusUnprocessableEntity, "register failed", nil)

	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return response.ErrorResponse(http.StatusBadRequest, "email already registered", nil)
	}
	if err != nil && !errors.Is(err, ports.ErrUserNotFound) {
		log.Error().Msg("ERROR FindByEmail: " + err.Error())
		return registerFailed
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Msg("ERROR bcrypt.GenerateFromPassword: " + err.Error())
		return registerFailed
	}

	user := &domain.User{
		ID:       uuid.New(),
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
		Provider: "email",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		log.Error().Msg("ERROR userRepo.Create: " + err.Error())
		return registerFailed
	}

	return response.SuccessResponse(http.StatusCreated, "register success, do login", nil)
}

func (s *authService) Login(ctx context.Context, req domain.LoginReq) response.Response {
	invalidCredential := response.ErrorResponse(http.StatusUnauthorized, "invalid credentials", nil)

	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil || user == nil || user.Provider != "email" {
		log.Info().Msg(fmt.Sprintf("email %s isn't email provider", req.Email))
		return invalidCredential
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		log.Info().Msg(fmt.Sprintf("email %s invalid password", req.Email))
		return invalidCredential
	}

	token, err := s.generateAndStoreToken(ctx, user.ID)
	if err != nil {
		log.Error().Msg("ERROR generateAndStoreToken: " + err.Error())
		return response.ErrorResponse(http.StatusUnprocessableEntity, "login failed", nil)
	}

	return response.SuccessResponse(http.StatusOK, "login success", token)
}

func (s *authService) GoogleSSO(ctx context.Context, req domain.GoogleSSOReq) response.Response {
	loginFailed := response.ErrorResponse(http.StatusUnprocessableEntity, "login failed", nil)

	endpoint, err := url.Parse(s.googleTokenInfoURL)
	if err != nil {
		log.Error().Msg("ERROR url.Parse: invalid google tokeninfo configuration")
		return loginFailed
	}
	query := endpoint.Query()
	query.Set("id_token", req.IDToken)
	endpoint.RawQuery = query.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		log.Error().Msg("invalid google id_token")
		return loginFailed
	}
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		log.Error().Msg("ERROR httpClient.Do: " + err.Error())
		return loginFailed
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Error().Msg(fmt.Sprintf("ERROR httpClient.Do: %d", resp.StatusCode))
		return loginFailed
	}

	var googleClaims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Sub   string `json:"sub"`
		Aud   string `json:"aud"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleClaims); err != nil || googleClaims.Email == "" || googleClaims.Sub == "" {
		log.Error().Msg("ERROR json.NewDecoder: " + err.Error())
		return loginFailed
	}
	if clientID := os.Getenv("GOOGLE_CLIENT_ID"); clientID != "" && googleClaims.Aud != clientID {
		log.Error().Msg("ERROR GOOGLE_CLIENT_ID doesn't match with googleClaims.Aud")
		return loginFailed
	}

	user, err := s.userRepo.FindByEmail(ctx, googleClaims.Email)
	if err != nil && !errors.Is(err, ports.ErrUserNotFound) {
		log.Error().Msg("ERROR userRepo.FindByEmail: " + err.Error())
		return loginFailed
	}
	if user == nil {
		user = &domain.User{
			ID:       uuid.New(),
			Email:    googleClaims.Email,
			Name:     googleClaims.Name,
			Provider: "google",
			GoogleID: req.IDToken,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			log.Error().Msg("ERROR userRepo.Create: " + err.Error())
			return loginFailed
		}
	}

	token, err := s.generateAndStoreToken(ctx, user.ID)
	if err != nil {
		log.Error().Msg("ERROR generateAndStoreToken: " + err.Error())
		return loginFailed
	}

	return response.SuccessResponse(http.StatusOK, "login success", token)
}

func (s *authService) Logout(ctx context.Context, token string) response.Response {
	if err := s.tokenRepo.Revoke(ctx, token); err != nil {
		log.Error().Msg("ERROR tokenRepo.Revoke: " + err.Error())
		return response.ErrorResponse(http.StatusUnprocessableEntity, "logout failed", nil)
	}
	return response.SuccessResponse(http.StatusOK, "logout success", nil)
}

func (s *authService) generateAndStoreToken(ctx context.Context, userID uuid.UUID) (*domain.TokenResp, error) {
	expHoursStr := os.Getenv("JWT_EXPIRATION_HOURS")
	expHours, _ := strconv.Atoi(expHoursStr)
	if expHours == 0 {
		expHours = 24
	}

	ttlSec := expHours * 3600
	token, err := s.generateToken(userID, expHours)
	if err != nil {
		return nil, err
	}

	err = s.tokenRepo.Store(ctx, userID, token.AccessToken, ttlSec)
	if err != nil {
		return nil, err
	}

	return &domain.TokenResp{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    ttlSec,
	}, nil
}

func (s *authService) RefreshToken(ctx context.Context, userID uuid.UUID) response.Response {
	refreshTokenFailed := response.ErrorResponse(http.StatusUnprocessableEntity, "refresh token failed", nil)

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil && !errors.Is(err, ports.ErrUserNotFound) {
		log.Error().Msg("ERROR userRepo.FindByEmail: " + err.Error())
		return refreshTokenFailed
	}

	token, err := s.generateAndStoreToken(ctx, user.ID)
	if err != nil {
		log.Error().Msg("ERROR generateAndStoreToken: " + err.Error())
		return refreshTokenFailed
	}

	return response.SuccessResponse(http.StatusOK, "refresh token success", token)
}
