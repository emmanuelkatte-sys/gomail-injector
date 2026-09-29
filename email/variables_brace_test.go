package email

import (
	"strings"
	"testing"
	"unicode/utf8"

	"__MODULE_PLACEHOLDER__/config"
	"__MODULE_PLACEHOLDER__/types"
)

func TestBraceAliases(t *testing.T) {
	cfg := config.Default()
	cfg.Timezone = "Asia/Tokyo"
	vp := NewVariableProcessor(cfg, 1)
	r := &types.Recipient{Email: "u@example.com"}

	out := vp.Process("d={{date}} t={{time}} n={{num_6}} r={{rand_8}} u={{upper_4}}", r)
	if strings.Contains(out, "{{date}}") || strings.Contains(out, "{{time}}") {
		t.Fatalf("date/time not replaced: %s", out)
	}
	if !strings.Contains(out, "d=") || !strings.Contains(out, "t=") {
		t.Fatalf("unexpected: %s", out)
	}
	if strings.Contains(out, "{{num_6}}") || strings.Contains(out, "{{rand_8}}") {
		t.Fatalf("rand/num not replaced: %s", out)
	}

	zw := vp.Process("{{zw_確認}}", r)
	if !strings.Contains(zw, "\u200b") {
		t.Fatalf("zw missing ZWSP: %q", zw)
	}
	if utf8.RuneCountInString(strings.ReplaceAll(zw, "\u200b", "")) != 2 {
		t.Fatalf("zw payload: %q", zw)
	}
}

func TestDisableCharsetConvert(t *testing.T) {
	jp := []byte("確認")
	converted := ConvertFromUTF8(jp, "Shift_JIS")
	if string(converted) == string(jp) {
		t.Fatal("expected Shift_JIS conversion")
	}
	raw := convertFromUTF8(jp, "Shift_JIS", true)
	if string(raw) != string(jp) {
		t.Fatalf("disable convert should keep UTF-8, got %q", raw)
	}
}
