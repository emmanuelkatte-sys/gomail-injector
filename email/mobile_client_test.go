package email

import (
	"strings"
	"testing"
	"time"

	"__MODULE_PLACEHOLDER__/config"
	"__MODULE_PLACEHOLDER__/utils"
)

func TestGenerateMobileClientReceived(t *testing.T) {
	hg := NewHeaderGenerator(&config.HeadersConfig{Enabled: true}, 1)
	got := hg.GenerateMobileClientReceived("mail.example.com")
	if got == "" {
		t.Fatal("expected received header")
	}
	if !strings.HasPrefix(got, "Received: from ") {
		t.Fatalf("unexpected prefix: %q", got)
	}
	for _, needle := range []string{"ESMTPSA", "mail.example.com", "by mail.example.com"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("missing %q in %q", needle, got)
		}
	}
	for _, forbidden := range []string{"Authenticated sender", "TLS1_", "Haraka"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("forbidden %q in %q", forbidden, got)
		}
	}
}

func TestGenerateNaturalClientMessageID(t *testing.T) {
	id := utils.GenerateNaturalClientMessageID("sub.example.com")
	if !strings.Contains(id, "@sub.example.com") {
		t.Fatalf("bad id: %q", id)
	}
	parts := strings.Split(strings.TrimSuffix(id, "@sub.example.com"), ".")
	if len(parts) < 4 {
		t.Fatalf("expected compact timestamp segments, got %q", id)
	}
	if _, err := time.Parse("20060102150405", parts[0]); err != nil {
		t.Fatalf("timestamp parse: %v (%q)", err, parts[0])
	}
}

func TestIsMobileBlockedHeader(t *testing.T) {
	if IsMobileBlockedHeader("Authentication-Results") {
		// ok
	} else {
		t.Fatal("auth results should be blocked")
	}
	if IsMobileBlockedHeader("DKIM-Signature") {
		t.Fatal("injector fake dkim should not be blocked")
	}
	if IsMobileBlockedHeader("Subject") {
		t.Fatal("subject should not be blocked")
	}
}
