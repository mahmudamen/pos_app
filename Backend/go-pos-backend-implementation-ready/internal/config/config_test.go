package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadReturnsDefaultsWhenNoEnv(t *testing.T) {
	clearEnv()
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppName != "pos-api" {
		t.Errorf("AppName: got %q, want %q", cfg.AppName, "pos-api")
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr: got %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.HTTPReadTimeout != 10_000_000_000 {
		t.Errorf("HTTPReadTimeout: got %v, want 10s", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != 15_000_000_000 {
		t.Errorf("HTTPWriteTimeout: got %v, want 15s", cfg.HTTPWriteTimeout)
	}
	if cfg.BcryptCost != 12 {
		t.Errorf("BcryptCost: got %d, want 12", cfg.BcryptCost)
	}
	if cfg.HTTPMaxBodyBytes != 1<<20 {
		t.Errorf("HTTPMaxBodyBytes: got %d, want %d", cfg.HTTPMaxBodyBytes, 1<<20)
	}
	if cfg.DBMaxConns != 20 {
		t.Errorf("DBMaxConns: got %d, want 20", cfg.DBMaxConns)
	}
	if cfg.DBMinConns != 2 {
		t.Errorf("DBMinConns: got %d, want 2", cfg.DBMinConns)
	}
	if cfg.RedisAddr != "127.0.0.1:6379" {
		t.Errorf("RedisAddr: got %q, want %q", cfg.RedisAddr, "127.0.0.1:6379")
	}
	if cfg.JWTIssuer != "pos-api" {
		t.Errorf("JWTIssuer: got %q, want %q", cfg.JWTIssuer, "pos-api")
	}
	if !cfg.MetricsEnabled {
		t.Errorf("MetricsEnabled: got false, want true (default)")
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "*" {
		t.Errorf("CORSAllowedOrigins: got %v, want [*]", cfg.CORSAllowedOrigins)
	}
}

func TestLoadParsesCORSAllowedOrigins(t *testing.T) {
	clearEnv()
	t.Setenv("CORS_ALLOWED_ORIGINS", "  https://app.example.com , https://admin.example.com  ")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://app.example.com", "https://admin.example.com"}
	if len(cfg.CORSAllowedOrigins) != len(want) {
		t.Fatalf("CORSAllowedOrigins: got %v, want %v", cfg.CORSAllowedOrigins, want)
	}
	for i := range want {
		if cfg.CORSAllowedOrigins[i] != want[i] {
			t.Fatalf("CORSAllowedOrigins[%d]: got %q, want %q", i, cfg.CORSAllowedOrigins[i], want[i])
		}
	}
}

func TestLoadParsesMetricsEnabledFalse(t *testing.T) {
	clearEnv()
	t.Setenv("METRICS_ENABLED", "false")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MetricsEnabled {
		t.Fatalf("MetricsEnabled: got true, want false")
	}
}

func TestLoadRejectsInvalidMetricsEnabled(t *testing.T) {
	clearEnv()
	t.Setenv("METRICS_ENABLED", "sometimes")
	if _, err := Load(); err == nil {
		t.Fatalf("Load: expected error for METRICS_ENABLED=sometimes")
	}
}

func TestLoadKeepsWildcardDefaultWhenCORSBlank(t *testing.T) {
	clearEnv()
	t.Setenv("CORS_ALLOWED_ORIGINS", "   ")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "*" {
		t.Fatalf("CORSAllowedOrigins: got %v, want [*]", cfg.CORSAllowedOrigins)
	}
}

func TestLoadDefaultsPriceCron(t *testing.T) {
	clearEnv()
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PriceCronEnabled {
		t.Fatal("price cron should default to disabled")
	}
	if cfg.PriceCronInterval != 6*time.Hour {
		t.Fatalf("expected default 6h interval, got %s", cfg.PriceCronInterval)
	}
	if cfg.PriceCronVariationPct != 3 {
		t.Fatalf("expected default 3 pct variation, got %d", cfg.PriceCronVariationPct)
	}
}

func TestLoadParsesPriceCron(t *testing.T) {
	clearEnv()
	t.Setenv("PRICE_CRON_ENABLED", "true")
	t.Setenv("PRICE_CRON_INTERVAL", "30m")
	t.Setenv("PRICE_CRON_VARIATION_PCT", "7")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.PriceCronEnabled || cfg.PriceCronInterval != 30*time.Minute || cfg.PriceCronVariationPct != 7 {
		t.Fatalf("price cron parse mismatch: %+v", cfg)
	}
}

func TestLoadRejectsInvalidPriceCronInterval(t *testing.T) {
	clearEnv()
	t.Setenv("PRICE_CRON_INTERVAL", "0s")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for zero interval")
	}
}

func TestLoadRejectsInvalidPriceCronEnabled(t *testing.T) {
	clearEnv()
	t.Setenv("PRICE_CRON_ENABLED", "maybe")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for PRICE_CRON_ENABLED=maybe")
	}
}

func TestLoadRejectsPriceCronVariationOutOfRange(t *testing.T) {
	clearEnv()
	t.Setenv("PRICE_CRON_VARIATION_PCT", "99")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for variation out of range")
	}
}

func TestLoadRejectsInvalidBcryptCost(t *testing.T) {
	clearEnv()
	t.Setenv("BCRYPT_COST", "2")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for BCRYPT_COST=2")
	}
}

func TestLoadRejectsBcryptCostTooHigh(t *testing.T) {
	clearEnv()
	t.Setenv("BCRYPT_COST", "32")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for BCRYPT_COST=32")
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	clearEnv()
	t.Setenv("HTTP_READ_TIMEOUT", "not-a-duration")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestLoadRejectsMinConnsGreaterThanMaxConns(t *testing.T) {
	clearEnv()
	t.Setenv("DB_MAX_CONNS", "2")
	t.Setenv("DB_MIN_CONNS", "5")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when min > max")
	}
}

func TestLoadRejectsRefreshTTLEqualToAccessTTL(t *testing.T) {
	clearEnv()
	t.Setenv("JWT_ACCESS_TTL", "15m")
	t.Setenv("JWT_REFRESH_TTL", "15m")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when refresh TTL <= access TTL")
	}
}

func TestLoadRejectsRefreshTTLBeforeAccessTTL(t *testing.T) {
	clearEnv()
	t.Setenv("JWT_ACCESS_TTL", "1h")
	t.Setenv("JWT_REFRESH_TTL", "5m")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when refresh TTL < access TTL")
	}
}

func TestLoadRejectsNegativeMaxBodyBytes(t *testing.T) {
	clearEnv()
	t.Setenv("HTTP_MAX_BODY_BYTES", "-1")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for negative max body bytes")
	}
}

func TestLoadRejectsInvalidInt(t *testing.T) {
	clearEnv()
	t.Setenv("DB_MAX_CONNS", "not-a-number")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-integer DB_MAX_CONNS")
	}
}

func TestLoadRejectsInvalidInt64(t *testing.T) {
	clearEnv()
	t.Setenv("HTTP_MAX_BODY_BYTES", "not-a-number")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-int64 HTTP_MAX_BODY_BYTES")
	}
}

func TestLoadRejectsZeroDBMaxConns(t *testing.T) {
	clearEnv()
	t.Setenv("DB_MAX_CONNS", "0")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for DB_MAX_CONNS=0")
	}
}

func TestLoadRejectsNegativeDBMinConns(t *testing.T) {
	clearEnv()
	t.Setenv("DB_MIN_CONNS", "-1")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for negative DB_MIN_CONNS")
	}
}

func TestLoadDefaultsCashierDiscountPct(t *testing.T) {
	clearEnv()
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CashierDiscountPct != 5 {
		t.Errorf("CashierDiscountPct: got %d, want 5", cfg.CashierDiscountPct)
	}
}

func TestLoadRejectsCashierDiscountPctOutOfRange(t *testing.T) {
	for _, value := range []string{"101", "-1"} {
		clearEnv()
		t.Setenv("CASHIER_DISCOUNT_PCT", value)
		if _, err := Load(); err == nil {
			t.Fatalf("expected error for CASHIER_DISCOUNT_PCT=%s", value)
		}
	}
}

func TestLoadRejectsInvalidCashierDiscountPct(t *testing.T) {
	clearEnv()
	t.Setenv("CASHIER_DISCOUNT_PCT", "abc")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for CASHIER_DISCOUNT_PCT=abc")
	}
}

func TestLoadOverridesFromEnv(t *testing.T) {
	clearEnv()
	t.Setenv("APP_NAME", "my-pos")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("BCRYPT_COST", "8")
	t.Setenv("CASHIER_DISCOUNT_PCT", "7")
	t.Setenv("DB_MAX_CONNS", "10")
	t.Setenv("DB_MIN_CONNS", "1")
	t.Setenv("MAX_SESSIONS_PER_USER", "2")
	t.Setenv("LOGIN_RATE_MAX", "9")
	t.Setenv("LOGIN_RATE_WINDOW", "90s")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AppName != "my-pos" {
		t.Errorf("AppName: got %q, want %q", cfg.AppName, "my-pos")
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr: got %q, want %q", cfg.HTTPAddr, ":9090")
	}
	if cfg.BcryptCost != 8 {
		t.Errorf("BcryptCost: got %d, want 8", cfg.BcryptCost)
	}
	if cfg.CashierDiscountPct != 7 {
		t.Errorf("CashierDiscountPct: got %d, want 7", cfg.CashierDiscountPct)
	}
	if cfg.DBMaxConns != 10 {
		t.Errorf("DBMaxConns: got %d, want 10", cfg.DBMaxConns)
	}
	if cfg.DBMinConns != 1 {
		t.Errorf("DBMinConns: got %d, want 1", cfg.DBMinConns)
	}
	if cfg.MaxSessionsPerUser != 2 {
		t.Errorf("MaxSessionsPerUser: got %d, want 2", cfg.MaxSessionsPerUser)
	}
	if cfg.LoginRateMax != 9 {
		t.Errorf("LoginRateMax: got %d, want 9", cfg.LoginRateMax)
	}
	if cfg.LoginRateWindow != 90*time.Second {
		t.Errorf("LoginRateWindow: got %v, want 90s", cfg.LoginRateWindow)
	}
}

func TestLoadRejectsZeroMaxSessions(t *testing.T) {
	clearEnv()
	t.Setenv("MAX_SESSIONS_PER_USER", "0")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for MAX_SESSIONS_PER_USER=0")
	}
}

func TestLoadRejectsInvalidMaxSessions(t *testing.T) {
	clearEnv()
	t.Setenv("MAX_SESSIONS_PER_USER", "abc")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid MAX_SESSIONS_PER_USER")
	}
}

func TestLoadRejectsZeroLoginRateMax(t *testing.T) {
	clearEnv()
	t.Setenv("LOGIN_RATE_MAX", "0")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for LOGIN_RATE_MAX=0")
	}
}

func TestLoadApiRateLimitDefaults(t *testing.T) {
	clearEnv()
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ApiRateLimitEnabled {
		t.Error("ApiRateLimitEnabled: expected false by default")
	}
	if cfg.ApiRateLimitMax != 300 {
		t.Errorf("ApiRateLimitMax: got %d, want 300", cfg.ApiRateLimitMax)
	}
	if cfg.ApiRateLimitWindow != time.Minute {
		t.Errorf("ApiRateLimitWindow: got %v, want 1m", cfg.ApiRateLimitWindow)
	}
}

func TestLoadCustomApiRateLimit(t *testing.T) {
	clearEnv()
	t.Setenv("RATE_LIMIT_ENABLED", "true")
	t.Setenv("RATE_LIMIT_MAX", "100")
	t.Setenv("RATE_LIMIT_WINDOW", "30s")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ApiRateLimitEnabled {
		t.Error("ApiRateLimitEnabled: expected true")
	}
	if cfg.ApiRateLimitMax != 100 {
		t.Errorf("ApiRateLimitMax: got %d, want 100", cfg.ApiRateLimitMax)
	}
	if cfg.ApiRateLimitWindow != 30*time.Second {
		t.Errorf("ApiRateLimitWindow: got %v, want 30s", cfg.ApiRateLimitWindow)
	}
}

func TestLoadRejectsZeroApiRateLimitMax(t *testing.T) {
	clearEnv()
	t.Setenv("RATE_LIMIT_MAX", "0")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for RATE_LIMIT_MAX=0")
	}
}

func TestLoadRejectsInvalidApiRateLimitBoolean(t *testing.T) {
	clearEnv()
	t.Setenv("RATE_LIMIT_ENABLED", "maybe")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid RATE_LIMIT_ENABLED")
	}
}

func TestLoadCustomDurations(t *testing.T) {
	clearEnv()
	t.Setenv("HTTP_READ_TIMEOUT", "30s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "45s")
	t.Setenv("JWT_ACCESS_TTL", "30m")
	t.Setenv("JWT_REFRESH_TTL", "336h")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPReadTimeout != 30_000_000_000 {
		t.Errorf("HTTPReadTimeout: got %v, want 30s", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != 45_000_000_000 {
		t.Errorf("HTTPWriteTimeout: got %v, want 45s", cfg.HTTPWriteTimeout)
	}
}

func clearEnv() {
	keys := []string{
		"APP_NAME", "HTTP_ADDR", "DATABASE_URL", "REDIS_ADDR", "REDIS_PASSWORD",
		"JWT_ISSUER", "JWT_ACCESS_SECRET", "JWT_REFRESH_SECRET",
		"LOG_LEVEL", "LOG_FORMAT", "HTTP_READ_TIMEOUT", "HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT", "HTTP_SHUTDOWN_TIMEOUT", "HTTP_MAX_BODY_BYTES",
		"DB_MAX_CONNS", "DB_MIN_CONNS", "DB_MAX_CONN_LIFETIME", "DB_MAX_CONN_IDLE_TIME",
		"REDIS_DB", "BCRYPT_COST", "JWT_ACCESS_TTL", "JWT_REFRESH_TTL",
		"RATE_LIMIT_ENABLED", "RATE_LIMIT_MAX", "RATE_LIMIT_WINDOW",
	}
	for _, k := range keys {
		_ = os.Unsetenv(k)
	}
}
