package email

import "strings"

// mtaLeakHeaderNames 对齐 PMTA <source 0/0> remove-header / retain-x-* no，
// 以及 Haraka pmta_header_align.js。半成品不写这些头，避免和邮局收尾打架。
var mtaLeakHeaderNames = map[string]struct{}{
	"received":                 {},
	"return-path":              {},
	"dkim-signature":           {},
	"x-virtual-mta":            {},
	"x-job":                    {},
	"x-originating-ip":         {},
	"x-php-script":             {},
	"x-php-originating-script": {},
	"user-agent":               {},
	"x-msmail-priority":        {},
	"x-mimeole":                {},
	"x-sender":                 {},
	"x-antiabuse":              {},
	"x-source":                 {},
	"x-source-args":            {},
	"x-source-dir":             {},
	"x-haraka":                 {},
	"x-haraka-uuid":            {},
	"x-haraka-transaction":     {},
	"x-sending-zone":           {},
	"x-zonemta-queue-id":       {},
}

func IsMTALeakHeader(name string) bool {
	_, ok := mtaLeakHeaderNames[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func headerFieldName(line string) string {
	i := strings.IndexByte(line, ':')
	if i <= 0 {
		return ""
	}
	return strings.TrimSpace(line[:i])
}
