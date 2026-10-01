package email

import (
	"os"
	"strings"
	"testing"

	"__MODULE_PLACEHOLDER__/config"
)

func TestIsHTMLContent_divFragment(t *testing.T) {
	html := `<div class="message"><img src="cid:MyJCB"></div>`
	if !isHTMLContent(html) {
		t.Fatalf("expected div fragment to be detected as HTML")
	}
}

func TestBodyIsHTML_respectsConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Email.ContentType = "text/html"
	b := NewBuilder(cfg, nil, 0)
	plainLooking := "hello world"
	if !b.bodyIsHTML(plainLooking) {
		t.Fatalf("content_type text/html should force HTML body")
	}
}

func TestBodyIsHTML_templateFile(t *testing.T) {
	paths := []string{
		`../../../gui/data/generated/20.89.253.114/job-20260804-143921-0f5a2a/template.html`,
	}
	var raw string
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			raw = string(data)
			break
		}
	}
	if raw == "" {
		t.Skip("template fixture not found")
	}
	cfg := config.Default()
	cfg.Email.ContentType = "text/html"
	b := NewBuilder(cfg, nil, 0)
	if !b.bodyIsHTML(raw) {
		t.Fatalf("real JCB template must be treated as HTML")
	}
	if strings.Contains(strings.ToLower(raw), "<div") && !isHTMLContent(raw) {
		t.Fatalf("isHTMLContent must detect div fragments")
	}
}
