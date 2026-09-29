package utils

import (
	"strings"
	"unicode"
)

// BuildEnvelopeVERP 将收件人编码进信封 MAIL FROM（经典 plus-VERP）。
// 例：base@mail.example.com + user@foo.co.jp
//   → base+user=foo.co.jp@mail.example.com
// 域名保持不变，便于 SPF 对齐；禁止用「最后两段」改域。
func BuildEnvelopeVERP(envelopeFrom, recipientEmail string) string {
	envelopeFrom = strings.TrimSpace(envelopeFrom)
	recipientEmail = strings.TrimSpace(strings.ToLower(recipientEmail))
	if envelopeFrom == "" || recipientEmail == "" || !strings.Contains(recipientEmail, "@") {
		return envelopeFrom
	}
	at := strings.LastIndex(envelopeFrom, "@")
	if at <= 0 || at >= len(envelopeFrom)-1 {
		return envelopeFrom
	}
	local := envelopeFrom[:at]
	domain := envelopeFrom[at+1:]
	encoded := sanitizeVERPLocal(strings.ReplaceAll(recipientEmail, "@", "="))
	if encoded == "" {
		return envelopeFrom
	}
	return local + "+" + encoded + "@" + domain
}

func sanitizeVERPLocal(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		case r == '.' || r == '_' || r == '-' || r == '=' || r == '+':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
