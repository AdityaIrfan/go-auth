package jwt

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestGenerateToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	userID := uuid.New()
	tokenString, err := GenerateToken(userID, 2)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims := new(CustomClaims)
	token, err := jwtlib.ParseWithClaims(tokenString, claims, func(*jwtlib.Token) (interface{}, error) { return []byte("test-secret"), nil })
	if err != nil || !token.Valid {
		t.Fatalf("ParseWithClaims() token=%v error=%v", token.Valid, err)
	}
	if claims.UserID != userID {
		t.Fatalf("user_id=%v, want %v", claims.UserID, userID)
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil {
		t.Fatal("missing registered time claims")
	}
	duration := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if duration < 2*time.Hour-time.Second || duration > 2*time.Hour+time.Second {
		t.Fatalf("token duration=%v", duration)
	}
}
