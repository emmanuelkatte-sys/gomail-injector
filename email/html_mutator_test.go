package email

import (
	"strings"
	"testing"

	"__MODULE_PLACEHOLDER__/config"
)

func TestMutateHTML_cssJitterChangesSource(t *testing.T) {
	in := `<div style="font-size: 14px; line-height: 1.9; color: #333333;">hello</div>`
	cfg := config.HtmlMutatorConfig{Enabled: true, CssJitter: true}
	out := MutateHTML(in, cfg, "test@au.com|tpl.html|1")
	if out == in {
		t.Fatalf("expected CSS jitter to change HTML source")
	}
	if !strings.Contains(out, "#333333") {
		t.Fatalf("text color should be preserved, got %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected visible text preserved, got %q", out)
	}
}

func TestMutateHTML_preservesBackgroundAndAlignment(t *testing.T) {
	in := `<div style="background-color:#ffffff;text-align:center;margin:0 auto;max-width:580px;padding:10px 20px;color:#00146E;font-size:14px;">x</div>`
	cfg := config.HtmlMutatorConfig{Enabled: true, CssJitter: true}
	out := MutateHTML(in, cfg, "preserve-layout")
	for _, keep := range []string{
		"background-color:#ffffff",
		"text-align:center",
		"margin:0 auto",
		"max-width:580px",
		"padding:10px 20px",
		"color:#00146E",
	} {
		if !strings.Contains(out, keep) {
			t.Fatalf("expected %q preserved in %q", keep, out)
		}
	}
}

func TestMutateHTML_injectAttrs(t *testing.T) {
	in := `<div style="margin: 0;">text</div>`
	cfg := config.HtmlMutatorConfig{Enabled: true, InjectAttrs: true}
	out := MutateHTML(in, cfg, "seed-a")
	if !strings.Contains(out, `data-x="`) && !strings.Contains(out, `class="c-`) {
		t.Fatalf("expected injected attributes, got %q", out)
	}
}

func TestMutateHTML_tagSwapOutsideTable(t *testing.T) {
	in := `<p style="margin:0;">A</p><table><tr><td><p>B</p></td></tr></table>`
	cfg := config.HtmlMutatorConfig{
		Enabled: true, TagSwap: true, PreserveTables: true,
	}
	seenDiv := false
	seenTableP := false
	for i := 0; i < 20; i++ {
		out := MutateHTML(in, cfg, "swap@au.com|tpl|"+string(rune('a'+i)))
		if strings.Contains(out, "<div") && strings.Contains(out, "</div>") {
			seenDiv = true
		}
		if strings.Contains(out, "<td><p>B</p></td>") || strings.Contains(out, "<td><p ") {
			seenTableP = true
		}
	}
	if !seenDiv {
		t.Fatalf("expected p->div swap outside table within retries")
	}
	if !seenTableP {
		t.Fatalf("expected table cell <p> preserved")
	}
}

func TestMutateHTML_skipsLinksAndImages(t *testing.T) {
	in := `<a href="{url}"><img src="cid:Logo"></a><div>ok</div>`
	cfg := config.HtmlMutatorConfig{Enabled: true, InjectAttrs: true}
	out := MutateHTML(in, cfg, "skip-links")
	if strings.Contains(out, `<a data-x=`) || strings.Contains(out, `<img data-x=`) {
		t.Fatalf("should not inject attrs on a/img, got %q", out)
	}
	if !strings.Contains(out, `href="{url}"`) || !strings.Contains(out, `src="cid:Logo"`) {
		t.Fatalf("href/src must be preserved, got %q", out)
	}
}

func TestMutateHTML_disabled(t *testing.T) {
	in := `<p style="font-size:14px;">x</p>`
	cfg := config.HtmlMutatorConfig{Enabled: false, CssJitter: true, InjectAttrs: true, TagSwap: true}
	out := MutateHTML(in, cfg, "noop")
	if out != in {
		t.Fatalf("disabled mutator must return input unchanged")
	}
}
