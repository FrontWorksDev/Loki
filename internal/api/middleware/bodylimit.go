package middleware

import (
	"encoding/json"
	"net/http"
)

// BodyLimitOption はボディサイズ制限ミドルウェアのオプション。
type BodyLimitOption func(*bodyLimitConfig)

type bodyLimitConfig struct {
	maxBytes    int64
	exemptPaths map[string]struct{}
}

// WithBodyLimitExemptPaths は指定パスをボディサイズ制限の対象外にする。
func WithBodyLimitExemptPaths(paths ...string) BodyLimitOption {
	return func(c *bodyLimitConfig) {
		if c.exemptPaths == nil {
			c.exemptPaths = make(map[string]struct{}, len(paths))
		}
		for _, p := range paths {
			c.exemptPaths[p] = struct{}{}
		}
	}
}

// NewBodyLimit はリクエストボディサイズの上限を強制するミドルウェアを返す。
// 上限超過時は 413 Payload Too Large をJSONで返す。
//
// Content-Length が信頼できる場合は事前判定で 413 を返してハンドラに到達させない。
// chunked transfer 等で Content-Length が不明な場合は MaxBytesReader が読み取り中に
// エラーを返すため、下流ハンドラ（典型的には Huma）側でエラー応答に変換される。
// Loki では各 operation にも `MaxBodyBytes` を宣言しているので二段防御となる。
func NewBodyLimit(maxBytes int64, opts ...BodyLimitOption) func(http.Handler) http.Handler {
	cfg := &bodyLimitConfig{maxBytes: maxBytes}
	for _, opt := range opts {
		opt(cfg)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, exempt := cfg.exemptPaths[r.URL.Path]; exempt || cfg.maxBytes <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Content-Length が分かる場合は事前判定で早期リターン。
			if r.ContentLength > cfg.maxBytes {
				writePayloadTooLarge(w, cfg.maxBytes)
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, cfg.maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// writePayloadTooLarge は413レスポンスをHuma互換のRFC 7807スタイルJSONで返す。
func writePayloadTooLarge(w http.ResponseWriter, maxBytes int64) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	resp := map[string]any{
		"$schema":   "https://example.com/errors/payload-too-large.json",
		"title":     "Payload Too Large",
		"status":    http.StatusRequestEntityTooLarge,
		"detail":    "リクエストボディが上限を超えています",
		"max_bytes": maxBytes,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
