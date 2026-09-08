package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	Issuer        string
	AccessSecret  []byte
	RefreshSecret []byte
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

func (m TokenManager) Issue(now time.Time, tokenType TokenType, tenantID, userID, deviceID, sessionID string) (string, error) {
	return m.IssueWithRole(now, tokenType, tenantID, userID, deviceID, sessionID, "")
}

func (m TokenManager) IssueWithRole(now time.Time, tokenType TokenType, tenantID, userID, deviceID, sessionID, role string) (string, error) {
	secret, ttl, err := m.secretAndTTL(tokenType)
	if err != nil {
		return "", err
	}
	if len(secret) < 32 {
		return "", fmt.Errorf("%s secret must be at least 32 bytes", tokenType)
	}
	now = now.UTC()
	claims := Claims{
		TenantID: tenantID, UserID: userID, DeviceID: deviceID, SessionID: sessionID, TokenType: tokenType, Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: m.Issuer, Subject: userID, IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)), ID: sessionID,
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func (m TokenManager) Parse(raw string, expectedType TokenType) (Claims, error) {
	secret, _, err := m.secretAndTTL(expectedType)
	if err != nil || len(secret) < 32 {
		return Claims{}, ErrInvalidToken
	}
	parsed, err := jwt.ParseWithClaims(raw, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return secret, nil
	}, jwt.WithIssuer(m.Issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !parsed.Valid {
		return Claims{}, ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || claims.TokenType != expectedType {
		return Claims{}, ErrInvalidToken
	}
	return *claims, nil
}

func (m TokenManager) secretAndTTL(tokenType TokenType) ([]byte, time.Duration, error) {
	switch tokenType {
	case AccessToken:
		return m.AccessSecret, m.AccessTTL, nil
	case RefreshToken:
		return m.RefreshSecret, m.RefreshTTL, nil
	default:
		return nil, 0, ErrInvalidToken
	}
}
