package http

import (
	"net/http"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

const claimsKey = "auth_claims"

func RequireAccessToken(tokens security.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			abortError(c, http.StatusUnauthorized, "unauthorized", "authorization is required")
			return
		}
		claims, err := tokens.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), security.AccessToken)
		if err != nil {
			abortError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
			return
		}
		c.Set(claimsKey, claims)
		c.Next()
	}
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
