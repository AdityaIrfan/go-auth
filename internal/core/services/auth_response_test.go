package services

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"kda-auth-service/internal/core/domain"
	jwtPkg "kda-auth-service/pkg/jwt"
	"net/http"
	"testing"
)

func TestAuthTokenFailuresReturnServiceEnvelope(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	for _, google := range []bool{false, true} {
		for _, storeFailure := range []bool{false, true} {
			t.Setenv("GOOGLE_CLIENT_ID", "client")
			svc := newServiceForTest(&fakeUserRepository{findByEmailFn: func(context.Context, string) (*domain.User, error) {
				return &domain.User{ID: uuid.New(), Provider: "email", Password: string(hash)}, nil
			}}, &fakeTokenRepository{storeFn: func(context.Context, uuid.UUID, string, int) error { return errors.New("redis") }})
			svc.generateToken = func(uuid.UUID, int) (jwtPkg.Token, error) {
				if !storeFailure {
					return jwtPkg.Token{}, errors.New("signing")
				}
				return jwtPkg.Token{AccessToken: "access", RefreshToken: "refresh"}, nil
			}
			if google {
				svc.httpClient = roundTripFunc(func(*http.Request) (*http.Response, error) {
					return jsonResponse(200, `{"email":"user@example.com","sub":"1","aud":"client"}`), nil
				})
				assertServiceResponse(t, svc.GoogleSSO(context.Background(), domain.GoogleSSOReq{IDToken: "valid"}), 422, "login failed")
			} else {
				assertServiceResponse(t, svc.Login(context.Background(), domain.LoginReq{Email: "user@example.com", Password: "secret123"}), 422, "login failed")
			}
		}
	}
}

func TestLogoutResponse(t *testing.T) {
	svc := newServiceForTest(&fakeUserRepository{}, &fakeTokenRepository{revokeFn: func(_ context.Context, token string) error {
		if token != "access" {
			t.Fatal("wrong token")
		}
		return nil
	}})
	r := svc.Logout(context.Background(), "access")
	assertServiceResponse(t, r, 200, "logout success")
	if r.Data != nil {
		t.Fatal("logout must omit data")
	}
}
