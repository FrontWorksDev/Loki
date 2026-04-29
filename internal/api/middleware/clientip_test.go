package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{name: "remote_addr_with_port", remoteAddr: "1.2.3.4:1234", want: "1.2.3.4"},
		{name: "xff_single", xff: "203.0.113.10", remoteAddr: "10.0.0.1:1234", want: "203.0.113.10"},
		{name: "xff_multi_takes_leftmost", xff: "203.0.113.10, 10.0.0.1, 192.168.1.1", remoteAddr: "10.0.0.1:1234", want: "203.0.113.10"},
		{name: "xff_with_spaces_trimmed", xff: "  203.0.113.10  ,  10.0.0.1  ", remoteAddr: "10.0.0.1:1234", want: "203.0.113.10"},
		{name: "remote_addr_without_port", remoteAddr: "1.2.3.4", want: "1.2.3.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if got := clientIP(req); got != tt.want {
				t.Errorf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
