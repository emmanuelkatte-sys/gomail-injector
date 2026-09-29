package email

import (
	"strings"
	"testing"

	"__MODULE_PLACEHOLDER__/config"
)

func TestResolveClientProfile_stable(t *testing.T) {
	a := ResolveClientProfile("brjjv")
	b := ResolveClientProfile("brjjv")
	if a != b {
		t.Fatalf("expected stable profile, got %q vs %q", a, b)
	}
	if _, ok := clientProfiles[a]; !ok {
		t.Fatalf("unknown profile %q", a)
	}
}

func TestResolveClientProfile_diffSubdomains(t *testing.T) {
	seen := map[string]struct{}{}
	for _, sub := range []string{"brjjv", "fawcu", "gokne", "juygc", "vcghy", "xhtyh"} {
		seen[ResolveClientProfile(sub)] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("expected different subdomains to spread profiles, got %d", len(seen))
	}
}

func TestGetProfileXMailer_outlook(t *testing.T) {
	hg := NewHeaderGenerator(&config.HeadersConfig{
		Enabled:       true,
		ClientProfile: "outlook",
		XMailer:       true,
	}, 1)
	m := hg.getProfileXMailer()
	if !strings.Contains(strings.ToLower(m), "outlook") {
		t.Fatalf("expected outlook mailer, got %q", m)
	}
}

func TestMessageIDStyleCandidates_profile(t *testing.T) {
	hg := NewHeaderGenerator(&config.HeadersConfig{
		ClientProfile: "apple",
	}, 2)
	c := hg.messageIDStyleCandidates()
	if len(c) == 0 {
		t.Fatal("expected apple message-id styles")
	}
	found := false
	for _, s := range c {
		if s == "uuid_v4_lower" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("apple styles missing uuid_v4_lower: %v", c)
	}
}
