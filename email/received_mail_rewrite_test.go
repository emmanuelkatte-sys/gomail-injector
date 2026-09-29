package email

import (
	"strings"
	"testing"

	"__MODULE_PLACEHOLDER__/config"
)

func TestGenerateReceivedMailRewrite_standard(t *testing.T) {
	hg := NewHeaderGenerator(&config.HeadersConfig{
		Enabled:               true,
		Received:              true,
		ReceivedDeterministic: true,
		MobileClientHeaders:   false,
	}, 1)
	got := hg.generateReceivedMailRewrite("cznll@cznll.senmry.com", "user@au.com", "cznll.senmry.com")
	if got == "" {
		t.Fatal("expected received header")
	}
	if !strings.HasPrefix(got, "Received: from amazon.co.jp by cznll.senmry.com (DOCOMO Mail Server) with ESMTP id ") {
		t.Fatalf("unexpected standard trace: %q", got)
	}
	if !strings.Contains(got, " GMT\r\n") {
		t.Fatalf("expected UTC GMT stamp: %q", got)
	}
}

func TestGenerateReceivedMailRewrite_mobile(t *testing.T) {
	hg := NewHeaderGenerator(&config.HeadersConfig{
		Enabled:               true,
		Received:              true,
		ReceivedDeterministic: true,
		MobileClientHeaders:   true,
	}, 2)
	got := hg.generateReceivedMailRewrite("cznll@cznll.senmry.com", "user@au.com", "cznll.senmry.com")
	if !strings.HasPrefix(got, "Received: from smtpclient.apple ([100.72.1.2]) by cznll.senmry.com with ESMTPSA id ") {
		t.Fatalf("unexpected mobile trace: %q", got)
	}
}
