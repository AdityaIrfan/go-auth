package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"kda-auth-service/internal/core/domain"
	"kda-auth-service/pkg/config"
	"kda-auth-service/pkg/response"

	"github.com/google/uuid"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
)

type validatorAdapter struct{ validate *validator.Validate }

func (v validatorAdapter) Validate(value interface{}) error { return v.validate.Struct(value) }

type fakeAuthService struct {
	refreshFn   func(context.Context, uuid.UUID) response.Response
	registerFn  func(context.Context, domain.RegisterReq) response.Response
	loginFn     func(context.Context, domain.LoginReq) response.Response
	googleSSOFn func(context.Context, domain.GoogleSSOReq) response.Response
	logoutFn    func(context.Context, string) response.Response
}

func (f *fakeAuthService) Register(ctx context.Context, req domain.RegisterReq) response.Response {
	return f.registerFn(ctx, req)
}
func (f *fakeAuthService) Login(ctx context.Context, req domain.LoginReq) response.Response {
	return f.loginFn(ctx, req)
}
func (f *fakeAuthService) GoogleSSO(ctx context.Context, req domain.GoogleSSOReq) response.Response {
	return f.googleSSOFn(ctx, req)
}
func (f *fakeAuthService) Logout(ctx context.Context, token string) response.Response {
	return f.logoutFn(ctx, token)
}

func newHandlerContext(method, target, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	e.Validator = validatorAdapter{validate: validator.New()}
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func responseBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}

func TestAuthHandlerRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		called := false
		h := NewAuthHandler(&fakeAuthService{registerFn: func(_ context.Context, req domain.RegisterReq) response.Response {
			called = req.Email == "user@example.com" && req.Name == "Jane"
			return response.SuccessResponse(201, "register success, do login", nil)
		}})
		ctx, rec := newHandlerContext(http.MethodPost, "/api/v1/auth/register", `{"email":"user@example.com","password":"secret123","name":"Jane"}`)
		if err := h.Register(ctx); err != nil || rec.Code != http.StatusCreated || !called {
			t.Fatalf("Register() code=%d err=%v called=%v", rec.Code, err, called)
		}
		if responseBody(t, rec)["status"] != "success" {
			t.Fatal("expected success response")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{})
		ctx, rec := newHandlerContext(http.MethodPost, "/", `{`)
		if err := h.Register(ctx); err != nil || rec.Code != http.StatusBadRequest {
			t.Fatalf("code=%d err=%v", rec.Code, err)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{})
		ctx, rec := newHandlerContext(http.MethodPost, "/", `{"email":"bad","password":"123"}`)
		if err := h.Register(ctx); err != nil || rec.Code != http.StatusBadRequest {
			t.Fatalf("code=%d err=%v", rec.Code, err)
		}
		body := responseBody(t, rec)
		if body["message"] != "invalid request body" || body["errors"] != nil {
			t.Fatalf("unexpected response: %#v", body)
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{registerFn: func(context.Context, domain.RegisterReq) response.Response {
			return response.ErrorResponse(400, "email already registered", nil)
		}})
		ctx, rec := newHandlerContext(http.MethodPost, "/", `{"email":"user@example.com","password":"secret123","name":"Jane"}`)
		if err := h.Register(ctx); err != nil || rec.Code != 400 || responseBody(t, rec)["message"] != "email already registered" {
			t.Fatalf("code=%d err=%v", rec.Code, err)
		}
	})
}

func TestAuthHandlerLogin(t *testing.T) {
	valid := `{"email":"user@example.com","password":"secret123"}`
	t.Run("success", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{loginFn: func(context.Context, domain.LoginReq) response.Response {
			return response.SuccessResponse(200, "login success", &domain.TokenResp{AccessToken: "token", RefreshToken: "refresh", TokenType: "Bearer", ExpiresIn: 3600})
		}})
		ctx, rec := newHandlerContext(http.MethodPost, "/", valid)
		if err := h.Login(ctx); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("code=%d err=%v", rec.Code, err)
		}
		if responseBody(t, rec)["data"].(map[string]interface{})["access_token"] != "token" {
			t.Fatal("missing token")
		}
	})
	t.Run("malformed JSON", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{})
		ctx, rec := newHandlerContext(http.MethodPost, "/", `{`)
		_ = h.Login(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("code=%d", rec.Code)
		}
	})
	t.Run("validation error", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{})
		ctx, rec := newHandlerContext(http.MethodPost, "/", `{}`)
		_ = h.Login(ctx)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("code=%d", rec.Code)
		}
	})
	t.Run("invalid credentials", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{loginFn: func(context.Context, domain.LoginReq) response.Response {
			return response.ErrorResponse(401, "invalid credentials", nil)
		}})
		ctx, rec := newHandlerContext(http.MethodPost, "/", valid)
		_ = h.Login(ctx)
		if rec.Code != http.StatusUnauthorized || responseBody(t, rec)["message"] != "invalid credentials" {
			t.Fatalf("body=%s", rec.Body.String())
		}
	})
}

func TestAuthHandlerLogout(t *testing.T) {
	t.Run("success and strips bearer prefix", func(t *testing.T) {
		var token string
		h := NewAuthHandler(&fakeAuthService{logoutFn: func(_ context.Context, got string) response.Response {
			token = got
			return response.SuccessResponse(200, "logout success", nil)
		}})
		ctx, rec := newHandlerContext(http.MethodPost, "/", "")
		ctx.Request().Header.Set(echo.HeaderAuthorization, "Bearer abc")
		_ = h.Logout(ctx)
		if rec.Code != http.StatusOK || token != "abc" {
			t.Fatalf("code=%d token=%q", rec.Code, token)
		}
	})
	t.Run("service failure", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{logoutFn: func(context.Context, string) response.Response {
			return response.ErrorResponse(422, "logout failed", nil)
		}})
		ctx, rec := newHandlerContext(http.MethodPost, "/", "")
		_ = h.Logout(ctx)
		if rec.Code != 422 || responseBody(t, rec)["message"] != "logout failed" {
			t.Fatalf("body=%s", rec.Body.String())
		}
	})
}

type oauthTransportFunc func(*http.Request) (*http.Response, error)

func (f oauthTransportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func setGoogleOAuthConfig(t *testing.T) {
	t.Helper()
	original := config.GoogleOauthConfig
	config.GoogleOauthConfig = &oauth2.Config{
		ClientID:     "google-client-id",
		ClientSecret: "google-client-secret",
		RedirectURL:  "http://localhost:8080/api/v1/auth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.test/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.google.test/token",
		},
	}
	t.Cleanup(func() { config.GoogleOauthConfig = original })
}

func setDefaultHTTPClient(t *testing.T, status int, body string) {
	t.Helper()
	original := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: oauthTransportFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.String() != "https://oauth2.google.test/token" {
			t.Fatalf("unexpected OAuth request: %s %s", req.Method, req.URL)
		}
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})}
	t.Cleanup(func() { http.DefaultClient = original })
}

func TestAuthHandlerGoogleSSOLogin(t *testing.T) {
	setGoogleOAuthConfig(t)
	h := NewAuthHandler(&fakeAuthService{})
	ctx, rec := newHandlerContext(http.MethodGet, "/api/v1/auth/google/login", "")

	if err := h.GoogleSSOLogin(ctx); err != nil {
		t.Fatalf("GoogleSSOLogin() error = %v", err)
	}
	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d", rec.Code)
	}
	location, err := url.Parse(rec.Header().Get(echo.HeaderLocation))
	if err != nil {
		t.Fatalf("invalid Location header: %v", err)
	}
	query := location.Query()
	if location.Host != "accounts.google.test" || query.Get("client_id") != "google-client-id" || query.Get("response_type") != "code" {
		t.Fatalf("unexpected redirect URL: %s", location)
	}
	if query.Get("state") == "" || query.Get("access_type") != "offline" || query.Get("prompt") != "consent" {
		t.Fatalf("missing OAuth parameters: %s", location.RawQuery)
	}
	for _, scope := range []string{"openid", "email", "profile"} {
		if !strings.Contains(query.Get("scope"), scope) {
			t.Fatalf("scope %q missing from %q", scope, query.Get("scope"))
		}
	}
}

func TestGenerateStateOauthCookie(t *testing.T) {
	first := generateStateOauthCookie()
	second := generateStateOauthCookie()
	decoded, err := base64.URLEncoding.DecodeString(first)
	if err != nil || len(decoded) != 16 {
		t.Fatalf("state = %q, decoded bytes = %d, error = %v", first, len(decoded), err)
	}
	if first == second {
		t.Fatal("OAuth state values must be unique")
	}
}

func TestAuthHandlerGoogleSSOCallback(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		setGoogleOAuthConfig(t)
		setDefaultHTTPClient(t, http.StatusOK, `{"access_token":"google-access","token_type":"Bearer","id_token":"google-id-token"}`)
		var receivedIDToken string
		h := NewAuthHandler(&fakeAuthService{googleSSOFn: func(_ context.Context, req domain.GoogleSSOReq) response.Response {
			receivedIDToken = req.IDToken
			return response.SuccessResponse(200, "login success", &domain.TokenResp{AccessToken: "application-jwt", RefreshToken: "refresh", TokenType: "Bearer", ExpiresIn: 86400})
		}})
		ctx, rec := newHandlerContext(http.MethodGet, "/api/v1/auth/google/callback?code=authorization-code", "")
		_ = h.GoogleSSOCallback(ctx)
		if rec.Code != http.StatusOK || receivedIDToken != "google-id-token" {
			t.Fatalf("status=%d id_token=%q body=%s", rec.Code, receivedIDToken, rec.Body.String())
		}
		body := responseBody(t, rec)
		if body["message"] != "login success" || body["data"].(map[string]interface{})["access_token"] != "application-jwt" {
			t.Fatalf("body=%#v", body)
		}
	})

	t.Run("token exchange failure", func(t *testing.T) {
		setGoogleOAuthConfig(t)
		setDefaultHTTPClient(t, http.StatusBadRequest, `{"error":"invalid_grant"}`)
		h := NewAuthHandler(&fakeAuthService{})
		ctx, rec := newHandlerContext(http.MethodGet, "/api/v1/auth/google/callback?code=bad-code", "")
		_ = h.GoogleSSOCallback(ctx)
		if rec.Code != http.StatusUnauthorized || responseBody(t, rec)["message"] != "login failed" {
			t.Fatalf("body=%s", rec.Body.String())
		}
	})

	t.Run("missing id token", func(t *testing.T) {
		setGoogleOAuthConfig(t)
		setDefaultHTTPClient(t, http.StatusOK, `{"access_token":"google-access","token_type":"Bearer"}`)
		h := NewAuthHandler(&fakeAuthService{})
		ctx, rec := newHandlerContext(http.MethodGet, "/api/v1/auth/google/callback?code=authorization-code", "")
		_ = h.GoogleSSOCallback(ctx)
		if rec.Code != http.StatusUnauthorized || responseBody(t, rec)["message"] != "login failed" {
			t.Fatalf("body=%s", rec.Body.String())
		}
	})

	t.Run("service rejects identity", func(t *testing.T) {
		setGoogleOAuthConfig(t)
		setDefaultHTTPClient(t, http.StatusOK, `{"access_token":"google-access","token_type":"Bearer","id_token":"google-id-token"}`)
		h := NewAuthHandler(&fakeAuthService{googleSSOFn: func(context.Context, domain.GoogleSSOReq) response.Response {
			return response.ErrorResponse(422, "login failed", nil)
		}})
		ctx, rec := newHandlerContext(http.MethodGet, "/api/v1/auth/google/callback?code=authorization-code", "")
		_ = h.GoogleSSOCallback(ctx)
		if rec.Code != 422 || responseBody(t, rec)["message"] != "login failed" {
			t.Fatalf("body=%s", rec.Body.String())
		}
	})
}

func (f *fakeAuthService) RefreshToken(ctx context.Context, id uuid.UUID) response.Response {
	return f.refreshFn(ctx, id)
}
