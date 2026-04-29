package middleware

import (
	"net"
	"net/http"
	"strings"
)

// clientIP はリクエストからクライアントIPを抽出する。
// X-Forwarded-For が存在する場合は最左の値、不在の場合は r.RemoteAddr を使う。
// Cloud Run はGoogleフロントエンドが X-Forwarded-For を上書きするため、最左を信頼してよい。
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// 最左のIP（外部からの本来のクライアントIP）。
		if idx := strings.IndexByte(xff, ','); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
