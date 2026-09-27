package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"kda-auth-service/internal/core/domain"
	"kda-auth-service/internal/core/ports"
	"kda-auth-service/pkg/jwt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	userRepo           ports.UserRepository
	tokenRepo          ports.TokenCacheRepository
	httpClient         httpDoer
	googleTokenInfoURL string
	generateToken      func(uuid.UUID, int) (string, error)
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

func (s *authService) Register(ctx context.Context, req domain.RegisterReq) error {
	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return errors.New("email already registered")
	}
	if err != nil && !errors.Is(err, ports.ErrUserNotFound) {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &domain.User{
		ID:       uuid.New(),
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
		Provider: "email",
	}

	return s.userRepo.Create(ctx, user)
}

func (s *authService) Login(ctx context.Context, req domain.LoginReq) (*domain.TokenResp, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil || user == nil || user.Provider != "email" {
		return nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	return s.generateAndStoreToken(ctx, user.ID)
}

func (s *authService) GoogleSSO(ctx context.Context, req domain.GoogleSSOReq) (*domain.TokenResp, error) {
	endpoint, err := url.Parse(s.googleTokenInfoURL)
	if err != nil {
		return nil, errors.New("invalid google tokeninfo configuration")
	}
	query := endpoint.Query()
	query.Set("id_token", req.IDToken)
	endpoint.RawQuery = query.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, errors.New("invalid google id_token")
	}
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.New("invalid google id_token")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid google id_token")
	}

	var googleClaims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Sub   string `json:"sub"`
		Aud   string `json:"aud"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleClaims); err != nil || googleClaims.Email == "" || googleClaims.Sub == "" {
		return nil, errors.New("invalid google id_token")
	}
	if clientID := os.Getenv("GOOGLE_CLIENT_ID"); clientID != "" && googleClaims.Aud != clientID {
		return nil, errors.New("invalid google id_token")
	}

	user, err := s.userRepo.FindByEmail(ctx, googleClaims.Email)
	if err != nil && !errors.Is(err, ports.ErrUserNotFound) {
		return nil, err
	}
	if user == nil {
		// Auto register if not found
		user = &domain.User{
			ID:       uuid.New(),
			Email:    googleClaims.Email,
			Name:     googleClaims.Name,
			Provider: "google",
			GoogleID: googleClaims.Sub,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, err
		}
	}

	return s.generateAndStoreToken(ctx, user.ID)
}

func (s *authService) Logout(ctx context.Context, token string) error {
	return s.tokenRepo.Revoke(ctx, token)
}

func (s *authService) generateAndStoreToken(ctx context.Context, userID uuid.UUID) (*domain.TokenResp, error) {
	expHoursStr := os.Getenv("JWT_EXPIRATION_HOURS")
	expHours, _ := strconv.Atoi(expHoursStr)
	if expHours == 0 {
		expHours = 24
	}

	ttlSec := expHours * 3600
	tokenStr, err := s.generateToken(userID, expHours)
	if err != nil {
		return nil, err
	}

	err = s.tokenRepo.Store(ctx, userID, tokenStr, ttlSec)
	if err != nil {
		return nil, err
	}

	return &domain.TokenResp{
		AccessToken: tokenStr,
		TokenType:   "Bearer",
		ExpiresIn:   ttlSec,
	}, nil
}
