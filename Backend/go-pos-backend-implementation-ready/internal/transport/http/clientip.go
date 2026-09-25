package http

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
)

// ConfigureTrustedProxies installs the validated CIDR list on the engine and
// returns what was applied. It must run before the first request; every
// consumer of gin.ClientIP() in this codebase is correct once it has.
func ConfigureTrustedProxies(engine *gin.Engine, trustedProxies []string) error {
	return engine.SetTrustedProxies(trustedProxies)
}

// LogTrustedProxies records the applied list, so an operator can read from the
// startup log which hops are allowed to set a client address.
func LogTrustedProxies(logger *slog.Logger, trustedProxies []string, err error) {
	if err != nil {
		logger.Warn("could not apply the trusted proxy list; falling back to the default edge list",
			"error", err)
		return
	}
	logger.Info("client IP trust configured", "trusted_proxies", len(trustedProxies),
		"cidrs", strings.Join(trustedProxies, ","))
}
