package middleware

import (
	"encoding/json"
	"errors"
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
			lw := &limitDetectingWriter{ResponseWriter: w}
			next.ServeHTTP(lw, r)

			// ハンドラ内で MaxBytesReader が消費中にエラーを起こしていた場合は
			// すでにレスポンスが書かれている可能性が高いため、ここでは何もしない。
			_ = lw
		})
	}
}

// limitDetectingWriter は将来 MaxBytesError を捕捉する用途のためのフック。
// 現状はパススルー。
type limitDetectingWriter struct {
	http.ResponseWriter
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

// IsMaxBytesError は MaxBytesReader 由来のエラーか判定するヘルパー。
func IsMaxBytesError(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}
