// Package middleware はAPIサーバー用のChi互換ミドルウェアを提供する。
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// NewLogging は構造化ログを出力するミドルウェアを返す。
// log/slog のJSONハンドラ前提で、リクエストごとに1行のJSONログを出力する。
// status >= 500 の場合はerrorレベル、4xx は warnレベル、それ以外はinfoレベルで出力する。
func NewLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &responseWriterWrapper{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			requestID := chimw.GetReqID(r.Context())
			if requestID != "" {
				ww.Header().Set("X-Request-Id", requestID)
			}

			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.status),
				slog.Int64("duration_ms", duration.Milliseconds()),
				slog.String("remote_ip", clientIP(r)),
				slog.Int64("bytes_out", ww.bytesWritten),
				slog.String("user_agent", r.UserAgent()),
				slog.String("request_id", requestID),
			}

			level := levelForStatus(ww.status)
			logger.LogAttrs(r.Context(), level, "http_request", toAttrSlice(attrs)...)
		})
	}
}

// responseWriterWrapper は http.ResponseWriter をラップしてステータスコードと
// レスポンスバイト数を記録する。レスポンスボディ自体は保存しない（画像バイナリの
// 転送性能を維持するため）。
type responseWriterWrapper struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
	wroteHeader  bool
}

func (w *responseWriterWrapper) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}

// Flush は http.Flusher の実装をパススルーする（ストリーミング応答対応）。
func (w *responseWriterWrapper) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func levelForStatus(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

func toAttrSlice(items []any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(items))
	for _, it := range items {
		if a, ok := it.(slog.Attr); ok {
			attrs = append(attrs, a)
		}
	}
	return attrs
}
