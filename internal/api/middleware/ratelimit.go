package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter はキー（クライアントIP等）単位の許可判定を提供する。
// 将来的に Redis 等の分散実装に差し替えられるようインタフェースとして公開する。
type RateLimiter interface {
	Allow(key string) bool
	// RetryAfter は次に許可される推定時刻までの秒数を返す（0以下なら即時許可可能）。
	RetryAfter(key string) int
}

// inMemoryLimiter は IP ごとの *rate.Limiter を sync.Map で保持するインメモリ実装。
type inMemoryLimiter struct {
	rps          rate.Limit
	burst        int
	limiters     sync.Map // map[string]*limiterEntry
	cleanupEvery time.Duration
	idleTimeout  time.Duration

	stopOnce sync.Once
	stopCh   chan struct{}
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen atomic_time
}

// atomic_time は time.Time の sync.Mutex 保護版。
type atomic_time struct {
	mu sync.Mutex
	t  time.Time
}

func (a *atomic_time) Set(t time.Time) {
	a.mu.Lock()
	a.t = t
	a.mu.Unlock()
}

func (a *atomic_time) Get() time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.t
}

// NewInMemoryRateLimiter は分あたりリクエスト数とバースト値からインメモリレートリミッタを生成する。
func NewInMemoryRateLimiter(requestsPerMinute, burst int) RateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60
	}
	if burst <= 0 {
		burst = 1
	}
	rl := &inMemoryLimiter{
		rps:          rate.Limit(float64(requestsPerMinute) / 60.0),
		burst:        burst,
		cleanupEvery: 5 * time.Minute,
		idleTimeout:  10 * time.Minute,
		stopCh:       make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (l *inMemoryLimiter) get(key string) *limiterEntry {
	if v, ok := l.limiters.Load(key); ok {
		entry := v.(*limiterEntry)
		entry.lastSeen.Set(time.Now())
		return entry
	}
	entry := &limiterEntry{limiter: rate.NewLimiter(l.rps, l.burst)}
	entry.lastSeen.Set(time.Now())
	actual, _ := l.limiters.LoadOrStore(key, entry)
	return actual.(*limiterEntry)
}

// Allow はキーに対するリクエストを許可するかを返す。
func (l *inMemoryLimiter) Allow(key string) bool {
	return l.get(key).limiter.Allow()
}

// RetryAfter は次に許可されるまでの推定秒数を返す。
func (l *inMemoryLimiter) RetryAfter(key string) int {
	entry := l.get(key)
	reservation := entry.limiter.Reserve()
	defer reservation.Cancel()
	if !reservation.OK() {
		return 1
	}
	delay := reservation.Delay()
	return max(int(delay.Round(time.Second).Seconds()), 1)
}

// Stop はクリーンアップループを停止する（テストや graceful shutdown 用）。
func (l *inMemoryLimiter) Stop() {
	l.stopOnce.Do(func() { close(l.stopCh) })
}

func (l *inMemoryLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanupEvery)
	defer ticker.Stop()
	for {
		select {
		case <-l.stopCh:
			return
		case now := <-ticker.C:
			l.cleanup(now)
		}
	}
}

func (l *inMemoryLimiter) cleanup(now time.Time) {
	l.limiters.Range(func(k, v any) bool {
		entry := v.(*limiterEntry)
		if now.Sub(entry.lastSeen.Get()) > l.idleTimeout {
			l.limiters.Delete(k)
		}
		return true
	})
}

// RateLimitOption はレートリミットミドルウェアのオプション。
type RateLimitOption func(*rateLimitConfig)

type rateLimitConfig struct {
	exemptPaths map[string]struct{}
}

// WithExemptPaths は指定パスをレートリミットの対象外にする。
func WithExemptPaths(paths ...string) RateLimitOption {
	return func(c *rateLimitConfig) {
		if c.exemptPaths == nil {
			c.exemptPaths = make(map[string]struct{}, len(paths))
		}
		for _, p := range paths {
			c.exemptPaths[p] = struct{}{}
		}
	}
}

// NewRateLimit はクライアントIP単位のレートリミットミドルウェアを返す。
// OPTIONS（CORSプリフライト）と除外パスはカウント対象外。
// 制限超過時は429+Retry-Afterヘッダーを付けてJSONで応答する。
func NewRateLimit(limiter RateLimiter, opts ...RateLimitOption) func(http.Handler) http.Handler {
	cfg := &rateLimitConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			if _, exempt := cfg.exemptPaths[r.URL.Path]; exempt {
				next.ServeHTTP(w, r)
				return
			}

			ip := clientIP(r)
			if !limiter.Allow(ip) {
				retry := limiter.RetryAfter(ip)
				writeTooManyRequests(w, retry)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeTooManyRequests(w http.ResponseWriter, retryAfterSec int) {
	retryAfterSec = max(retryAfterSec, 1)
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSec))
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusTooManyRequests)
	resp := map[string]any{
		"$schema":     "https://example.com/errors/too-many-requests.json",
		"title":       "Too Many Requests",
		"status":      http.StatusTooManyRequests,
		"detail":      "リクエストレートの上限に達しました。しばらく待ってから再試行してください。",
		"retry_after": retryAfterSec,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
