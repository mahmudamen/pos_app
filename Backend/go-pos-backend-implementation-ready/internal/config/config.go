package config

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName             string
	HTTPAddr            string
	HTTPReadTimeout     time.Duration
	HTTPWriteTimeout    time.Duration
	HTTPIdleTimeout     time.Duration
	HTTPShutdownTimeout time.Duration
	HTTPMaxBodyBytes    int64
	DatabaseURL         string
	DBMaxConns          int32
	DBMinConns          int32
	DBMaxConnLifetime   time.Duration
	DBMaxConnIdleTime   time.Duration
	RedisAddr           string
	RedisPassword       string
	RedisDB             int
	JWTIssuer           string
	JWTAccessSecret     string
	JWTRefreshSecret    string
	JWTAccessTTL        time.Duration
	JWTRefreshTTL       time.Duration
	BcryptCost          int
	CashierDiscountPct  int
	CORSAllowedOrigins  []string
	LogLevel            string
	LogFormat           string
	MaxSessionsPerUser  int
	LoginRateMax        int
	LoginRateWindow     time.Duration
	ApiRateLimitEnabled bool
	ApiRateLimitMax     int
	ApiRateLimitWindow  time.Duration
	MetricsEnabled      bool
}

func Load() (Config, error) {
	c := Config{
		AppName:          envOr("APP_NAME", "pos-api"),
		HTTPAddr:         envOr("HTTP_ADDR", ":8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		RedisAddr:        envOr("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:    os.Getenv("REDIS_PASSWORD"),
		JWTIssuer:        envOr("JWT_ISSUER", "pos-api"),
		JWTAccessSecret:  os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
		LogLevel:         envOr("LOG_LEVEL", "info"),
		LogFormat:        envOr("LOG_FORMAT", "json"),
	}
	var err error
	if c.HTTPReadTimeout, err = duration("HTTP_READ_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if c.HTTPWriteTimeout, err = duration("HTTP_WRITE_TIMEOUT", 15*time.Second); err != nil {
		return Config{}, err
	}
	if c.HTTPIdleTimeout, err = duration("HTTP_IDLE_TIMEOUT", 60*time.Second); err != nil {
		return Config{}, err
	}
	if c.HTTPShutdownTimeout, err = duration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second); err != nil {
		return Config{}, err
	}
	if c.DBMaxConnLifetime, err = duration("DB_MAX_CONN_LIFETIME", 30*time.Minute); err != nil {
		return Config{}, err
	}
	if c.DBMaxConnIdleTime, err = duration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute); err != nil {
		return Config{}, err
	}
	if c.JWTAccessTTL, err = duration("JWT_ACCESS_TTL", 15*time.Minute); err != nil {
		return Config{}, err
	}
	if c.JWTRefreshTTL, err = duration("JWT_REFRESH_TTL", 7*24*time.Hour); err != nil {
		return Config{}, err
	}
	if c.LoginRateWindow, err = duration("LOGIN_RATE_WINDOW", 5*time.Minute); err != nil {
		return Config{}, err
	}
	if c.ApiRateLimitWindow, err = duration("RATE_LIMIT_WINDOW", 1*time.Minute); err != nil {
		return Config{}, err
	}
	bcryptCost, err := intValue("BCRYPT_COST", 12)
	if err != nil {
		return Config{}, err
	}
	if c.HTTPMaxBodyBytes, err = int64Value("HTTP_MAX_BODY_BYTES", 1<<20); err != nil {
		return Config{}, err
	}
	maxConns, err := intValue("DB_MAX_CONNS", 20)
	if err != nil {
		return Config{}, err
	}
	minConns, err := intValue("DB_MIN_CONNS", 2)
	if err != nil {
		return Config{}, err
	}
	redisDB, err := intValue("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}
	if maxConns < 1 || maxConns > math.MaxInt32 {
		return Config{}, fmt.Errorf("DB_MAX_CONNS out of range: %d", maxConns)
	}
	if minConns < 0 || minConns > maxConns {
		return Config{}, fmt.Errorf("invalid database pool sizes: min=%d max=%d", minConns, maxConns)
	}
	if redisDB < 0 || redisDB > math.MaxInt32 {
		return Config{}, fmt.Errorf("REDIS_DB out of range: %d", redisDB)
	}
	if c.HTTPMaxBodyBytes < 1 {
		return Config{}, fmt.Errorf("HTTP_MAX_BODY_BYTES must be positive")
	}
	if c.JWTAccessTTL <= 0 || c.JWTRefreshTTL <= 0 || c.JWTRefreshTTL <= c.JWTAccessTTL {
		return Config{}, fmt.Errorf("JWT refresh TTL must be greater than access TTL")
	}
	if bcryptCost < 4 || bcryptCost > 31 {
		return Config{}, fmt.Errorf("BCRYPT_COST must be between 4 and 31")
	}
	cashierDiscountPct, err := intValue("CASHIER_DISCOUNT_PCT", 5)
	if err != nil {
		return Config{}, err
	}
	if cashierDiscountPct < 0 || cashierDiscountPct > 100 {
		return Config{}, fmt.Errorf("CASHIER_DISCOUNT_PCT must be between 0 and 100")
	}
	maxSessions, err := intValue("MAX_SESSIONS_PER_USER", 5)
	if err != nil {
		return Config{}, err
	}
	if maxSessions < 1 {
		return Config{}, fmt.Errorf("MAX_SESSIONS_PER_USER must be at least 1")
	}
	loginRateMax, err := intValue("LOGIN_RATE_MAX", 5)
	if err != nil {
		return Config{}, err
	}
	if loginRateMax < 1 {
		return Config{}, fmt.Errorf("LOGIN_RATE_MAX must be at least 1")
	}
	apiRateLimitMax, err := intValue("RATE_LIMIT_MAX", 300)
	if err != nil {
		return Config{}, err
	}
	if apiRateLimitMax < 1 {
		return Config{}, fmt.Errorf("RATE_LIMIT_MAX must be at least 1")
	}
	apiRateLimitEnabled, err := boolValue("RATE_LIMIT_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	// #nosec G115 -- all four values are range-checked above
	// (maxConns <= math.MaxInt32, minConns <= maxConns, redisDB <= MaxInt32,
	// bcryptCost in 4..31).
	c.DBMaxConns, c.DBMinConns, c.RedisDB, c.BcryptCost = int32(maxConns), int32(minConns), redisDB, bcryptCost
	c.CashierDiscountPct = cashierDiscountPct
	c.MaxSessionsPerUser = maxSessions
	c.LoginRateMax = loginRateMax
	c.ApiRateLimitMax = apiRateLimitMax
	c.ApiRateLimitEnabled = apiRateLimitEnabled
	c.CORSAllowedOrigins = stringList("CORS_ALLOWED_ORIGINS", []string{"*"})
	metricsEnabled, err := boolValue("METRICS_ENABLED", true)
	if err != nil {
		return Config{}, err
	}
	c.MetricsEnabled = metricsEnabled
	return c, nil
}

// boolValue parses a plain bool env var; an empty or unset variable returns
// the fallback, and a non-boolean value is an error.
func boolValue(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func duration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func intValue(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

// stringList parses a comma-separated env var into a trimmed slice. An empty
// or unset variable returns the fallback. Entries are kept as written;
// "*" is the dev-only wildcard.
func stringList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func int64Value(key string, fallback int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}
