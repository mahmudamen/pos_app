package config

import "testing"

func TestNormalizeHostStripsSchemePortAndCase(t *testing.T) {
	cases := map[string]string{
		"api.xamltech.com":            "api.xamltech.com",
		"API.XAMLtech.com":            "api.xamltech.com",
		"api.xamltech.com:443":        "api.xamltech.com",
		"api.xamltech.com:8080":       "api.xamltech.com",
		"xamltech.com.":               "xamltech.com",
		"https://api.xamltech.com":    "api.xamltech.com",
		"https://posgo.xamltech.com/": "posgo.xamltech.com",
		"  PosGo.XamlTech.com  ":      "posgo.xamltech.com",
		"[::1]:8080":                  "::1",
		"::1":                         "::1",
		"":                            "",
	}
	for host, want := range cases {
		if got := NormalizeHost(host); got != want {
			t.Errorf("NormalizeHost(%q) = %q, want %q", host, got, want)
		}
	}
}

func TestSurfaceRoutingDisabledResolvesNothing(t *testing.T) {
	r := SurfaceRouting{CompanyDomain: DefaultCompanyDomain, SaaSDomain: DefaultSaaSDomain, POSDomain: DefaultPOSDomain}
	for _, host := range []string{DefaultCompanyDomain, DefaultSaaSDomain, DefaultPOSDomain} {
		if got := r.Resolve(host); got != SurfaceUnknown {
			t.Errorf("disabled routing: Resolve(%q) = %q, want SurfaceUnknown", host, got)
		}
	}
}

func TestSurfaceRoutingResolvesTheThreeDomains(t *testing.T) {
	r := SurfaceRouting{Enabled: true, CompanyDomain: DefaultCompanyDomain, SaaSDomain: DefaultSaaSDomain, POSDomain: DefaultPOSDomain}
	cases := map[string]Surface{
		"xamltech.com":             SurfaceCompany,
		"www.xamltech.com":         SurfaceUnknown,
		"api.xamltech.com":         SurfaceSaaS,
		"API.XAMLTECH.com:443":     SurfaceSaaS,
		"posgo.xamltech.com":       SurfacePOS,
		"posgo.xamltech.com:8443":  SurfacePOS,
		"store.xamltech.com":       SurfaceUnknown,
		"example.com":              SurfaceUnknown,
		"evil-posgo.xamltech.com":  SurfaceUnknown,
		"xamltech.com.attacker.io": SurfaceUnknown,
	}
	for host, want := range cases {
		if got := r.Resolve(host); got != want {
			t.Errorf("Resolve(%q) = %q, want %q", host, got, want)
		}
	}
}

func TestSurfaceRoutingHonorsCustomDomains(t *testing.T) {
	r := SurfaceRouting{Enabled: true, CompanyDomain: "studio.example", SaaSDomain: "console.example", POSDomain: "shop.example"}
	if got := r.Resolve("shop.example"); got != SurfacePOS {
		t.Errorf("Resolve(shop.example) = %q, want pos", got)
	}
	if got := r.Resolve(DefaultPOSDomain); got != SurfaceUnknown {
		t.Errorf("default domain should not resolve once overridden, got %q", got)
	}
}

func TestIsInternalHost(t *testing.T) {
	for _, host := range []string{"", "localhost", "127.0.0.1:8080", "::1", "api", "caddy", "host.docker.internal"} {
		if !IsInternalHost(host) {
			t.Errorf("IsInternalHost(%q) = false, want true", host)
		}
	}
	for _, host := range []string{"api.xamltech.com", "posgo.xamltech.com", "evil.com", "api.attacker.io"} {
		if IsInternalHost(host) {
			t.Errorf("IsInternalHost(%q) = true, want false", host)
		}
	}
}

func TestLoadSurfaceRoutingDefaultsToDisabled(t *testing.T) {
	clearEnv()
	r, err := loadSurfaceRouting()
	if err != nil {
		t.Fatal(err)
	}
	if r.Enabled {
		t.Error("surface routing must be opt-in")
	}
	if r.CompanyDomain != DefaultCompanyDomain || r.SaaSDomain != DefaultSaaSDomain || r.POSDomain != DefaultPOSDomain {
		t.Errorf("default domains: %+v", r)
	}
}

func TestLoadSurfaceRoutingParsesEnv(t *testing.T) {
	clearEnv()
	t.Setenv("SURFACE_ROUTING_ENABLED", "true")
	t.Setenv("COMPANY_DOMAIN", "xamltech.com")
	t.Setenv("SAAS_DOMAIN", "https://api.xamltech.com/")
	t.Setenv("POS_DOMAIN", "posgo.xamltech.com:443")
	r, err := loadSurfaceRouting()
	if err != nil {
		t.Fatal(err)
	}
	if !r.Enabled {
		t.Error("expected routing enabled")
	}
	if r.Resolve("posgo.xamltech.com") != SurfacePOS {
		t.Errorf("POS domain not normalized: %+v", r)
	}
	if r.Resolve("api.xamltech.com") != SurfaceSaaS {
		t.Errorf("SaaS domain not normalized: %+v", r)
	}
}

func TestLoadSurfaceRoutingRejectsBadValues(t *testing.T) {
	clearEnv()
	t.Setenv("SURFACE_ROUTING_ENABLED", "maybe")
	if _, err := loadSurfaceRouting(); err == nil {
		t.Error("expected error for a non-boolean SURFACE_ROUTING_ENABLED")
	}

	clearEnv()
	t.Setenv("SURFACE_ROUTING_ENABLED", "true")
	t.Setenv("POS_DOMAIN", "   ")
	if _, err := loadSurfaceRouting(); err == nil {
		t.Error("expected error for a blank POS_DOMAIN")
	}

	clearEnv()
	t.Setenv("SURFACE_ROUTING_ENABLED", "true")
	t.Setenv("SAAS_DOMAIN", "api.xamltech.com:443")
	t.Setenv("POS_DOMAIN", "API.xamltech.com")
	if _, err := loadSurfaceRouting(); err == nil {
		t.Error("expected error when two surfaces share a domain")
	}
}
