package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"kda-auth-service/internal/core/domain"
	"kda-auth-service/internal/core/ports"
	jwtPkg "kda-auth-service/pkg/jwt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepository struct {
	findByIDFn    func(context.Context, uuid.UUID) (*domain.User, error)
	createFn      func(context.Context, *domain.User) error
	findByEmailFn func(context.Context, string) (*domain.User, error)
}

func (f *fakeUserRepository) Create(ctx context.Context, user *domain.User) error {
	return f.createFn(ctx, user)
}

func (f *fakeUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return f.findByEmailFn(ctx, email)
}

type fakeTokenRepository struct {
	storeFn  func(context.Context, uuid.UUID, string, int) error
	revokeFn func(context.Context, string) error
}

func (f *fakeTokenRepository) Store(ctx context.Context, userID uuid.UUID, token string, ttl int) error {
	return f.storeFn(ctx, userID, token, ttl)
}

func (f *fakeTokenRepository) Revoke(ctx context.Context, token string) error {
	return f.revokeFn(ctx, token)
}

func (f *fakeTokenRepository) Validate(context.Context, string) (bool, error) { return true, nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func newServiceForTest(ur *fakeUserRepository, tr *fakeTokenRepository) *authService {
	return NewAuthService(ur, tr).(*authService)
}

func TestRegister(t *testing.T) {
	t.Run("creates an email user with hashed password", func(t *testing.T) {
		var created *domain.User
		svc := newServiceForTest(&fakeUserRepository{
			findByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, ports.ErrUserNotFound },
			createFn:      func(_ context.Context, user *domain.User) error { created = user; return nil },
		}, &fakeTokenRepository{})

		err := svc.Register(context.Background(), domain.RegisterReq{Email: "user@example.com", Password: "secret123", Name: "Jane"})
		assertServiceResponse(t, err, 201, "register success, do login")
		if err.StatusCode != 201 {
			t.Fatalf("Register() error = %v", err)
		}
		if created == nil || created.ID == uuid.Nil || created.Provider != "email" || created.Email != "user@example.com" {
			t.Fatalf("unexpected created user: %#v", created)
		}
		if bcrypt.CompareHashAndPassword([]byte(created.Password), []byte("secret123")) != nil {
			t.Fatal("password was not bcrypt hashed")
		}
	})

	t.Run("rejects duplicate email", func(t *testing.T) {
		svc := newServiceForTest(&fakeUserRepository{
			findByEmailFn: func(context.Context, string) (*domain.User, error) { return &domain.User{}, nil },
			createFn:      func(context.Context, *domain.User) error { t.Fatal("Create should not be called"); return nil },
		}, &fakeTokenRepository{})
		if err := svc.Register(context.Background(), domain.RegisterReq{Password: "secret123"}); err.StatusCode != 400 || err.Message != "email already registered" {
			t.Fatalf("Register() error = %v", err)
		}
	})

	t.Run("returns password hashing error", func(t *testing.T) {
		svc := newServiceForTest(&fakeUserRepository{
			findByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, ports.ErrUserNotFound },
			createFn:      func(context.Context, *domain.User) error { return nil },
		}, &fakeTokenRepository{})
		if err := svc.Register(context.Background(), domain.RegisterReq{Password: strings.Repeat("x", 73)}); err.StatusCode != 422 {
			t.Fatal("Register() expected bcrypt error")
		}
	})

	t.Run("returns lookup error", func(t *testing.T) {
		want := errors.New("database unavailable")
		svc := newServiceForTest(&fakeUserRepository{
			findByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, want },
			createFn:      func(context.Context, *domain.User) error { t.Fatal("Create should not be called"); return nil },
		}, &fakeTokenRepository{})
		if err := svc.Register(context.Background(), domain.RegisterReq{Password: "secret123"}); err.StatusCode != 422 {
			t.Fatalf("Register() error = %v, want %v", err, want)
		}
	})

	t.Run("returns create error", func(t *testing.T) {
		want := errors.New("database unavailable")
		svc := newServiceForTest(&fakeUserRepository{
			findByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, ports.ErrUserNotFound },
			createFn:      func(context.Context, *domain.User) error { return want },
		}, &fakeTokenRepository{})
		if err := svc.Register(context.Background(), domain.RegisterReq{Password: "secret123"}); err.StatusCode != 422 {
			t.Fatalf("Register() error = %v, want %v", err, want)
		}
	})
}

func TestLogin(t *testing.T) {
	userID := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	validUser := &domain.User{ID: userID, Email: "user@example.com", Password: string(hash), Provider: "email"}

	t.Run("returns and stores token", func(t *testing.T) {
		t.Setenv("JWT_EXPIRATION_HOURS", "2")
		var storedTTL int
		svc := newServiceForTest(&fakeUserRepository{
			findByEmailFn: func(context.Context, string) (*domain.User, error) { return validUser, nil },
			createFn:      func(context.Context, *domain.User) error { return nil },
		}, &fakeTokenRepository{
			storeFn: func(_ context.Context, gotID uuid.UUID, token string, ttl int) error {
				if gotID != userID || token != "signed-token" {
					t.Fatalf("unexpected Store args: %v %q", gotID, token)
				}
				storedTTL = ttl
				return nil
			},
		})
		svc.generateToken = func(id uuid.UUID, hours int) (jwtPkg.Token, error) {
			if id != userID || hours != 2 {
				t.Fatalf("unexpected token args: %v %d", id, hours)
			}
			return jwtPkg.Token{AccessToken: "signed-token", RefreshToken: "refresh"}, nil
		}

		res := svc.Login(context.Background(), domain.LoginReq{Email: validUser.Email, Password: "secret123"})
		assertServiceResponse(t, res, 200, "login success")
		got, ok := res.Data.(*domain.TokenResp)
		if res.StatusCode != 200 || !ok || got.AccessToken != "signed-token" || got.RefreshToken != "refresh" || got.TokenType != "Bearer" || got.ExpiresIn != 7200 || storedTTL != 7200 {
			t.Fatalf("Login() = %#v, %v; ttl=%d", got, res, storedTTL)
		}
	})

	for _, tc := range []struct {
		name string
		user *domain.User
		err  error
		pass string
	}{
		{name: "missing user", err: errors.New("not found"), pass: "secret123"},
		{name: "nil user", pass: "secret123"},
		{name: "non-email provider", user: &domain.User{Provider: "google"}, pass: "secret123"},
		{name: "wrong password", user: validUser, pass: "wrong"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := newServiceForTest(&fakeUserRepository{
				findByEmailFn: func(context.Context, string) (*domain.User, error) { return tc.user, tc.err },
				createFn:      func(context.Context, *domain.User) error { return nil },
			}, &fakeTokenRepository{})
			if err := svc.Login(context.Background(), domain.LoginReq{Password: tc.pass}); err.StatusCode != 401 || err.Message != "invalid credentials" {
				t.Fatalf("Login() error = %v", err)
			}
		})
	}
}

func TestGoogleSSO(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-123")
	userID := uuid.New()

	t.Run("uses existing user", func(t *testing.T) {
		svc := newServiceForTest(&fakeUserRepository{
			findByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
				if email != "user@example.com" {
					t.Fatalf("email = %q", email)
				}
				return &domain.User{ID: userID}, nil
			},
			createFn: func(context.Context, *domain.User) error { t.Fatal("Create should not be called"); return nil },
		}, &fakeTokenRepository{storeFn: func(context.Context, uuid.UUID, string, int) error { return nil }})
		svc.httpClient = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Query().Get("id_token") != "id token/+" {
				t.Fatalf("id_token was not encoded correctly: %s", req.URL.RawQuery)
			}
			return jsonResponse(http.StatusOK, `{"email":"user@example.com","name":"Jane","sub":"google-1","aud":"client-123"}`), nil
		})
		svc.generateToken = func(uuid.UUID, int) (jwtPkg.Token, error) {
			return jwtPkg.Token{AccessToken: "token", RefreshToken: "refresh"}, nil
		}

		res := svc.GoogleSSO(context.Background(), domain.GoogleSSOReq{IDToken: "id token/+"})
		assertServiceResponse(t, res, 200, "login success")
		got, ok := res.Data.(*domain.TokenResp)
		if res.StatusCode != 200 || !ok || got.AccessToken != "token" {
			t.Fatalf("GoogleSSO() = %#v, %v", got, res)
		}
	})

	t.Run("auto registers new user", func(t *testing.T) {
		var created *domain.User
		svc := newServiceForTest(&fakeUserRepository{
			findByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, ports.ErrUserNotFound },
			createFn:      func(_ context.Context, user *domain.User) error { user.ID = userID; created = user; return nil },
		}, &fakeTokenRepository{storeFn: func(context.Context, uuid.UUID, string, int) error { return nil }})
		svc.httpClient = roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"email":"new@example.com","name":"New User","sub":"google-2","aud":"client-123"}`), nil
		})
		svc.generateToken = func(uuid.UUID, int) (jwtPkg.Token, error) {
			return jwtPkg.Token{AccessToken: "token", RefreshToken: "refresh"}, nil
		}

		err := svc.GoogleSSO(context.Background(), domain.GoogleSSOReq{IDToken: "valid"})
		if err.StatusCode != 200 || created == nil || created.Provider != "google" || created.GoogleID != "valid" {
			t.Fatalf("GoogleSSO() created = %#v, error = %v", created, err)
		}
	})

	tests := []struct {
		name          string
		panicExpected bool
		client        httpDoer
	}{
		{name: "network error", client: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })},
		{name: "non-200 response", client: roundTripFunc(func(*http.Request) (*http.Response, error) { return jsonResponse(http.StatusBadRequest, `{}`), nil })},
		{name: "invalid JSON", client: roundTripFunc(func(*http.Request) (*http.Response, error) { return jsonResponse(http.StatusOK, `{`), nil })},
		{name: "missing claims", panicExpected: true, client: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"aud":"client-123"}`), nil
		})},
		{name: "wrong audience", client: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"email":"a@b.com","sub":"1","aud":"other"}`), nil
		})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newServiceForTest(&fakeUserRepository{findByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, nil }, createFn: func(context.Context, *domain.User) error { return nil }}, &fakeTokenRepository{})
			svc.httpClient = tc.client
			if tc.panicExpected {
				defer func() {
					if recover() == nil {
						t.Fatal("current missing-claims path should panic; update this characterization when runtime is fixed")
					}
				}()
			}
			if err := svc.GoogleSSO(context.Background(), domain.GoogleSSOReq{IDToken: "bad"}); err.StatusCode != 422 || err.Message != "login failed" {
				t.Fatalf("GoogleSSO() error = %v", err)
			}
		})
	}
}

func TestGoogleSSOCreateFailure(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "")
	want := errors.New("create failed")
	svc := newServiceForTest(&fakeUserRepository{
		findByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, ports.ErrUserNotFound },
		createFn:      func(context.Context, *domain.User) error { return want },
	}, &fakeTokenRepository{})
	svc.httpClient = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"email":"new@example.com","sub":"google-2"}`), nil
	})
	if err := svc.GoogleSSO(context.Background(), domain.GoogleSSOReq{IDToken: "valid"}); err.StatusCode != 422 {
		t.Fatalf("GoogleSSO() error = %v", err)
	}
}

func TestGoogleSSOConfigurationAndRepositoryFailures(t *testing.T) {
	t.Run("invalid tokeninfo URL", func(t *testing.T) {
		svc := newServiceForTest(&fakeUserRepository{}, &fakeTokenRepository{})
		svc.googleTokenInfoURL = "://invalid-url"
		if err := svc.GoogleSSO(context.Background(), domain.GoogleSSOReq{IDToken: "token"}); err.StatusCode != 422 || err.Message != "login failed" {
			t.Fatalf("GoogleSSO() error = %v", err)
		}
	})

	t.Run("user lookup error", func(t *testing.T) {
		t.Setenv("GOOGLE_CLIENT_ID", "client-123")
		want := errors.New("database unavailable")
		svc := newServiceForTest(&fakeUserRepository{
			findByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, want },
			createFn:      func(context.Context, *domain.User) error { t.Fatal("Create should not be called"); return nil },
		}, &fakeTokenRepository{})
		svc.httpClient = roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"email":"user@example.com","sub":"google-1","aud":"client-123"}`), nil
		})
		if err := svc.GoogleSSO(context.Background(), domain.GoogleSSOReq{IDToken: "valid"}); err.StatusCode != 422 {
			t.Fatalf("GoogleSSO() error = %v, want %v", err, want)
		}
	})
}

func TestTokenFailuresAndLogout(t *testing.T) {
	userID := uuid.New()
	t.Setenv("JWT_EXPIRATION_HOURS", "invalid")
	want := errors.New("token failure")
	svc := newServiceForTest(&fakeUserRepository{}, &fakeTokenRepository{})
	svc.generateToken = func(uuid.UUID, int) (jwtPkg.Token, error) {
		return jwtPkg.Token{AccessToken: "", RefreshToken: "refresh"}, want
	}
	if _, err := svc.generateAndStoreToken(context.Background(), userID); !errors.Is(err, want) {
		t.Fatalf("generateAndStoreToken() error = %v", err)
	}

	svc.generateToken = func(uuid.UUID, int) (jwtPkg.Token, error) {
		return jwtPkg.Token{AccessToken: "token", RefreshToken: "refresh"}, nil
	}
	svc.tokenRepo = &fakeTokenRepository{storeFn: func(context.Context, uuid.UUID, string, int) error { return want }}
	if _, err := svc.generateAndStoreToken(context.Background(), userID); !errors.Is(err, want) {
		t.Fatalf("generateAndStoreToken() store error = %v", err)
	}

	var revoked string
	svc.tokenRepo = &fakeTokenRepository{revokeFn: func(_ context.Context, token string) error { revoked = token; return want }}
	if err := svc.Logout(context.Background(), "abc"); err.StatusCode != 422 || revoked != "abc" {
		t.Fatalf("Logout() = %v, revoked %q", err, revoked)
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func (f *fakeUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return f.findByIDFn(ctx, id)
}
