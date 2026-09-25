package config

import (
	"net"
	"strings"
	"testing"
)

func TestParseTrustedProxiesDefaultsToTheEdgeAndCloudflare(t *testing.T) {
	got, err := ParseTrustedProxies("")
	if err != nil {
		t.Fatalf("ParseTrustedProxies(\"\"): %v", err)
	}
	if len(got) != len(DefaultTrustedProxies()) {
		t.Fatalf("default list has %d entries, want %d", len(got), len(DefaultTrustedProxies()))
	}
	for _, want := range []string{"127.0.0.1/32", "172.16.0.0/12", "104.16.0.0/13", "198.41.128.0/17"} {
		if !contains(got, want) {
			t.Errorf("default trusted list is missing %s", want)
		}
	}
}

func TestDefaultTrustedProxiesCoversTheContainerNetwork(t *testing.T) {
	for _, cidr := range DefaultTrustedProxies() {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			t.Errorf("default list has an invalid CIDR %q: %v", cidr, err)
		}
	}
}

func TestParseTrustedProxiesOverride(t *testing.T) {
	got, err := ParseTrustedProxies(" 10.1.0.0/16, 192.168.5.5/32 ")
	if err != nil {
		t.Fatalf("ParseTrustedProxies: %v", err)
	}
	want := []string{"10.1.0.0/16", "192.168.5.5/32"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d: got %s, want %s", i, got[i], want[i])
		}
	}
}

func TestParseTrustedProxiesRejectsUntrusted(t *testing.T) {
	cases := map[string]string{
		"not a cidr":        "definitely-not-a-cidr",
		"bare address":      "10.0.0.1",
		"catch-all IPv4":    "0.0.0.0/0",
		"catch-all IPv6":    "::/0",
		"one bad in a list": "10.0.0.0/8,nope",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseTrustedProxies(raw); err == nil {
				t.Fatalf("ParseTrustedProxies(%q) accepted an untrusted value", raw)
			}
		})
	}
}

func TestParseTrustedProxiesRejectionExplainsWhy(t *testing.T) {
	_, err := ParseTrustedProxies("0.0.0.0/0")
	if err == nil {
		t.Fatal("expected an error for 0.0.0.0/0")
	}
	if !strings.Contains(err.Error(), "client-controlled") {
		t.Errorf("error should say why the prefix is rejected, got: %v", err)
	}
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
