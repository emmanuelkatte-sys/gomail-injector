package email

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestApplyReverseBidiOverlayKeepsExisting(t *testing.T) {
	chunked := string(bidiLRI) + string(bidiRLO) + "Ci" + string(bidiPDF) + string(bidiPDI) +
		string(bidiLRI) + string(bidiRLO) + "ol" + string(bidiPDF) + string(bidiPDI) +
		string(bidiLRI) + string(bidiRLO) + "du" + string(bidiPDF) + string(bidiPDI)
	out := ApplyReverseBidiAtKeywords(chunked+" Apple", []string{"iCloud", "Apple"}, false)
	if RestoreBidiPlaintext(out) != "iCloud Apple" {
		t.Fatalf("restore = %q", RestoreBidiPlaintext(out))
	}
	if !hasRtlOverride(out) {
		t.Fatal("expected Apple to be reversed")
	}
}

func TestApplyReverseBidiVSMatch(t *testing.T) {
	text := "ご登録情報の確\uFE00認"
	out := ApplyReverseBidiAtKeywords(text, []string{"確認"}, false)
	if !hasRtlOverride(out) {
		t.Fatal("VS-separated 確認 should match")
	}
	if RestoreBidiPlaintext(out) != "ご登録情報の確認" {
		t.Fatalf("restore = %q", RestoreBidiPlaintext(out))
	}
}

func TestApplyReverseBidiDecoyDoesNotBlock(t *testing.T) {
	html := "<td>" + string(bidiRLO) + "トカド様用利信セソ用配</td>\nSAISON CARD"
	out := ApplyReverseBidiAtKeywords(html, []string{"SAISON", "CARD"}, true)
	if strings.Contains(out, "SAISON") {
		t.Fatal("SAISON should be obfuscated after decoy RLO")
	}
	if !strings.Contains(RestoreBidiPlaintext(out), "SAISON") {
		t.Fatal("restore should contain SAISON")
	}
}

func TestWrapUsesLRIAndPDI(t *testing.T) {
	out := ApplyReverseBidiAtKeywords("お支払い方法の確認", []string{"支払", "確認"}, false)
	if !strings.Contains(out, string(bidiLRI)+string(bidiRLO)) {
		t.Fatal("expected U+2066 + U+202E opener")
	}
	if !strings.Contains(out, string(bidiPDF)+string(bidiPDI)) {
		t.Fatal("expected U+202C + U+2069 closer")
	}
	if RestoreBidiPlaintext(out) != "お支払い方法の確認" {
		t.Fatalf("restore = %q", RestoreBidiPlaintext(out))
	}
}

func TestChunkSizeRandom2to4(t *testing.T) {
	for i := 0; i < 20; i++ {
		out := ApplyReverseBidiAtKeywords("iCloud", []string{"iCloud"}, false)
		if RestoreBidiPlaintext(out) != "iCloud" {
			t.Fatalf("restore = %q", RestoreBidiPlaintext(out))
		}
		inner := 0
		rs := []rune(out)
		for j := 0; j < len(rs); j++ {
			if rs[j] == bidiRLO {
				k := j + 1
				for k < len(rs) && rs[k] != bidiPDF {
					k++
				}
				n := k - j - 1
				if n < 2 || n > 4 {
					t.Fatalf("chunk rune count %d", n)
				}
				inner++
				j = k
			}
		}
		if inner == 0 {
			t.Fatal("expected at least one reverse chunk")
		}
		_ = utf8.RuneCountInString(out)
	}
}
