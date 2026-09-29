// Package smtp 使用 github.com/wneessen/go-mail 连接本地/远程 MTA。
//
// 邮件正文仍由 email.Builder 产出完整 RFC822；本包只负责 SMTP 握手与 DATA，
// 避免 EML 再解析破坏自定义头 / BIDI / 乱序等反指纹细节。
package smtp

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/wneessen/go-mail"
	gosmtp "github.com/wneessen/go-mail/smtp"
)

// AuthConfig SMTP 认证配置
type AuthConfig struct {
	UseAuth  bool
	Username string
	Password string
}

func smtpSecurity(cfg PoolConfig) string {
	sec := strings.ToUpper(strings.TrimSpace(cfg.Security))
	switch sec {
	case "STARTTLS", "SSL", "SMTPS":
		return sec
	case "NONE", "NO", "OFF", "PLAIN":
		return ""
	}
	if cfg.UseTLS {
		return "STARTTLS"
	}
	if _, portStr, err := net.SplitHostPort(cfg.Addr); err == nil {
		if port, err := strconv.Atoi(portStr); err == nil && port == 465 {
			return "SSL"
		}
	}
	return ""
}

// newMailClient 根据池配置创建 go-mail Client（STARTTLS / 隐式 SSL / 明文 + AUTH / 超时）。
// 与 PHPMailer SMTPAutoTLS=false 对齐：未指定 STARTTLS/SSL 时强制 NoTLS，不做机会性升级。
func newMailClient(cfg PoolConfig) (*mail.Client, error) {
	host, portStr, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		host = cfg.Addr
		portStr = "25"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		port = 25
	}

	opts := []mail.Option{
		mail.WithTimeout(cfg.Timeout),
	}
	helo := strings.TrimSpace(cfg.HELO)
	if helo != "" {
		opts = append(opts, mail.WithHELO(helo))
	}

	sec := smtpSecurity(cfg)
	encrypted := sec == "STARTTLS" || sec == "SSL" || sec == "SMTPS"
	tlsCfg := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: cfg.SkipVerify,
		MinVersion:         tls.VersionTLS12,
	}
	// 先设 TLS 策略再 WithPort，避免 go-mail 把端口改回 587/25 默认值后覆盖我们的配置。
	switch sec {
	case "STARTTLS":
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory), mail.WithTLSConfig(tlsCfg))
	case "SSL", "SMTPS":
		opts = append(opts, mail.WithSSL(), mail.WithTLSConfig(tlsCfg))
	default:
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	}
	opts = append(opts, mail.WithPort(port))

	if cfg.Auth != nil && cfg.Auth.UseAuth && cfg.Auth.Username != "" {
		// AutoDiscover 在无 TLS 时不会选用 PLAIN/LOGIN（安全默认），本地 127.0.0.1:587 需 PLAIN-NOENC。
		authType := mail.SMTPAuthAutoDiscover
		if !encrypted {
			authType = mail.SMTPAuthPlainNoEnc
		}
		opts = append(opts,
			mail.WithSMTPAuth(authType),
			mail.WithUsername(cfg.Auth.Username),
			mail.WithPassword(cfg.Auth.Password),
		)
	}

	return mail.NewClient(host, opts...)
}

// GreetingLooksLikeHaraka 读 220 欢迎语，判断对面是不是 Haraka（YAML 写成 pmta 时也能补 AUTH）。
func GreetingLooksLikeHaraka(addr string, timeout time.Duration) bool {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(line), "haraka")
}

func dialSMTPClient(c *mail.Client, timeout time.Duration) (*gosmtp.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	sc, err := c.DialToSMTPClientWithContext(ctx)
	if err != nil {
		return nil, &SendError{Stage: StageConnect, Err: fmt.Errorf("go-mail dial: %w", err)}
	}
	return sc, nil
}

func closeSMTPClient(sc *gosmtp.Client) {
	if sc == nil {
		return
	}
	_ = sc.Quit()
	_ = sc.Close()
}

func recycleSMTPClient(sc *gosmtp.Client) *gosmtp.Client {
	if sc == nil {
		return nil
	}
	if err := sc.Reset(); err != nil {
		_ = sc.Close()
		return nil
	}
	return sc
}

// sendOnSMTPClient 在已拨号的连接上提交 RAW（不 QUIT，供 KeepAlive 复用）。
func sendOnSMTPClient(sc *gosmtp.Client, from, to string, raw []byte) error {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return &SendError{Stage: StageMail, Message: "envelope from/to 为空"}
	}

	// 禁止 sc.Mail()：go-mail/net/smtp 在 EHLO 宣告扩展时会追加
	// BODY=8BITMIME SMTPUTF8。PowerMTA 收到 SMTPUTF8 后会把整封信标成
	// 「需要 SMTPUTF8」，au/docomo 等只宣告 8BITMIME 的 MX 会 5.6.7 退信。
	if err := smtpMailFromPlain(sc, from); err != nil {
		return wrapSMTPStage(StageMail, err)
	}
	if err := sc.Rcpt(to); err != nil {
		return wrapSMTPStage(StageRcpt, err)
	}

	w, err := sc.Data()
	if err != nil {
		return wrapSMTPStage(StageData, err)
	}
	payload := ensureCRLF(raw)
	if _, err := w.Write(payload); err != nil {
		_ = w.Close()
		return &SendError{Stage: StageData, Err: fmt.Errorf("写入正文: %w", err)}
	}
	// Close 发送结束点并读最终应答；断线 → 结果未知
	if err := w.Close(); err != nil {
		if isRetryableNetErr(err) || isConnClosedErr(err) {
			return newUnknownDotError(err)
		}
		return wrapSMTPStage(StageDot, err)
	}
	return nil
}

// sendRawViaGoMail 拨号、提交 RAW 后立即断开（非 KeepAlive）。
func sendRawViaGoMail(c *mail.Client, timeout time.Duration, from, to string, raw []byte) error {
	sc, err := dialSMTPClient(c, timeout)
	if err != nil {
		return err
	}
	err = sendOnSMTPClient(sc, from, to, raw)
	if err != nil {
		_ = sc.Close()
		return err
	}
	closeSMTPClient(sc)
	return nil
}

func wrapSMTPStage(stage Stage, err error) *SendError {
	if err == nil {
		return nil
	}
	var te *textproto.Error
	if errors.As(err, &te) {
		return &SendError{Stage: stage, Code: te.Code, Message: te.Msg, Err: err}
	}
	// go-mail / net/smtp 有时包装成 "550 ..." 字符串
	code, msg := parseSMTPCodeMessage(err.Error())
	return &SendError{Stage: stage, Code: code, Message: msg, Err: err}
}

func parseSMTPCodeMessage(s string) (int, string) {
	s = strings.TrimSpace(s)
	if len(s) >= 3 {
		if c, err := strconv.Atoi(s[:3]); err == nil && c >= 200 && c < 600 {
			msg := strings.TrimSpace(s[3:])
			msg = strings.TrimLeft(msg, "- ")
			return c, msg
		}
	}
	// 在整段文本中找 4xx/5xx
	for i := 0; i+2 < len(s); i++ {
		if s[i] >= '4' && s[i] <= '5' && s[i+1] >= '0' && s[i+1] <= '9' && s[i+2] >= '0' && s[i+2] <= '9' {
			if i > 0 {
				prev := s[i-1]
				if (prev >= '0' && prev <= '9') || (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z') {
					continue
				}
			}
			c, _ := strconv.Atoi(s[i : i+3])
			return c, strings.TrimSpace(s)
		}
	}
	return 0, s
}

func isConnClosedErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, key := range []string{"eof", "connection reset", "broken pipe", "use of closed"} {
		if strings.Contains(msg, key) {
			return true
		}
	}
	return false
}

// smtpMailFromPlain 发送不含 BODY= / SMTPUTF8 参数的 MAIL FROM。
func smtpMailFromPlain(sc *gosmtp.Client, from string) error {
	if sc == nil || sc.Text == nil {
		return fmt.Errorf("smtp client not ready")
	}
	id, err := sc.Text.Cmd("MAIL FROM:<%s>", from)
	if err != nil {
		return err
	}
	sc.Text.StartResponse(id)
	defer sc.Text.EndResponse(id)
	_, _, err = sc.Text.ReadResponse(250)
	return err
}

func ensureCRLF(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	s := string(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")
	return []byte(s)
}
