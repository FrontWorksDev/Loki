package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeLimiter は呼び出しキー単位の許可を制御するためのテスト用 RateLimiter。
type fakeLimiter struct {
	allow map[string]bool
	calls atomic.Int64
}

func (f *fakeLimiter) Allow(key string) bool {
	f.calls.Add(1)
	v, ok := f.allow[key]
	if !ok {
		return true
	}
	return v
}

func (f *fakeLimiter) RetryAfter(string) int { return 1 }
func (f *fakeLimiter) Close() error          { return nil }

func TestRateLimit_Behaviors(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("under_limit_passes", func(t *testing.T) {
		fl := &fakeLimiter{allow: map[string]bool{"1.2.3.4": true}}
		h := NewRateLimit(fl)(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rr.Code)
		}
	})

	t.Run("over_limit_returns_429_with_retry_after", func(t *testing.T) {
		fl := &fakeLimiter{allow: map[string]bool{"1.2.3.4": false}}
		h := NewRateLimit(fl)(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("status = %d, want 429", rr.Code)
		}
		if rr.Header().Get("Retry-After") == "" {
			t.Error("Retry-After missing")
		}
	})

	t.Run("options_method_exempt", func(t *testing.T) {
		fl := &fakeLimiter{allow: map[string]bool{"1.2.3.4": false}}
		h := NewRateLimit(fl)(okHandler)

		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("OPTIONS status = %d, want 200 (exempt)", rr.Code)
		}
		if got := fl.calls.Load(); got != 0 {
			t.Errorf("limiter called %d times for OPTIONS, want 0", got)
		}
	})

	t.Run("exempt_path_not_counted", func(t *testing.T) {
		fl := &fakeLimiter{allow: map[string]bool{"1.2.3.4": false}}
		h := NewRateLimit(fl, WithExemptPaths("/api/v1/health"))(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rr.Code)
		}
	})

	t.Run("xff_used_as_key", func(t *testing.T) {
		fl := &fakeLimiter{allow: map[string]bool{"203.0.113.10": false, "10.0.0.1": true}}
		h := NewRateLimit(fl)(okHandler)

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		req.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		// XFF最左の 203.0.113.10 がキーで、許可マップでは false なので 429。
		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("status = %d, want 429 (XFF leftmost should be the key)", rr.Code)
		}
	})

	t.Run("different_ips_independent", func(t *testing.T) {
		fl := &fakeLimiter{allow: map[string]bool{"1.1.1.1": false, "2.2.2.2": true}}
		h := NewRateLimit(fl)(okHandler)

		makeReq := func(ip string) *httptest.ResponseRecorder {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			req.RemoteAddr = ip + ":1234"
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			return rr
		}

		if got := makeReq("1.1.1.1").Code; got != http.StatusTooManyRequests {
			t.Errorf("ip 1.1.1.1: %d, want 429", got)
		}
		if got := makeReq("2.2.2.2").Code; got != http.StatusOK {
			t.Errorf("ip 2.2.2.2: %d, want 200", got)
		}
	})
}

func TestInMemoryLimiter_BasicAllowAndDeny(t *testing.T) {
	// 60 req/min == 1 req/sec, burst 1 → 連続で2回呼ぶと2回目は拒否されるはず。
	rl := NewInMemoryRateLimiter(60, 1)
	defer rl.(*inMemoryLimiter).Stop()

	if !rl.Allow("ip-A") {
		t.Fatal("first call should be allowed")
	}
	if rl.Allow("ip-A") {
		t.Error("second call should be denied (burst=1, no token refilled)")
	}
	// 別IPは独立。
	if !rl.Allow("ip-B") {
		t.Error("different ip should be allowed independently")
	}
}

func TestInMemoryLimiter_RetryAfter(t *testing.T) {
	rl := NewInMemoryRateLimiter(60, 1)
	defer rl.(*inMemoryLimiter).Stop()

	// 1個のトークンを消費。
	if !rl.Allow("ip-A") {
		t.Fatal("first call should be allowed")
	}
	// 2件目はトークンがない状態 → RetryAfter は1秒以上を返す。
	if got := rl.RetryAfter("ip-A"); got < 1 {
		t.Errorf("RetryAfter = %d, want >= 1", got)
	}
}

func TestInMemoryLimiter_DefaultsForInvalidArgs(t *testing.T) {
	// 0/負値が渡された場合のフォールバック動作を確認。
	rl := NewInMemoryRateLimiter(0, 0)
	defer rl.(*inMemoryLimiter).Stop()

	// burst >= 1 にフォールバックされていれば最初の Allow は true。
	if !rl.Allow("ip-X") {
		t.Error("invalid args should fall back to working defaults; first Allow expected to succeed")
	}
}

func TestInMemoryLimiter_Cleanup(t *testing.T) {
	rl := NewInMemoryRateLimiter(60, 1).(*inMemoryLimiter)
	defer rl.Stop()

	rl.Allow("stale-ip")
	if _, ok := rl.limiters.Load("stale-ip"); !ok {
		t.Fatal("stale-ip not registered")
	}

	// idleTimeout を超過した時刻でクリーンアップ実行。
	future := time.Now().Add(rl.idleTimeout + time.Minute)
	rl.cleanup(future)

	if _, ok := rl.limiters.Load("stale-ip"); ok {
		t.Error("stale-ip should have been cleaned up")
	}
}
