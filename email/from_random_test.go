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
