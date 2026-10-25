// Package auth creates and verifies the JWTs used by the API.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims are the JWT claims of this API, carrying the user ID.
type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID `json:"userId"`
}

// JwtOptions overrides the defaults used when generating or verifying a
// token; zero values keep the default.
type JwtOptions struct {
	Issuer     string
	Audience   string
	Expiration time.Duration
	Leeway     time.Duration
}

// GenerateJwtToken signs an HS256 token for userID, applying opts on top of
// the default issuer, audience, expiration and leeway.
func GenerateJwtToken(secret string, userID uuid.UUID, opts ...JwtOptions) (string, error) {
	o := JwtOptions{
		Issuer:     "go-gin-template",
		Audience:   "go-gin-template",
		Expiration: 24 * time.Hour,
		Leeway:     30 * time.Second,
	}
	if len(opts) > 0 {
		if opts[0].Issuer != "" {
			o.Issuer = opts[0].Issuer
		}
		if opts[0].Audience != "" {
			o.Audience = opts[0].Audience
		}
		if opts[0].Expiration > 0 {
			o.Expiration = opts[0].Expiration
		}
		if opts[0].Leeway > 0 {
			o.Leeway = opts[0].Leeway
		}
	}

	now := time.Now()
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    o.Issuer,
			Audience:  jwt.ClaimStrings{o.Audience},
			Subject:   userID.String(),
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(o.Expiration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// VerifyJwtToken parses tokenString, checking the signature, the issuer, the
// audience, the expiration and the leeway, and returns its claims.
func VerifyJwtToken(secret string, tokenString string, opts ...JwtOptions) (*Claims, error) {
	o := JwtOptions{
		Issuer:   "go-gin-template",
		Audience: "go-gin-template",
		Leeway:   30 * time.Second,
	}
	if len(opts) > 0 {
		if opts[0].Issuer != "" {
			o.Issuer = opts[0].Issuer
		}
		if opts[0].Audience != "" {
			o.Audience = opts[0].Audience
		}
		if opts[0].Leeway > 0 {
			o.Leeway = opts[0].Leeway
		}
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(o.Issuer),
		jwt.WithAudience(o.Audience),
		jwt.WithLeeway(o.Leeway),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
