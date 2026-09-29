package smtp

import (
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"strings"
)

// Stage 标识 SMTP 事务阶段，用于分类重试策略。
type Stage string

const (
	StageConnect Stage = "connect"
	StageMail    Stage = "mail"
	StageRcpt    Stage = "rcpt"
	StageData    Stage = "data" // DATA 命令（354）或写入正文过程
	StageDot     Stage = "dot"  // 已发送结束点，等待最终 250
)

// SendError 带阶段/响应码的发送错误。
type SendError struct {
	Stage         Stage
	Code          int
	Message       string
	Err           error
	UnknownResult bool // DATA 已结束但未收到最终应答：可能已受理，禁止立即重注
}

func (e *SendError) Error() string {
	if e == nil {
		return ""
	}
	prefix := ""
	if e.UnknownResult {
		prefix = "结果未知: "
	}
	if e.Code > 0 {
		if e.Message != "" {
			return fmt.Sprintf("%sSMTP %s [%d]: %s", prefix, e.Stage, e.Code, e.Message)
		}
		return fmt.Sprintf("%sSMTP %s [%d]", prefix, e.Stage, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("%sSMTP %s: %v", prefix, e.Stage, e.Err)
	}
	if e.Message != "" {
		return fmt.Sprintf("%sSMTP %s: %s", prefix, e.Stage, e.Message)
	}
	return fmt.Sprintf("%sSMTP %s 失败", prefix, e.Stage)
}

func (e *SendError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Temporary 4xx：可延迟重试。
func (e *SendError) Temporary() bool {
	return e != nil && e.Code >= 400 && e.Code < 500
}

// Permanent 5xx：当前投递不再自动重试。
func (e *SendError) Permanent() bool {
	return e != nil && e.Code >= 500 && e.Code < 600
}

// InvalidRecipient 仅 RCPT 阶段、明确表示邮箱不存在/不可用的 5xx。
// 不把 552（配额）、554（策略/拒收）、535（认证）、MAIL FROM 5xx 等一律当无效地址。
func (e *SendError) InvalidRecipient() bool {
	if e == nil || e.Stage != StageRcpt || !e.Permanent() {
		return false
	}
	switch e.Code {
	case 550, 551, 553:
		return true
	default:
		return false
	}
}

// ShouldRetry 是否应对当前投递自动重试。
func (e *SendError) ShouldRetry() bool {
	if e == nil || e.UnknownResult {
		return false
	}
	if e.Permanent() {
		return false
	}
	if e.Temporary() {
		return true
	}
	// 无 SMTP 码的网络/超时：仅在 DATA 结束前可重试
	if e.Stage == StageDot {
		return false
	}
	return isRetryableNetErr(e.Err) || isRetryableNetErr(e)
}

func isRetryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, key := range []string{
		"connection reset", "broken pipe", "i/o timeout", "use of closed network",
		"connection refused", "eof", "wsarecv", "wsasend",
	} {
		if strings.Contains(msg, key) {
			return true
		}
	}
	return false
}

func newSMTPError(stage Stage, code int, lines []string, err error) *SendError {
	msg := ""
	if len(lines) > 0 {
		msg = strings.TrimSpace(lines[0])
		if len(msg) > 3 && msg[0] >= '0' && msg[0] <= '9' {
			// "550 mailbox unavailable" → keep as-is
		}
	}
	if msg == "" && err != nil {
		msg = err.Error()
		var te *textproto.Error
		if errors.As(err, &te) {
			code = te.Code
			msg = te.Msg
		}
	}
	return &SendError{Stage: stage, Code: code, Message: msg, Err: err}
}

func newUnknownDotError(err error) *SendError {
	return &SendError{
		Stage:         StageDot,
		Message:       "DATA 已结束，等待最终应答时连接中断",
		Err:           err,
		UnknownResult: true,
	}
}

// IsUnknownResult DATA 结束后结果未知。
func (e *SendError) IsUnknownResult() bool {
	return e != nil && e.UnknownResult
}

// AsSendError 提取 *SendError。
func AsSendError(err error) (*SendError, bool) {
	var se *SendError
	if errors.As(err, &se) {
		return se, true
	}
	return nil, false
}
