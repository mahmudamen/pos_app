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
	AppName               string
	HTTPAddr              string
	HTTPReadTimeout       time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration
	HTTPShutdownTimeout   time.Duration
	HTTPMaxBodyBytes      int64
	DatabaseURL           string
	DBMaxConns            int32
	DBMinConns            int32
	DBMaxConnLifetime     time.Duration
	DBMaxConnIdleTime     time.Duration
	RedisAddr             string
	RedisPassword         string
	RedisDB               int
	JWTIssuer             string
	JWTAccessSecret       string
	JWTRefreshSecret      string
	JWTAccessTTL          time.Duration
	JWTRefreshTTL         time.Duration
	BcryptCost            int
	CashierDiscountPct    int
	CORSAllowedOrigins    []string
	LogLevel              string
	LogFormat             string
	MaxSessionsPerUser    int
	LoginRateMax          int
	LoginRateWindow       time.Duration
	ApiRateLimitEnabled   bool
	ApiRateLimitMax       int
	ApiRateLimitWindow    time.Duration
	MetricsEnabled        bool
	PriceCronEnabled      bool
	PriceCronInterval     time.Duration
	PriceCronVariationPct int
	Trial                 TrialConfig
}

// TrialConfig is the configurable free-trial and identity policy. Every field
// is backend-owned: the client never supplies trial duration, scope, or dates.
type TrialConfig struct {
	DurationDays                 int
	Scope                        string
	RequireEmailVerification     bool
	RequirePhoneVerification     bool
	RequireDeviceIntegrity       bool
	MaxOrganizationsPerAccount   int
	MaxActiveInstallations       int
	SuspiciousRegistrationPolicy string
	OfflinePolicy                string
	RegisterRatePerIPPerHour     int
	EmailTokenTTL                time.Duration
	OTPTTL                       time.Duration
	PromoTrialsEnabled           bool
	Mailer                       string
	SMSSender                    string
}

// ValidTrialScopes lists the supported trial-identity scopes.
var ValidTrialScopes = []string{"account", "organization", "verified_phone", "business_identity"}

// ValidSuspiciousPolicies lists how a suspicious registration is treated.
var ValidSuspiciousPolicies = []string{"allow", "review", "deny"}

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

	priceCronEnabled, err := boolValue("PRICE_CRON_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	if c.PriceCronInterval, err = duration("PRICE_CRON_INTERVAL", 6*time.Hour); err != nil {
		return Config{}, err
	}
	if c.PriceCronInterval <= 0 {
		return Config{}, fmt.Errorf("PRICE_CRON_INTERVAL must be positive")
	}
	priceCronVariation, err := intValue("PRICE_CRON_VARIATION_PCT", 3)
	if err != nil {
		return Config{}, err
	}
	if priceCronVariation < 0 || priceCronVariation > 25 {
		return Config{}, fmt.Errorf("PRICE_CRON_VARIATION_PCT must be between 0 and 25")
	}
	c.PriceCronEnabled = priceCronEnabled
	c.PriceCronVariationPct = priceCronVariation
	trial, err := loadTrialConfig()
	if err != nil {
		return Config{}, err
	}
	c.Trial = trial
	return c, nil
}

func loadTrialConfig() (TrialConfig, error) {
	t := TrialConfig{
		Scope:                        envOr("TRIAL_SCOPE", "account"),
		SuspiciousRegistrationPolicy: envOr("SUSPICIOUS_REGISTRATION_POLICY", "review"),
		OfflinePolicy:                envOr("OFFLINE_TRIAL_POLICY", "grace24h"),
		Mailer:                       envOr("MAILER", "noop"),
		SMSSender:                    envOr("SMS_SENDER", "noop"),
	}
	if !oneOf(t.Scope, ValidTrialScopes) {
		return TrialConfig{}, fmt.Errorf("TRIAL_SCOPE must be one of %v, got %q", ValidTrialScopes, t.Scope)
	}
	if !oneOf(t.SuspiciousRegistrationPolicy, ValidSuspiciousPolicies) {
		return TrialConfig{}, fmt.Errorf("SUSPICIOUS_REGISTRATION_POLICY must be one of %v, got %q", ValidSuspiciousPolicies, t.SuspiciousRegistrationPolicy)
	}
	if !oneOf(t.Mailer, []string{"noop", "smtp"}) {
		return TrialConfig{}, fmt.Errorf("MAILER must be noop or smtp, got %q", t.Mailer)
	}
	if !oneOf(t.SMSSender, []string{"noop", "provider"}) {
		return TrialConfig{}, fmt.Errorf("SMS_SENDER must be noop or provider, got %q", t.SMSSender)
	}
	var err error
	if t.DurationDays, err = intValue("TRIAL_DURATION_DAYS", 14); err != nil {
		return TrialConfig{}, err
	}
	if t.DurationDays < 0 || t.DurationDays > 3650 {
		return TrialConfig{}, fmt.Errorf("TRIAL_DURATION_DAYS must be between 0 and 3650")
	}
	if t.MaxOrganizationsPerAccount, err = intValue("MAX_ORGANIZATIONS_PER_ACCOUNT", 3); err != nil {
		return TrialConfig{}, err
	}
	if t.MaxOrganizationsPerAccount < 1 {
		return TrialConfig{}, fmt.Errorf("MAX_ORGANIZATIONS_PER_ACCOUNT must be at least 1")
	}
	if t.MaxActiveInstallations, err = intValue("MAX_ACTIVE_INSTALLATIONS", 10); err != nil {
		return TrialConfig{}, err
	}
	if t.MaxActiveInstallations < 1 {
		return TrialConfig{}, fmt.Errorf("MAX_ACTIVE_INSTALLATIONS must be at least 1")
	}
	if t.RegisterRatePerIPPerHour, err = intValue("REGISTER_RATE_PER_IP_PER_HOUR", 5); err != nil {
		return TrialConfig{}, err
	}
	if t.RegisterRatePerIPPerHour < 1 {
		return TrialConfig{}, fmt.Errorf("REGISTER_RATE_PER_IP_PER_HOUR must be at least 1")
	}
	if t.EmailTokenTTL, err = duration("EMAIL_TOKEN_TTL", 15*time.Minute); err != nil {
		return TrialConfig{}, err
	}
	if t.OTPTTL, err = duration("OTP_TTL", 5*time.Minute); err != nil {
		return TrialConfig{}, err
	}
	if t.EmailTokenTTL <= 0 || t.OTPTTL <= 0 {
		return TrialConfig{}, fmt.Errorf("EMAIL_TOKEN_TTL and OTP_TTL must be positive")
	}
	if t.RequireEmailVerification, err = boolValue("REQUIRE_EMAIL_VERIFICATION", false); err != nil {
		return TrialConfig{}, err
	}
	if t.RequirePhoneVerification, err = boolValue("REQUIRE_PHONE_VERIFICATION", false); err != nil {
		return TrialConfig{}, err
	}
	if t.RequireDeviceIntegrity, err = boolValue("REQUIRE_DEVICE_INTEGRITY", false); err != nil {
		return TrialConfig{}, err
	}
	if t.PromoTrialsEnabled, err = boolValue("PROMO_TRIALS_ENABLED", false); err != nil {
		return TrialConfig{}, err
	}
	return t, nil
}

func oneOf(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
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
