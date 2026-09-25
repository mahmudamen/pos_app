package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Surface identifies which public domain a request arrived on. The three
// surfaces are deliberately disjoint: the company site carries no API, the
// SaaS host carries the control plane only, and the POS host carries the
// storefront/cashier API only.
type Surface string

const (
	// SurfaceUnknown is both "routing disabled" and "host is not one of the
	// three public domains" — in both cases no surface gating is applied.
	SurfaceUnknown Surface = ""
	SurfaceCompany Surface = "company"
	SurfaceSaaS    Surface = "saas"
	SurfacePOS     Surface = "pos"
)

// Default public domains. They are overridable so a staging deployment (or a
// future per-region split) does not have to fork the config loader.
const (
	DefaultCompanyDomain = "xamltech.com"
	DefaultSaaSDomain    = "api.xamltech.com"
	DefaultPOSDomain     = "posgo.xamltech.com"
)

// SurfaceRouting maps request hosts onto the three public surfaces. When
// Enabled is false the whole feature is inert: every host and every route keeps
// working exactly as before, which is what local development, the OpenAPI
// generator and the existing test-suite rely on.
type SurfaceRouting struct {
	Enabled       bool
	CompanyDomain string
	SaaSDomain    string
	POSDomain     string
}

// internalHosts are never routed: the Docker Compose service names, loopback
// and the host aliases used by local development and the /metrics collector.
// None of them is publicly resolvable, so bypassing the gate for them cannot be
// reached from the internet.
var internalHosts = map[string]struct{}{
	"":                     {},
	"localhost":            {},
	"127.0.0.1":            {},
	"::1":                  {},
	"0.0.0.0":              {},
	"host.docker.internal": {},
	"api":                  {},
	"caddy":                {},
}

// IsInternalHost reports whether a request host is a local/container name that
// must bypass surface routing.
func IsInternalHost(host string) bool {
	_, ok := internalHosts[NormalizeHost(host)]
	return ok
}

// NormalizeHost lowercases a Host header and strips the port, any scheme and a
// trailing root dot, so "API.XAMLtech.com:443" and "api.xamltech.com" classify
// identically.
func NormalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}
	if idx := strings.Index(host, "://"); idx >= 0 {
		if parsed, err := url.Parse(host); err == nil && parsed.Host != "" {
			host = parsed.Host
		} else {
			host = host[idx+3:]
		}
		host = strings.ToLower(strings.TrimSpace(host))
	}
	if bare, _, err := net.SplitHostPort(host); err == nil {
		host = bare
	}
	host = strings.TrimSuffix(host, ".")
	return strings.Trim(host, "[]")
}

// Resolve maps a request host onto its surface. It returns SurfaceUnknown for
// any host that is not one of the three configured domains (and for every host
// while routing is disabled).
func (r SurfaceRouting) Resolve(host string) Surface {
	if !r.Enabled {
		return SurfaceUnknown
	}
	switch NormalizeHost(host) {
	case NormalizeHost(r.CompanyDomain):
		return SurfaceCompany
	case NormalizeHost(r.SaaSDomain):
		return SurfaceSaaS
	case NormalizeHost(r.POSDomain):
		return SurfacePOS
	default:
		return SurfaceUnknown
	}
}

func loadSurfaceRouting() (SurfaceRouting, error) {
	r := SurfaceRouting{
		CompanyDomain: envOr("COMPANY_DOMAIN", DefaultCompanyDomain),
		SaaSDomain:    envOr("SAAS_DOMAIN", DefaultSaaSDomain),
		POSDomain:     envOr("POS_DOMAIN", DefaultPOSDomain),
	}
	enabled, err := boolValue("SURFACE_ROUTING_ENABLED", false)
	if err != nil {
		return SurfaceRouting{}, err
	}
	r.Enabled = enabled
	if !enabled {
		return r, nil
	}
	domains := [][2]string{
		{"COMPANY_DOMAIN", r.CompanyDomain},
		{"SAAS_DOMAIN", r.SaaSDomain},
		{"POS_DOMAIN", r.POSDomain},
	}
	seen := make(map[string]string, len(domains))
	for _, d := range domains {
		key := NormalizeHost(d[1])
		if key == "" {
			return SurfaceRouting{}, fmt.Errorf("%s must be a domain when SURFACE_ROUTING_ENABLED is true", d[0])
		}
		if other, dup := seen[key]; dup {
			return SurfaceRouting{}, fmt.Errorf("%s and %s must be different domains (both %q)", other, d[0], key)
		}
		seen[key] = d[0]
	}
	return r, nil
}
