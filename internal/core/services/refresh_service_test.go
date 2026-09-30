package services

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"kda-auth-service/internal/core/domain"
	"kda-auth-service/internal/core/ports"
	jwtPkg "kda-auth-service/pkg/jwt"
	"testing"
)

func TestRefreshToken(t *testing.T) {
	id := uuid.New()
	for _, tc := range []struct {
		name                          string
		lookupErr, tokenErr, storeErr error
		code                          int
	}{
		{"success", nil, nil, nil, 200}, {"lookup failure", errors.New("db"), nil, nil, 422}, {"token failure", nil, errors.New("sign"), nil, 422}, {"store failure", nil, nil, errors.New("redis"), 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("JWT_EXPIRATION_HOURS", "2")
			stored := false
			svc := newServiceForTest(&fakeUserRepository{findByIDFn: func(_ context.Context, u uuid.UUID) (*domain.User, error) {
				if u != id {
					t.Fatal("wrong user")
				}
				return &domain.User{ID: id}, tc.lookupErr
			}}, &fakeTokenRepository{storeFn: func(_ context.Context, u uuid.UUID, token string, ttl int) error {
				stored = true
				if u != id || token != "access" || ttl != 7200 {
					t.Fatal("incorrect session")
				}
				return tc.storeErr
			}})
			svc.generateToken = func(u uuid.UUID, h int) (jwtPkg.Token, error) {
				if tc.lookupErr != nil {
					t.Fatal("token generation after lookup error")
				}
				if u != id || h != 2 {
					t.Fatal("incorrect claims")
				}
				return jwtPkg.Token{AccessToken: "access", RefreshToken: "refresh"}, tc.tokenErr
			}
			r := svc.RefreshToken(context.Background(), id)
			message := "refresh token failed"
			if tc.code == 200 {
				message = "refresh token success"
			}
			assertServiceResponse(t, r, tc.code, message)
			if stored != (tc.lookupErr == nil && tc.tokenErr == nil) {
				t.Fatal("unexpected store call")
			}
			if tc.code == 200 {
				data, ok := r.Data.(*domain.TokenResp)
				if !ok || data.AccessToken != "access" || data.RefreshToken != "refresh" || data.TokenType != "Bearer" || data.ExpiresIn != 7200 {
					t.Fatalf("data=%#v", r.Data)
				}
			}
		})
	}
}

// Characterizes an existing runtime bug without changing production behavior.
func TestRefreshMissingUserCurrentlyPanics(t *testing.T) {
	svc := newServiceForTest(&fakeUserRepository{findByIDFn: func(context.Context, uuid.UUID) (*domain.User, error) { return nil, ports.ErrUserNotFound }}, &fakeTokenRepository{})
	defer func() {
		if recover() == nil {
			t.Fatal("expected current nil-user panic; revise when runtime is fixed")
		}
	}()
	svc.RefreshToken(context.Background(), uuid.New())
}

func TestTokenTTLAndPair(t *testing.T) {
	for _, tc := range []struct {
		value string
		hours int
	}{{"", 24}, {"invalid", 24}, {"0", 24}, {"2", 2}, {"-1", -1}} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("JWT_EXPIRATION_HOURS", tc.value)
			id := uuid.New()
			svc := newServiceForTest(&fakeUserRepository{}, &fakeTokenRepository{storeFn: func(_ context.Context, u uuid.UUID, token string, ttl int) error {
				if u != id || token != "access" || ttl != tc.hours*3600 {
					t.Fatal("incorrect TTL or token")
				}
				return nil
			}})
			svc.generateToken = func(u uuid.UUID, h int) (jwtPkg.Token, error) {
				if u != id || h != tc.hours {
					t.Fatal("incorrect generation arguments")
				}
				return jwtPkg.Token{AccessToken: "access", RefreshToken: "refresh"}, nil
			}
			got, err := svc.generateAndStoreToken(context.Background(), id)
			if err != nil || got.RefreshToken != "refresh" || got.ExpiresIn != tc.hours*3600 {
				t.Fatalf("token=%#v err=%v", got, err)
			}
		})
	}
}
