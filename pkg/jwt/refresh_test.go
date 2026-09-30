package jwt

import (
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestUnpackRefreshToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	id := uuid.New()
	pair, err := GenerateToken(id, 2)
	if err != nil {
		t.Fatal(err)
	}
	claims := new(CustomClaims)
	if _, err := jwtlib.ParseWithClaims(pair.RefreshToken, claims, func(*jwtlib.Token) (interface{}, error) { return []byte("test-secret"), nil }); err != nil {
		t.Fatal(err)
	}
	if claims.ExpiresAt.Sub(claims.IssuedAt.Time) != 3*time.Hour {
		t.Fatal("refresh lifetime must be access lifetime plus one hour")
	}
	expired, err := GenerateToken(id, -2)
	if err != nil {
		t.Fatal(err)
	}
	none, err := jwtlib.NewWithClaims(jwtlib.SigningMethodNone, CustomClaims{UserID: id}).SignedString(jwtlib.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, token string
		valid       bool
	}{
		{"valid refresh", pair.RefreshToken, true}, {"access token currently accepted", pair.AccessToken, true},
		{"bad signature", pair.RefreshToken + "x", false}, {"malformed", "bad", false},
		{"expired", expired.RefreshToken, false}, {"non HMAC", none, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := UnpackRefreshToken(tc.token)
			if tc.valid {
				if err != nil || got != id {
					t.Fatalf("id=%v err=%v", got, err)
				}
			} else if err == nil {
				t.Fatal("invalid token accepted")
			}
		})
	}
}
