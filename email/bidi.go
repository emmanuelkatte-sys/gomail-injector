package email

import (
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// 【对齐 PHP ReverseBidi】不剥已有倒序/零宽；只对仍明文的关键字套混淆。
// 每段随机 2–4 字：U+2066 + U+202E + 反序 + U+202C + U+2069。
// 剩余单字原位当分隔；标点/空白不倒序。
//
// 限制：仅 UTF-8；调用方须先 bidiCharsetSupported。
const (
	bidiLRI rune = 0x2066 // Left-to-Right Isolate
	bidiRLO rune = 0x202E // Right-to-Left Override
	bidiRLE rune = 0x202B // Right-to-Left Embedding
	bidiPDF rune = 0x202C // Pop Directional Formatting
	bidiPDI rune = 0x2069 // Pop Directional Isolate
)

// bidiReverseHide：与正文关键字同一「随机分段 + 每段独立 RLO」规则。
func bidiReverseHide(text string) string {
	return obfuscateBidiWord(text)
}

func obfuscateBidiWord(text string) string {
	return obfuscateBidiWordRNG(text, nil)
}

func obfuscateBidiWordRNG(text string, rng *rand.Rand) string {
	if text == "" {
		return text
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	runes := []rune(text)
	var sb strings.Builder
	sb.Grow(len(text) + 48)
	run := make([]rune, 0, 16)
	flush := func() {
		if len(run) == 0 {
			return
		}
		sb.WriteString(obfuscateBidiRun(run, rng))
		run = run[:0]
	}
	for _, r := range runes {
		if isBidiSymbolOrSpace(r) {
			flush()
			sb.WriteRune(r)
		} else {
			run = append(run, r)
		}
	}
	flush()
	return sb.String()
}

// obfuscateBidiRun：随机 2–4 字一段，U+2066…U+2069 包裹；剩余 1 字当分隔。
func obfuscateBidiRun(run []rune, rng *rand.Rand) string {
	n := len(run)
	if n == 0 {
		return ""
	}
	if n == 1 {
		return string(run[0])
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	var sb strings.Builder
	sb.Grow(n*4 + 16)
	i := 0
	for _, size := range randomBidiSlots(n, rng) {
		if size == 1 {
			sb.WriteRune(run[i])
			i++
			continue
		}
		chunk := run[i : i+size]
		sb.WriteRune(bidiLRI)
		sb.WriteRune(bidiRLO)
		for j := len(chunk) - 1; j >= 0; j-- {
			sb.WriteRune(chunk[j])
		}
		sb.WriteRune(bidiPDF)
		sb.WriteRune(bidiPDI)
		i += size
	}
	return sb.String()
}

func randomBidiSlots(n int, rng *rand.Rand) []int {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	var slots []int
	remaining := n
	hasReverse := false
	for remaining > 0 {
		if remaining == 1 {
			slots = append(slots, 1)
			break
		}
		choices := make([]int, 0, 5)
		if hasReverse || remaining-1 >= 2 {
			choices = append(choices, 1)
		}
		max := 4
		if remaining < max {
			max = remaining
		}
		for size := 2; size <= max; size++ {
			choices = append(choices, size)
		}
		size := choices[rng.Intn(len(choices))]
		slots = append(slots, size)
		if size >= 2 {
			hasReverse = true
		}
		remaining -= size
	}
	if !hasReverse && n >= 2 {
		take := 4
		if n < take {
			take = n
		}
		out := []int{take}
		for i := 0; i < n-take; i++ {
			out = append(out, 1)
		}
		return out
	}
	return slots
}

// isBidiSymbolOrSpace mirrors PHP ReverseBidi::isSymbolOrSpace (\p{P}\p{S}\p{Z}).
func isBidiSymbolOrSpace(r rune) bool {
	return unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.IsSpace(r)
}

// ApplyReverseBidiAtKeywords 不剥已有倒序/零宽，只对仍明文的关键字套混淆。
func ApplyReverseBidiAtKeywords(text string, keywords []string, isHTML bool) string {
	return applyReverseBidiAtKeywordsRNG(text, keywords, isHTML, nil)
}

func applyReverseBidiAtKeywordsRNG(text string, keywords []string, isHTML bool, rng *rand.Rand) string {
	if text == "" || len(keywords) == 0 {
		return text
	}
	text = normalizeBidiEntities(text)
	norm := make([]string, 0, len(keywords))
	seen := map[string]bool{}
	for _, kw := range keywords {
		kw = stripVariationSelectors(kw)
		if kw == "" || seen[kw] || utf8.RuneCountInString(kw) < 2 {
			continue
		}
		seen[kw] = true
		norm = append(norm, kw)
	}
	sort.Slice(norm, func(i, j int) bool { return utf8.RuneCountInString(norm[i]) > utf8.RuneCountInString(norm[j]) })
	for _, kw := range norm {
		text = applyBidiInKeywordMatches(text, kw, isHTML, rng)
	}
	return normalizeBidiEntities(text)
}

func applyBidiInKeywordMatches(text, keyword string, isHTML bool, rng *rand.Rand) string {
	if keyword == "" || utf8.RuneCountInString(keyword) < 2 {
		return text
	}
	type span struct{ start, length int }
	var spans []span
	for _, pair := range findKeywordSpans(text, keyword) {
		pos, length := pair[0], pair[1]
		word := text[pos : pos+length]
		if isHTML && (isInsideHTMLTag(text, pos, pos+length) || isInsideHTMLAttribute(text, pos)) {
			continue
		}
		if isInsideBidiRun(text, pos) || hasRtlOverride(word) {
			continue
		}
		spans = append(spans, span{pos, length})
	}
	for i := len(spans) - 1; i >= 0; i-- {
		s := spans[i]
		word := stripVariationSelectors(text[s.start : s.start+s.length])
		word = RemoveZeroWidthRunes(word)
		replaced := obfuscateBidiWordRNG(word, rng)
		text = text[:s.start] + replaced + text[s.start+s.length:]
	}
	return text
}

var keywordGapReParts = `[\x{FE00}-\x{FE0F}\x{200B}\x{200C}\x{200D}\x{FEFF}\x{2060}]*`

// findKeywordSpans 允许关键字字符之间夹变体选择符/零宽。返回 byte [start, length]。
func findKeywordSpans(text, keyword string) [][2]int {
	if text == "" || keyword == "" {
		return nil
	}
	parts := make([]string, 0, utf8.RuneCountInString(keyword))
	for _, r := range keyword {
		parts = append(parts, regexp.QuoteMeta(string(r)))
	}
	re, err := regexp.Compile(strings.Join(parts, keywordGapReParts))
	if err != nil {
		return nil
	}
	idx := re.FindAllStringIndex(text, -1)
	if len(idx) == 0 {
		return nil
	}
	out := make([][2]int, 0, len(idx))
	for _, p := range idx {
		out = append(out, [2]int{p[0], p[1] - p[0]})
	}
	return out
}

// isInsideBidiRun：未闭合 RLO 只作用于同一 HTML 文本节点（顶部伪装条不挡住后面关键字）。
func isInsideBidiRun(text string, bytePos int) bool {
	if bytePos <= 0 || bytePos > len(text) {
		return false
	}
	prefix := text[:bytePos]
	rlo := strings.LastIndex(prefix, string(bidiRLO))
	rle := strings.LastIndex(prefix, string(bidiRLE))
	open := -1
	if rlo >= 0 && (rle < 0 || rlo > rle) {
		open = rlo
	} else if rle >= 0 {
		open = rle
	}
	if open < 0 {
		return false
	}
	slice := prefix[open:]
	if strings.ContainsRune(slice, bidiPDF) {
		return false
	}
	return !strings.Contains(slice, "<")
}

var bidiEntityRe = regexp.MustCompile(`(?i)&#x([0-9a-f]+);|&#([0-9]+);`)

func isBidiCodepoint(r rune) bool {
	switch r {
	case 0x200B, 0x200C, 0x200D, 0xFEFF, 0x2060,
		0x202A, 0x202B, 0x202C, 0x202D, 0x202E,
		0x2066, 0x2067, 0x2068, 0x2069:
		return true
	default:
		return false
	}
}

func normalizeBidiEntities(text string) string {
	if text == "" || !strings.Contains(text, "&#") {
		return text
	}
	return bidiEntityRe.ReplaceAllStringFunc(text, func(m string) string {
		sub := bidiEntityRe.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		var cp int64
		var err error
		if sub[1] != "" {
			cp, err = strconv.ParseInt(sub[1], 16, 32)
		} else {
			cp, err = strconv.ParseInt(sub[2], 10, 32)
		}
		if err != nil || !isBidiCodepoint(rune(cp)) {
			return m
		}
		return string(rune(cp))
	})
}

func ensureIsolatedFormat(text string) string {
	text = normalizeBidiEntities(text)
	if !hasRtlOverride(text) {
		return text
	}
	rs := []rune(text)
	var out []rune
	for i := 0; i < len(rs); i++ {
		ch := rs[i]
		if ch != bidiRLO && ch != bidiRLE {
			out = append(out, ch)
			continue
		}
		if i > 0 && (rs[i-1] == bidiLRI || rs[i-1] == 0x2067 || rs[i-1] == 0x2068) {
			out = append(out, ch)
			continue
		}
		j := i + 1
		for j < len(rs) && rs[j] != bidiPDF {
			j++
		}
		if j >= len(rs) {
			out = append(out, ch)
			continue
		}
		out = append(out, bidiLRI)
		out = append(out, rs[i:j+1]...)
		out = append(out, bidiPDI)
		i = j
	}
	return string(out)
}

func isolateKeepZW(s string) string {
	if s == "" || !hasRtlOverride(s) {
		return s
	}
	if strings.HasPrefix(s, string(bidiLRI)) && strings.HasSuffix(s, string(bidiPDI)) {
		return s
	}
	return string(bidiLRI) + s + string(bidiPDI)
}

// FinalizeBidiHeader 对齐 PHP ReverseBidi::finalizeHeaderField。
func FinalizeBidiHeader(text string) string {
	text = normalizeBidiEntities(text)
	text = ensureIsolatedFormat(text)
	return isolateKeepZW(text)
}

// randomRange 随机起止（至少 minLen）；n>4 时避免整段覆盖。对齐 PHP ReverseBidi::randomRange。
func randomRange(n, minLen int) (int, int) {
	if n <= minLen {
		return 0, n
	}
	maxStart := n - minLen
	start := rand.Intn(maxStart + 1)
	end := start + minLen + rand.Intn(n-(start+minLen)+1)
	if n > 4 && start == 0 && end == n {
		if maxStart >= 1 && rand.Intn(2) == 1 {
			start = 1 + rand.Intn(maxStart)
		} else if n-1 >= minLen {
			end = n - 1
		} else {
			start = 1
		}
	}
	return start, end
}

func stripVariationSelectors(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 0xFE00 && r <= 0xFE0F {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// bidiCharsetSupported 仅 UTF-8（含空串默认）支持 bidi 控制符；其它字符集无法编码这些字符。
func bidiCharsetSupported(charset string) bool {
	c := strings.ToUpper(strings.TrimSpace(charset))
	return c == "" || c == "UTF-8" || c == "UTF8"
}

func hasRtlOverride(s string) bool {
	return strings.ContainsRune(s, bidiRLO) || strings.ContainsRune(s, bidiRLE)
}

// IsolateBidi wraps leftover RLO input in LRI/PDI so a trailing '+' is not
// pulled into the RTL run (visual iC+duol vs iCloud+).
func IsolateBidi(s string) string {
	s = RemoveZeroWidthRunes(s)
	if s == "" || !hasRtlOverride(s) {
		return s
	}
	rs := []rune(s)
	if len(rs) >= 2 && rs[0] == bidiLRI && rs[len(rs)-1] == bidiPDI {
		return s
	}
	return string(bidiLRI) + s + string(bidiPDI)
}

// RestoreBidiPlaintext undoes leftover RLO/RLE hide (and isolate wrappers) so
// subject/display/template stored from a previous obfuscation pass become plaintext.
func RestoreBidiPlaintext(s string) string {
	if s == "" {
		return s
	}
	s = unwindRTLOverrides(s)
	s = unwrapIsolates(s)
	s = stripBidiControls(s)
	return stripVariationSelectors(s)
}

func unwindRTLOverrides(s string) string {
	for {
		rs := []rune(s)
		start := -1
		for i := len(rs) - 1; i >= 0; i-- {
			if rs[i] == bidiRLO || rs[i] == bidiRLE {
				start = i
				break
			}
		}
		if start < 0 {
			return s
		}
		end := -1
		for i := start + 1; i < len(rs); i++ {
			if rs[i] == bidiPDF {
				end = i
				break
			}
		}
		if end < 0 {
			s = string(append(append([]rune{}, rs[:start]...), rs[start+1:]...))
			continue
		}
		inner := reverseRunes(rs[start+1 : end])
		out := append(append([]rune{}, rs[:start]...), inner...)
		out = append(out, rs[end+1:]...)
		s = string(out)
	}
}

func unwrapIsolates(s string) string {
	for {
		rs := []rune(s)
		start := -1
		for i := len(rs) - 1; i >= 0; i-- {
			if rs[i] == bidiLRI || rs[i] == 0x2067 || rs[i] == 0x2068 {
				start = i
				break
			}
		}
		if start < 0 {
			return s
		}
		end := -1
		for i := start + 1; i < len(rs); i++ {
			if rs[i] == bidiPDI {
				end = i
				break
			}
		}
		if end < 0 {
			s = string(append(append([]rune{}, rs[:start]...), rs[start+1:]...))
			continue
		}
		out := append(append([]rune{}, rs[:start]...), rs[start+1:end]...)
		out = append(out, rs[end+1:]...)
		s = string(out)
	}
}

func reverseRunes(rs []rune) []rune {
	out := make([]rune, len(rs))
	for i, r := range rs {
		out[len(rs)-1-i] = r
	}
	return out
}

func stripBidiControls(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == 0x200B || r == 0x200C || r == 0x200D || r == 0xFEFF || r == 0x2060 ||
			(r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
