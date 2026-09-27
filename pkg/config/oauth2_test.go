package config

import (
	"testing"

	"golang.org/x/oauth2/google"
)

func TestGoogleConfig(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback")

	GoogleConfig()

	if GoogleOauthConfig == nil {
		t.Fatal("GoogleOauthConfig is nil")
	}
	if GoogleOauthConfig.ClientID != "client-id" || GoogleOauthConfig.ClientSecret != "client-secret" {
		t.Fatalf("unexpected credentials: %#v", GoogleOauthConfig)
	}
	if GoogleOauthConfig.RedirectURL != "http://localhost:8080/api/v1/auth/google/callback" {
		t.Fatalf("RedirectURL = %q", GoogleOauthConfig.RedirectURL)
	}
	if GoogleOauthConfig.Endpoint != google.Endpoint {
		t.Fatalf("Endpoint = %#v", GoogleOauthConfig.Endpoint)
	}
	wantScopes := []string{"openid", "email", "profile"}
	for i, want := range wantScopes {
		if len(GoogleOauthConfig.Scopes) <= i || GoogleOauthConfig.Scopes[i] != want {
			t.Fatalf("Scopes = %#v", GoogleOauthConfig.Scopes)
		}
	}
}
