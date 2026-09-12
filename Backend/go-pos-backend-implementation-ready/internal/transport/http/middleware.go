package http

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/example/pos-api/internal/infrastructure/ratelimit"
	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = randomID()
		}
		c.Set(requestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		args := []any{
			"request_id", c.GetString(requestIDKey),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if claims, ok := Claims(c); ok {
			args = append(args,
				"tenant_id", claims.TenantID,
				"user_id", claims.UserID,
				"device_id", claims.DeviceID,
				"role", claims.Role,
			)
		}
		logger.Info("http request", args...)
	}
}

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		args := []any{"request_id", c.GetString(requestIDKey), "error", recovered}
		if claims, ok := Claims(c); ok {
			args = append(args, "tenant_id", claims.TenantID, "user_id", claims.UserID)
		}
		logger.Error("panic recovered", args...)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code": "internal_error", "message": "internal server error", "request_id": c.GetString(requestIDKey),
		}})
	})
}

func MaxBodySize(bytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, bytes)
		c.Next()
	}
}

// CORS returns a permissive-by-default CORS middleware. Pass the origins from
// CORS_ALLOWED_ORIGINS: the single entry "*" keeps the dev default (reflect the
// wildcard), otherwise only the listed origins are echoed back.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		allow := "*"
		if len(allowedOrigins) > 0 && !contains(allowedOrigins, "*") {
			origin := c.GetHeader("Origin")
			if origin == "" || !contains(allowedOrigins, origin) {
				c.Next()
				return
			}
			allow = origin
		}
		c.Header("Access-Control-Allow-Origin", allow)
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}

func LoginRateLimitWith(l ratelimit.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !l.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":       "rate_limited",
					"message":    "too many login attempts, please try again later",
					"request_id": c.GetString(requestIDKey),
				},
			})
			return
		}
		c.Next()
	}
}

func LoginRateLimit(limit int, window time.Duration) gin.HandlerFunc {
	return LoginRateLimitWith(ratelimit.NewMemory(limit, window))
}

func randomID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(value[:])
}
