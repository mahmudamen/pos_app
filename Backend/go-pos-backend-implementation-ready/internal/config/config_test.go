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
	os.Setenv("CORS_ALLOWED_ORIGINS", "  https://app.example.com , https://admin.example.com  ")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")
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
	os.Setenv("METRICS_ENABLED", "false")
	defer os.Unsetenv("METRICS_ENABLED")
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
	os.Setenv("METRICS_ENABLED", "sometimes")
	defer os.Unsetenv("METRICS_ENABLED")
	if _, err := Load(); err == nil {
		t.Fatalf("Load: expected error for METRICS_ENABLED=sometimes")
	}
}

func TestLoadKeepsWildcardDefaultWhenCORSBlank(t *testing.T) {
	clearEnv()
	os.Setenv("CORS_ALLOWED_ORIGINS", "   ")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "*" {
		t.Fatalf("CORSAllowedOrigins: got %v, want [*]", cfg.CORSAllowedOrigins)
	}
}

func TestLoadRejectsInvalidBcryptCost(t *testing.T) {
	clearEnv()
	os.Setenv("BCRYPT_COST", "2")
	defer os.Unsetenv("BCRYPT_COST")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for BCRYPT_COST=2")
	}
}

func TestLoadRejectsBcryptCostTooHigh(t *testing.T) {
	clearEnv()
	os.Setenv("BCRYPT_COST", "32")
	defer os.Unsetenv("BCRYPT_COST")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for BCRYPT_COST=32")
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	clearEnv()
	os.Setenv("HTTP_READ_TIMEOUT", "not-a-duration")
	defer os.Unsetenv("HTTP_READ_TIMEOUT")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestLoadRejectsMinConnsGreaterThanMaxConns(t *testing.T) {
	clearEnv()
	os.Setenv("DB_MAX_CONNS", "2")
	os.Setenv("DB_MIN_CONNS", "5")
	defer os.Unsetenv("DB_MAX_CONNS")
	defer os.Unsetenv("DB_MIN_CONNS")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when min > max")
	}
}

func TestLoadRejectsRefreshTTLEqualToAccessTTL(t *testing.T) {
	clearEnv()
	os.Setenv("JWT_ACCESS_TTL", "15m")
	os.Setenv("JWT_REFRESH_TTL", "15m")
	defer os.Unsetenv("JWT_ACCESS_TTL")
	defer os.Unsetenv("JWT_REFRESH_TTL")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when refresh TTL <= access TTL")
	}
}

func TestLoadRejectsRefreshTTLBeforeAccessTTL(t *testing.T) {
	clearEnv()
	os.Setenv("JWT_ACCESS_TTL", "1h")
	os.Setenv("JWT_REFRESH_TTL", "5m")
	defer os.Unsetenv("JWT_ACCESS_TTL")
	defer os.Unsetenv("JWT_REFRESH_TTL")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when refresh TTL < access TTL")
	}
}

func TestLoadRejectsNegativeMaxBodyBytes(t *testing.T) {
	clearEnv()
	os.Setenv("HTTP_MAX_BODY_BYTES", "-1")
	defer os.Unsetenv("HTTP_MAX_BODY_BYTES")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for negative max body bytes")
	}
}

func TestLoadRejectsInvalidInt(t *testing.T) {
	clearEnv()
	os.Setenv("DB_MAX_CONNS", "not-a-number")
	defer os.Unsetenv("DB_MAX_CONNS")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-integer DB_MAX_CONNS")
	}
}

func TestLoadRejectsInvalidInt64(t *testing.T) {
	clearEnv()
	os.Setenv("HTTP_MAX_BODY_BYTES", "not-a-number")
	defer os.Unsetenv("HTTP_MAX_BODY_BYTES")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-int64 HTTP_MAX_BODY_BYTES")
	}
}

func TestLoadRejectsZeroDBMaxConns(t *testing.T) {
	clearEnv()
	os.Setenv("DB_MAX_CONNS", "0")
	defer os.Unsetenv("DB_MAX_CONNS")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for DB_MAX_CONNS=0")
	}
}

func TestLoadRejectsNegativeDBMinConns(t *testing.T) {
	clearEnv()
	os.Setenv("DB_MIN_CONNS", "-1")
	defer os.Unsetenv("DB_MIN_CONNS")
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
		os.Setenv("CASHIER_DISCOUNT_PCT", value)
		if _, err := Load(); err == nil {
			t.Fatalf("expected error for CASHIER_DISCOUNT_PCT=%s", value)
		}
		os.Unsetenv("CASHIER_DISCOUNT_PCT")
	}
}

func TestLoadRejectsInvalidCashierDiscountPct(t *testing.T) {
	clearEnv()
	os.Setenv("CASHIER_DISCOUNT_PCT", "abc")
	defer os.Unsetenv("CASHIER_DISCOUNT_PCT")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for CASHIER_DISCOUNT_PCT=abc")
	}
}

func TestLoadOverridesFromEnv(t *testing.T) {
	clearEnv()
	os.Setenv("APP_NAME", "my-pos")
	os.Setenv("HTTP_ADDR", ":9090")
	os.Setenv("BCRYPT_COST", "8")
	os.Setenv("CASHIER_DISCOUNT_PCT", "7")
	os.Setenv("DB_MAX_CONNS", "10")
	os.Setenv("DB_MIN_CONNS", "1")
	os.Setenv("MAX_SESSIONS_PER_USER", "2")
	os.Setenv("LOGIN_RATE_MAX", "9")
	os.Setenv("LOGIN_RATE_WINDOW", "90s")
	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("HTTP_ADDR")
		os.Unsetenv("BCRYPT_COST")
		os.Unsetenv("CASHIER_DISCOUNT_PCT")
		os.Unsetenv("DB_MAX_CONNS")
		os.Unsetenv("DB_MIN_CONNS")
		os.Unsetenv("MAX_SESSIONS_PER_USER")
		os.Unsetenv("LOGIN_RATE_MAX")
		os.Unsetenv("LOGIN_RATE_WINDOW")
	}()
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
	os.Setenv("MAX_SESSIONS_PER_USER", "0")
	defer os.Unsetenv("MAX_SESSIONS_PER_USER")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for MAX_SESSIONS_PER_USER=0")
	}
}

func TestLoadRejectsInvalidMaxSessions(t *testing.T) {
	clearEnv()
	os.Setenv("MAX_SESSIONS_PER_USER", "abc")
	defer os.Unsetenv("MAX_SESSIONS_PER_USER")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid MAX_SESSIONS_PER_USER")
	}
}

func TestLoadRejectsZeroLoginRateMax(t *testing.T) {
	clearEnv()
	os.Setenv("LOGIN_RATE_MAX", "0")
	defer os.Unsetenv("LOGIN_RATE_MAX")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for LOGIN_RATE_MAX=0")
	}
}

func TestLoadCustomDurations(t *testing.T) {
	clearEnv()
	os.Setenv("HTTP_READ_TIMEOUT", "30s")
	os.Setenv("HTTP_WRITE_TIMEOUT", "45s")
	os.Setenv("JWT_ACCESS_TTL", "30m")
	os.Setenv("JWT_REFRESH_TTL", "336h")
	defer func() {
		os.Unsetenv("HTTP_READ_TIMEOUT")
		os.Unsetenv("HTTP_WRITE_TIMEOUT")
		os.Unsetenv("JWT_ACCESS_TTL")
		os.Unsetenv("JWT_REFRESH_TTL")
	}()
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
	}
	for _, k := range keys {
		os.Unsetenv(k)
	}
}
