package config

import (
	"fmt"
	"net"
	"strings"
)

// Client IP trust for the per-IP rate limiters.
//
// Login, /v1/saas, registration and community-join all key off gin's
// ClientIP(), which walks X-Forwarded-For right to left and returns the first
// hop it does not consider trusted. With gin's default trusted list — every
// address — that walk runs off the end of the header and returns the leftmost
// entry, which the caller wrote. Anyone could send "X-Forwarded-For:
// 203.0.113.9" and get a fresh rate-limit bucket per request.
//
// So only the hops that can append to the header on our behalf are trusted:
// the reverse proxy in front of us, plus Cloudflare when a hostname is
// orange-clouded. Everything to the left of the first untrusted hop is
// attacker-controlled and is never returned.
var (
	// Loopback plus the private ranges the API is reachable from: the Caddy
	// container on the compose network, and anything curling 127.0.0.1 directly.
	edgeProxyCIDRs = []string{
		"127.0.0.1/32",
		"::1/128",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	// Cloudflare's published edge ranges. Only needed when a hostname is
	// orange-clouded: the header then reads "<client>, <edge>" and the edge hop
	// has to be recognised as trusted for the walk to reach the real client.
	// Source: https://www.cloudflare.com/ips-v4/ and .../ips-v6/. Override with
	// TRUSTED_PROXIES if Cloudflare changes them ahead of a release.
	cloudflareCIDRs = []string{
		"103.21.244.0/22",
		"103.22.200.0/22",
		"103.31.4.0/22",
		"104.16.0.0/13",
		"104.24.0.0/14",
		"108.162.192.0/18",
		"131.0.72.0/22",
		"141.101.64.0/18",
		"162.158.0.0/15",
		"172.64.0.0/13",
		"173.245.48.0/20",
		"188.114.96.0/20",
		"190.93.240.0/20",
		"197.234.240.0/22",
		"198.41.128.0/17",
		"2400:cb00::/32",
		"2405:8100::/32",
		"2405:b500::/32",
		"2606:4700::/32",
		"2803:f800::/32",
		"2a06:98c0::/29",
		"2c0f:f248::/32",
	}
)

// DefaultTrustedProxies is the edge plus Cloudflare.
func DefaultTrustedProxies() []string {
	out := make([]string, 0, len(edgeProxyCIDRs)+len(cloudflareCIDRs))
	out = append(out, edgeProxyCIDRs...)
	return append(out, cloudflareCIDRs...)
}

// ParseTrustedProxies turns a comma or space separated list of CIDRs into a
// validated one, rejecting the catch-all prefixes. An empty string means "use
// the defaults".
func ParseTrustedProxies(raw string) ([]string, error) {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(fields) == 0 {
		return DefaultTrustedProxies(), nil
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		_, network, err := net.ParseCIDR(field)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXIES: %q is not a CIDR", field)
		}
		ones, bits := network.Mask.Size()
		if ones == 0 && (bits == 32 || bits == 128) {
			return nil, fmt.Errorf(
				"TRUSTED_PROXIES: %q trusts every address, which makes X-Forwarded-For client-controlled",
				field,
			)
		}
		out = append(out, network.String())
	}
	return out, nil
}
