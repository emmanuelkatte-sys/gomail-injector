package smtp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wneessen/go-mail"
	gosmtp "github.com/wneessen/go-mail/smtp"
)

// PoolConfig 连接池配置
type PoolConfig struct {
	Addr           string
	Size           int
	Timeout        time.Duration
	UseTLS         bool
	Security       string // STARTTLS | SSL | SMTPS | 空=明文
	SkipVerify     bool
	HELO           string
	KeepAlive      bool // max_conn > 0 时复用 SMTP 连接；失败则关掉重建
	MaxSendPerConn int  // 保留兼容
	Auth           *AuthConfig
}

// Pool 基于 go-mail Client 的并发限流发送池。
type Pool struct {
	config PoolConfig
	client *mail.Client

	sem    chan struct{}
	idle   chan *gosmtp.Client
	mu     sync.Mutex
	closed int32

	created   int64
	failed    int64
	reused    int64
	totalSent int64
}

// NewPool 创建 go-mail SMTP 池（预拨一次验证连通性）。
func NewPool(config PoolConfig) (*Pool, error) {
	if config.Size <= 0 {
		config.Size = 10
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	mc, err := newMailClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建 go-mail Client 失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	if err := mc.DialWithContext(ctx); err != nil {
		_ = mc.Close()
		return nil, fmt.Errorf("无法连接到 SMTP 服务器 %s: %w", config.Addr, err)
	}
	_ = mc.Close()

	// Dial 后需可再次使用：重建 Client（go-mail Close 后不宜复用）
	mc, err = newMailClient(config)
	if err != nil {
		return nil, fmt.Errorf("重建 go-mail Client 失败: %w", err)
	}

	idleSize := config.Size
	if idleSize <= 0 {
		idleSize = 1
	}
	pool := &Pool{
		config: config,
		client: mc,
		sem:    make(chan struct{}, config.Size),
		idle:   make(chan *gosmtp.Client, idleSize),
	}
	atomic.AddInt64(&pool.created, 1)
	return pool, nil
}

// Send 发送一封原始邮件
func (p *Pool) Send(from, to string, msg []byte) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		return &SendError{Stage: StageConnect, Message: "连接池已关闭"}
	}

	select {
	case p.sem <- struct{}{}:
	case <-time.After(p.config.Timeout):
		return &SendError{Stage: StageConnect, Message: "获取发送槽位超时"}
	}
	defer func() { <-p.sem }()

	err := p.sendOnce(from, to, msg)
	if err != nil {
		atomic.AddInt64(&p.failed, 1)
		p.rebuildClient()
		return err
	}
	atomic.AddInt64(&p.totalSent, 1)
	return nil
}

func (p *Pool) currentClient() *mail.Client {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.client
}

func (p *Pool) rebuildClient() {
	mc, e2 := newMailClient(p.config)
	if e2 != nil {
		return
	}
	p.mu.Lock()
	old := p.client
	p.client = mc
	p.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	atomic.AddInt64(&p.created, 1)
}

func (p *Pool) takeIdle() *gosmtp.Client {
	if !p.config.KeepAlive || p.idle == nil {
		return nil
	}
	select {
	case sc := <-p.idle:
		return sc
	default:
		return nil
	}
}

func (p *Pool) putIdle(sc *gosmtp.Client) {
	if sc == nil {
		return
	}
	if !p.config.KeepAlive || atomic.LoadInt32(&p.closed) == 1 || p.idle == nil {
		closeSMTPClient(sc)
		return
	}
	sc = recycleSMTPClient(sc)
	if sc == nil {
		return
	}
	select {
	case p.idle <- sc:
	default:
		closeSMTPClient(sc)
	}
}

func (p *Pool) sendOnce(from, to string, msg []byte) error {
	client := p.currentClient()
	sc := p.takeIdle()
	reused := sc != nil
	if sc == nil {
		var err error
		sc, err = dialSMTPClient(client, p.config.Timeout)
		if err != nil {
			return err
		}
	}

	err := sendOnSMTPClient(sc, from, to, msg)
	if err != nil {
		_ = sc.Close()
		sc = nil
		if reused {
			if se, ok := AsSendError(err); ok && (se.UnknownResult || !se.ShouldRetry()) {
				return err
			}
			fresh, err2 := dialSMTPClient(p.currentClient(), p.config.Timeout)
			if err2 != nil {
				return err
			}
			err = sendOnSMTPClient(fresh, from, to, msg)
			if err != nil {
				_ = fresh.Close()
				return err
			}
			sc = fresh
			reused = false
		} else {
			return err
		}
	}

	if reused {
		atomic.AddInt64(&p.reused, 1)
	}
	if p.config.KeepAlive {
		p.putIdle(sc)
	} else {
		closeSMTPClient(sc)
	}
	return nil
}

// SendWithRetry 按 SMTP 阶段/码重试：4xx 延迟重试；5xx 不重试；
// DATA 结束后断线（结果未知）不重试，避免重复注入。
func (p *Pool) SendWithRetry(from, to string, msg []byte, maxRetries int) error {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		err := p.Send(from, to, msg)
		if err == nil {
			return nil
		}
		lastErr = err

		se, ok := AsSendError(err)
		if ok {
			if se.UnknownResult || !se.ShouldRetry() {
				return se
			}
		} else if i == maxRetries {
			break
		} else if !isRetryableNetErr(err) {
			return err
		}

		if i == maxRetries {
			break
		}
		time.Sleep(retryDelay4xx(i))
	}
	if se, ok := AsSendError(lastErr); ok {
		return se
	}
	return fmt.Errorf("发送失败（重试%d次）: %w", maxRetries, lastErr)
}

func retryDelay4xx(attempt int) time.Duration {
	base := 200 * time.Millisecond
	d := base << attempt
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// Close 关闭连接池
func (p *Pool) Close() {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.drainIdle()
	if p.client != nil {
		_ = p.client.Close()
		p.client = nil
	}
}

func (p *Pool) drainIdle() {
	if p.idle == nil {
		return
	}
	for {
		select {
		case sc := <-p.idle:
			closeSMTPClient(sc)
		default:
			return
		}
	}
}

// Stats 统计信息
func (p *Pool) Stats() PoolStats {
	inUse := len(p.sem)
	return PoolStats{
		Active:    inUse,
		Available: p.config.Size - inUse,
		Size:      p.config.Size,
		Created:   atomic.LoadInt64(&p.created),
		Failed:    atomic.LoadInt64(&p.failed),
		Reused:    atomic.LoadInt64(&p.reused),
		TotalSent: atomic.LoadInt64(&p.totalSent),
		Closed:    atomic.LoadInt32(&p.closed) == 1,
	}
}

// PoolStats 连接池统计信息
type PoolStats struct {
	Active    int
	Available int
	Size      int
	Created   int64
	Failed    int64
	Reused    int64
	TotalSent int64
	Closed    bool
}

func (p *Pool) IsClosed() bool   { return atomic.LoadInt32(&p.closed) == 1 }
func (p *Pool) Size() int        { return p.config.Size }
func (p *Pool) ActiveCount() int { return len(p.sem) }
func (p *Pool) TotalSent() int64 { return atomic.LoadInt64(&p.totalSent) }
