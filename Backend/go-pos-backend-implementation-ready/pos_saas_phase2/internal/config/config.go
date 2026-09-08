package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv             string
	AppName            string
	HTTPAddr           string
	DatabaseURL        string
	RedisURL           string
	JWTIssuer          string
	JWTAudience        string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	JWTSecret          string
	CORSAllowedOrigins []string
	LoginRatePerMinute int
}

func Load() (Config, error) {
	accessTTL, err := time.ParseDuration(get("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return Config{}, err
	}
	refreshTTL, err := time.ParseDuration(get("JWT_REFRESH_TTL", "720h"))
	if err != nil {
		return Config{}, err
	}
	rate, err := strconv.Atoi(get("RATE_LIMIT_LOGIN_PER_MINUTE", "10"))
	if err != nil || rate < 1 {
		return Config{}, errors.New("invalid login rate limit")
	}

	cfg := Config{
		AppEnv:             get("APP_ENV", "development"),
		AppName:            get("APP_NAME", "pos-saas-api"),
		HTTPAddr:           get("HTTP_ADDR", ":8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		RedisURL:           os.Getenv("REDIS_URL"),
		JWTIssuer:          get("JWT_ISSUER", "pos-saas"),
		JWTAudience:        get("JWT_AUDIENCE", "pos-client"),
		JWTAccessTTL:       accessTTL,
		JWTRefreshTTL:      refreshTTL,
		JWTSecret:          os.Getenv("JWT_SECRET_CHANGE_ME"),
		CORSAllowedOrigins: split(get("CORS_ALLOWED_ORIGINS", "")),
		LoginRatePerMinute: rate,
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		return Config{}, errors.New("REDIS_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT secret must be at least 32 characters")
	}
	return cfg, nil
}

func get(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func split(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
