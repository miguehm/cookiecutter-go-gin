package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func testOpts() JwtOptions {
	return JwtOptions{Issuer: "go-gin-template", Audience: "go-gin-template", Expiration: time.Hour, Leeway: 30 * time.Second}
}

func TestGenerateAndVerifyJwtToken(t *testing.T) {
	userID := uuid.New()
	token, err := GenerateJwtToken("test-secret-0123456789abcdef", userID, testOpts())
	if err != nil {
		t.Fatalf("GenerateJwtToken returned error: %v", err)
	}

	claims, err := VerifyJwtToken("test-secret-0123456789abcdef", token, testOpts())
	if err != nil {
		t.Fatalf("VerifyJwtToken returned error: %v", err)
	}

	if claims.UserID != userID {
		t.Fatalf("UserID = %s, want %s", claims.UserID, userID)
	}
	if claims.Issuer != "go-gin-template" {
		t.Fatalf("Issuer = %s", claims.Issuer)
	}
	if len(claims.Audience) == 0 || claims.Audience[0] != "go-gin-template" {
		t.Fatalf("Audience = %v", claims.Audience)
	}
	if claims.ID == "" {
		t.Fatal("expected jti")
	}
}

func TestVerifyJwtTokenRejectsWrongSecret(t *testing.T) {
	token, err := GenerateJwtToken("test-secret-0123456789abcdef", uuid.New(), testOpts())
	if err != nil {
		t.Fatalf("GenerateJwtToken returned error: %v", err)
	}

	if _, err := VerifyJwtToken("other-secret-0123456789abcdef", token, testOpts()); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestVerifyJwtTokenRejectsExpiredToken(t *testing.T) {
	claims := Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "go-gin-template",
			Audience:  jwt.ClaimStrings{"go-gin-template"},
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret-0123456789abcdef"))
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	if _, err := VerifyJwtToken("test-secret-0123456789abcdef", tokenString, testOpts()); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyJwtTokenRejectsMalformedToken(t *testing.T) {
	if _, err := VerifyJwtToken("test-secret-0123456789abcdef", "not-a-token", testOpts()); err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestVerifyJwtTokenRejectsWrongIssuer(t *testing.T) {
	claims := Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "wrong",
			Audience:  jwt.ClaimStrings{"go-gin-template"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret-0123456789abcdef"))
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	if _, err := VerifyJwtToken("test-secret-0123456789abcdef", tokenString, testOpts()); err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestVerifyJwtTokenRejectsNoneAlgorithm(t *testing.T) {
	claims := Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "go-gin-template",
			Audience:  jwt.ClaimStrings{"go-gin-template"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	if _, err := VerifyJwtToken("test-secret-0123456789abcdef", tokenString, testOpts()); err == nil {
		t.Fatal("expected error for none algorithm")
	}
}
