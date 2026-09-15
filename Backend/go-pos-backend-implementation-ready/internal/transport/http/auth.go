package http

import (
	"net/http"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

const claimsKey = "auth_claims"
const saasAdminRole = "saas_admin"

func RequireAccessToken(tokens security.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := authenticate(c, tokens)
		if !ok {
			return
		}
		c.Set(claimsKey, claims)
		c.Next()
	}
}

// RequireSaasAdmin authenticates a Bearer access token and requires the
// saas_admin platform role. Every /v1/saas control-plane route is protected by
// it, so the web admin can only ever act as a platform operator.
func RequireSaasAdmin(tokens security.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := authenticate(c, tokens)
		if !ok {
			return
		}
		if claims.Role != saasAdminRole {
			abortError(c, http.StatusForbidden, "forbidden", "saas_admin role is required")
			return
		}
		c.Set(claimsKey, claims)
		c.Next()
	}
}

func authenticate(c *gin.Context, tokens security.TokenManager) (security.Claims, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		abortError(c, http.StatusUnauthorized, "unauthorized", "authorization is required")
		return security.Claims{}, false
	}
	claims, err := tokens.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), security.AccessToken)
	if err != nil {
		abortError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return security.Claims{}, false
	}
	return claims, true
}

func Claims(c *gin.Context) (security.Claims, bool) {
	value, exists := c.Get(claimsKey)
	if !exists {
		return security.Claims{}, false
	}
	claims, ok := value.(security.Claims)
	return claims, ok
}

func abortError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{
		"code": code, "message": message, "request_id": c.GetString(requestIDKey),
	}})
}
