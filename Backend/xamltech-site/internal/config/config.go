// Package config loads runtime configuration for the XamlTech website service.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime settings. All values have safe defaults so the
// service boots with no environment at all.
type Config struct {
	// Addr is the TCP listen address, e.g. ":8090" or "127.0.0.1:8090".
	Addr string
	// BaseURL is the canonical public origin (scheme + host), no trailing slash.
	BaseURL string
	// Env is "production" or "development"; controls cookie security + caching.
	Env string
	// DefLang is the default language code ("ar" or "en").
	DefLang string
	// ReadTimeout / WriteTimeout bound each HTTP request.
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	// ContactTo is the mailbox that demo/contact form submissions are sent to.
	ContactTo string
	// SubmissionsFile, when set, appends form submissions as JSON lines.
	SubmissionsFile string
	// ClarityID is the Microsoft Clarity project id; empty disables the tag.
	ClarityID string
}

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	c := Config{
		Addr:            env("XT_ADDR", "127.0.0.1:8090"),
		BaseURL:         strings.TrimRight(env("XT_BASE_URL", "https://xamltech.com"), "/"),
		Env:             env("XT_ENV", "development"),
		DefLang:         env("XT_DEFAULT_LANG", "ar"),
		ReadTimeout:     durationEnv("XT_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    durationEnv("XT_WRITE_TIMEOUT", 30*time.Second),
		ContactTo:       env("XT_CONTACT_TO", "info@xamltech.com"),
		SubmissionsFile: env("XT_SUBMISSIONS_FILE", ""),
		ClarityID:       env("XT_CLARITY_ID", "y1l8d34cyq"),
	}
	if c.DefLang != "ar" && c.DefLang != "en" {
		c.DefLang = "ar"
	}
	return c
}

// IsProduction reports whether the service runs in production mode.
func (c Config) IsProduction() bool { return c.Env == "production" }

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func durationEnv(key string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// BoolEnv is a tiny helper used by tests.
func BoolEnv(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
