package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootDomainFromAddress(t *testing.T) {
	cases := map[string]string{
		"user@xxx.example.com": "example.com",
		"a@b.co.jp":            "co.jp",
		"only@localhost":       "localhost",
		"no-at":                "localhost",
		"User@Sub.Example.COM": "example.com",
	}
	for in, want := range cases {
		if got := RootDomainFromAddress(in); got != want {
			t.Fatalf("RootDomainFromAddress(%q)=%q want %q", in, got, want)
		}
	}
}

func TestResolvedSecurity(t *testing.T) {
	if got := (SMTPConfig{Security: "starttls"}).ResolvedSecurity(); got != "STARTTLS" {
		t.Fatalf("security starttls → %q", got)
	}
	if got := (SMTPConfig{Security: "SSL"}).ResolvedSecurity(); got != "SSL" {
		t.Fatalf("security SSL → %q", got)
	}
	if got := (SMTPConfig{UseTLS: true}).ResolvedSecurity(); got != "STARTTLS" {
		t.Fatalf("use_tls infer → %q", got)
	}
	if got := (SMTPConfig{Port: 465}).ResolvedSecurity(); got != "SSL" {
		t.Fatalf("port 465 infer → %q", got)
	}
	if got := (SMTPConfig{Port: 587}).ResolvedSecurity(); got != "" {
		t.Fatalf("plain 587 → %q", got)
	}
	if got := (SMTPConfig{Security: "none", UseTLS: true, Port: 465}).ResolvedSecurity(); got != "" {
		t.Fatalf("explicit none should disable infer, got %q", got)
	}
}

func TestLoadFromFileSMTPAlign(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	yaml := `
timezone: Asia/Shanghai
smtp:
  security: STARTTLS
  skip_verify: true
  max_conn: 8
  port: 587
performance:
  min_interval: 10
  max_interval: 20
encoding:
  disable_charset_convert: true
html_mutator:
  enabled: true
  css_jitter: true
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Timezone != "Asia/Shanghai" {
		t.Fatalf("timezone=%q", cfg.Timezone)
	}
	if cfg.SMTP.ResolvedSecurity() != "STARTTLS" {
		t.Fatalf("security=%q", cfg.SMTP.ResolvedSecurity())
	}
	if !cfg.SMTP.SkipVerify {
		t.Fatal("skip_verify should be true")
	}
	if cfg.SMTP.MaxConn != 8 {
		t.Fatalf("max_conn=%d", cfg.SMTP.MaxConn)
	}
	if cfg.Performance.MinInterval != 10 || cfg.Performance.MaxInterval != 20 {
		t.Fatalf("intervals %d/%d", cfg.Performance.MinInterval, cfg.Performance.MaxInterval)
	}
	if !cfg.Encoding.DisableCharsetConvert {
		t.Fatal("disable_charset_convert")
	}
	if !cfg.HtmlMutator.Enabled || !cfg.HtmlMutator.CssJitter {
		t.Fatal("html_mutator not loaded")
	}
}

func TestTimezoneFallbackEmailThenTokyo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(path, []byte("email:\n  timezone: Asia/Tokyo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ResolvedTimezone() != "Asia/Tokyo" {
		t.Fatalf("got %q", cfg.ResolvedTimezone())
	}

	path2 := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path2, []byte("sender:\n  from_address: a@b.com\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg2, err := LoadFromFile(path2)
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.ResolvedTimezone() != "Asia/Tokyo" {
		t.Fatalf("default timezone %q", cfg2.ResolvedTimezone())
	}
}
