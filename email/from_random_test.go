package email

import (
	"strings"
	"testing"
)

func TestRandomFromSameDomain(t *testing.T) {
	out := randomFromSameDomain("brjjv@brjjv.huisaudio.com")
	if out == "brjjv@brjjv.huisaudio.com" {
		t.Fatalf("expected different local-part")
	}
	if !strings.HasSuffix(out, "@brjjv.huisaudio.com") {
		t.Fatalf("domain must be preserved, got %q", out)
	}
}

func TestRandomFromSameDomain_invalid(t *testing.T) {
	in := "not-an-email"
	if randomFromSameDomain(in) != in {
		t.Fatalf("invalid address should be unchanged")
	}
}

func TestRandomEmailLocalPart_unique(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 30; i++ {
		seen[randomEmailLocalPart()] = struct{}{}
	}
	if len(seen) < 10 {
		t.Fatalf("expected varied local parts, got %d unique", len(seen))
	}
}

func TestRandomSubdomainSameDomain(t *testing.T) {
	out := randomSubdomainSameDomain("sato@yak.cooldans.shop")
	if !strings.HasPrefix(out, "sato@") {
		t.Fatalf("expected user part sato, got %q", out)
	}
	if !strings.HasSuffix(out, ".cooldans.shop") {
		t.Fatalf("expected root domain .cooldans.shop, got %q", out)
	}
	if out == "sato@yak.cooldans.shop" {
		t.Fatalf("expected randomized subdomain, got original %q", out)
	}

	out2 := randomSubdomainSameDomain("user@example.com")
	if !strings.HasPrefix(out2, "user@") || !strings.HasSuffix(out2, ".example.com") {
		t.Fatalf("expected user@*.example.com, got %q", out2)
	}
}
