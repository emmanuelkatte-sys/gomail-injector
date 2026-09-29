package smtp

import "testing"

func TestSmtpSecurityInfer(t *testing.T) {
	if got := smtpSecurity(PoolConfig{Security: "starttls"}); got != "STARTTLS" {
		t.Fatalf("got %q", got)
	}
	if got := smtpSecurity(PoolConfig{Security: "smtps"}); got != "SMTPS" {
		t.Fatalf("got %q", got)
	}
	if got := smtpSecurity(PoolConfig{UseTLS: true}); got != "STARTTLS" {
		t.Fatalf("use_tls infer %q", got)
	}
	if got := smtpSecurity(PoolConfig{Addr: "127.0.0.1:465"}); got != "SSL" {
		t.Fatalf("port 465 infer %q", got)
	}
	if got := smtpSecurity(PoolConfig{Addr: "127.0.0.1:587"}); got != "" {
		t.Fatalf("plain %q", got)
	}
}
