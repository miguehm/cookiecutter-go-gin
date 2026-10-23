package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestGenerateAndVerifyJwtToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	token, err := GenerateJwtToken("test-secret", userID)
	if err != nil {
		t.Fatalf("GenerateJwtToken returned error: %v", err)
	}

	claims, err := VerifyJwtToken("test-secret", token)
	if err != nil {
		t.Fatalf("VerifyJwtToken returned error: %v", err)
	}

	if claims.UserID != userID {
		t.Fatalf("UserID = %s, want %s", claims.UserID, userID)
	}
}

func TestVerifyJwtTokenRejectsWrongSecret(t *testing.T) {
	t.Parallel()

	token, err := GenerateJwtToken("test-secret", uuid.New())
	if err != nil {
		t.Fatalf("GenerateJwtToken returned error: %v", err)
	}

	if _, err := VerifyJwtToken("other-secret", token); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestVerifyJwtTokenRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	claims := Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	if _, err := VerifyJwtToken("test-secret", tokenString); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyJwtTokenRejectsMalformedToken(t *testing.T) {
	t.Parallel()

	if _, err := VerifyJwtToken("test-secret", "not-a-token"); err == nil {
		t.Fatal("expected error for malformed token")
	}
}
