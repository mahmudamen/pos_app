package http

import (
	"net/http"
	"strings"

	"github.com/example/pos-api/internal/config"
	"github.com/gin-gonic/gin"
)

const surfaceKey = "public_surface"

// sharedRoutes answer on every surface: session bootstrap (login/refresh/
// logout) must work from whichever domain the client is configured for, and the
// public reference data (countries/currencies) is needed before a session even
// exists.
var sharedRoutes = []string{
	"/v1/auth/login",
	"/v1/auth/refresh",
	"/v1/auth/logout",
}

// googleVerifyPath is the Search Console HTML-file verification token, published
// on all three surfaces (see publicRoutes). It lives here as a literal because
// the gate is the http package and the served bytes are the server package's
// concern; the two are kept in step by TestGoogleVerificationPublishedOnEverySurface.
const googleVerifyPath = "/google79d5199d984f63af.html"

// RequiredSurface is the only surface a path may be served on, or
// config.SurfaceUnknown when the path is shared (or not a gated API path).
func RequiredSurface(path string) config.Surface {
	switch {
	case under(path, "/v1/saas"), under(path, "/v1/platform"):
		return config.SurfaceSaaS
	case under(path, "/v1/meta"):
		return config.SurfaceUnknown
	}
	for _, shared := range sharedRoutes {
		if under(path, shared) {
			return config.SurfaceUnknown
		}
	}
	if strings.HasPrefix(path, "/v1/") {
		return config.SurfacePOS
	}
	return config.SurfaceUnknown
}

// under matches a path prefix on a segment boundary so "/v1/saasx" is not
// mistaken for the "/v1/saas" group.
func under(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

// publicRoutes are the non-API pages each surface publishes. The company domain
// is a brochure plus legal pages, the POS domain adds the self-order and
// install pages, and the console domain publishes nothing but its redirect to
// the admin single-page app.
//
// The Search Console verification file is on every surface: it is an inert
// 53-byte ownership token, and which domain the property is verified against is
// chosen in the Search Console UI, so gating it per host would only make
// verification fail on the domain that happens to be submitted.
var publicRoutes = map[config.Surface][]string{
	config.SurfaceCompany: {"/", "/pricing", "/private", "/screenshots", "/delete-account", googleVerifyPath},
	config.SurfacePOS:     {"/", "/pricing", "/private", "/screenshots", "/selforder", "/sw.js", "/apk", "/delete-account", googleVerifyPath},
	config.SurfaceSaaS:    {"/", googleVerifyPath},
}

// ServesAPI reports whether an API path is published on the surface. The company
// domain exposes no /v1 route at all, not even the session bootstrap, so a
// brand page can never be used to obtain a token.
func ServesAPI(surface config.Surface, path string) bool {
	if surface == config.SurfaceCompany {
		return false
	}
	want := RequiredSurface(path)
	return want == config.SurfaceUnknown || want == surface
}

// ServesPage reports whether a non-API path is published on the surface.
// Probes and the scrape endpoint answer everywhere so the platform health
// checks keep working regardless of which domain they were pointed at.
func ServesPage(surface config.Surface, path string) bool {
	if under(path, "/health") || path == "/metrics" {
		return true
	}
	for _, prefix := range publicRoutes[surface] {
		if under(path, prefix) {
			return true
		}
	}
	return false
}

// SurfaceRouting resolves the request Host onto one of the three public
// surfaces and refuses paths that belong to another one. It is a no-op while
// surface routing is disabled, so a single-domain deployment (and the whole
// offline test-suite) behaves exactly as before.
func SurfaceRouting(cfg config.SurfaceRouting) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		surface := cfg.Resolve(c.Request.Host)
		if surface == config.SurfaceUnknown {
			// Loopback and the Compose service names are not public hosts, and
			// the container-to-container hop must keep working.
			if config.IsInternalHost(c.Request.Host) {
				c.Next()
				return
			}
			abortError(c, http.StatusNotFound, "unknown_host", "unknown host")
			return
		}
		c.Set(surfaceKey, surface)
		path := c.Request.URL.Path
		if path == "/v1" || under(path, "/v1") {
			if !ServesAPI(surface, path) {
				abortError(c, http.StatusNotFound, "not_available_on_host", "this endpoint is not served on this domain")
				return
			}
		} else if !ServesPage(surface, path) {
			abortError(c, http.StatusNotFound, "not_available_on_host", "this page is not served on this domain")
			return
		}
		c.Next()
	}
}

// Surface returns the surface the request arrived on. config.SurfaceUnknown
// means surface routing is disabled (or the host is internal), in which case
// every surface-specific rule is skipped.
func Surface(c *gin.Context) config.Surface {
	if value, ok := c.Get(surfaceKey); ok {
		if surface, ok := value.(config.Surface); ok {
			return surface
		}
	}
	return config.SurfaceUnknown
}

// AllowsRole reports whether a user role may hold a session on a surface: the
// console host is platform operators only, the POS host is store staff only,
// and a disabled/internal surface accepts everyone.
func AllowsRole(surface config.Surface, role string) bool {
	switch surface {
	case config.SurfaceSaaS:
		return role == saasAdminRole
	case config.SurfacePOS:
		return role != "" && role != saasAdminRole
	default:
		return true
	}
}

// AbortRoleNotOnSurface writes the 403 a cross-surface sign-in gets. The code is
// identical on both hosts so clients can branch on one value; the message names
// the domain the account belongs on.
func AbortRoleNotOnSurface(c *gin.Context) {
	message := "this account cannot sign in on this domain"
	switch Surface(c) {
	case config.SurfaceSaaS:
		message = "sign in with a platform administrator account"
	case config.SurfacePOS:
		message = "platform administrators sign in on the console domain"
	}
	abortError(c, http.StatusForbidden, "wrong_surface", message)
}
