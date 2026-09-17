package security

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTService struct {
	Secret    []byte
	Issuer    string
	Audience  string
	AccessTTL time.Duration
}

type Claims struct {
	SessionID string `json:"sid"`
	TenantID  string `json:"tenant_id"`
	jwt.RegisteredClaims
}

func (s JWTService) CreateAccessToken(userID, sessionID, tenantID string) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		SessionID: sessionID,
		TenantID:  tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   userID,
			Issuer:    s.Issuer,
			Audience:  jwt.ClaimStrings{s.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.AccessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.Secret)
}

func (s JWTService) ParseAccessToken(raw string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.Secret, nil
	}, jwt.WithIssuer(s.Issuer), jwt.WithAudience(s.Audience))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
