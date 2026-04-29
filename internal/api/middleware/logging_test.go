package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"
)

func TestLogging(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		method         string
		path           string
		remoteAddr     string
		xForwardedFor  string
		userAgent      string
		wantLevel      string
		wantStatus     int
		wantBytes      int64
		wantRemoteIP   string
		wantPathLogged string
	}{
		{
			name: "200_OK_info_level",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("hello"))
			},
			method:         http.MethodGet,
			path:           "/api/v1/health",
			remoteAddr:     "1.2.3.4:5678",
			userAgent:      "test-agent/1.0",
			wantLevel:      "INFO",
			wantStatus:     200,
			wantBytes:      5,
			wantRemoteIP:   "1.2.3.4",
			wantPathLogged: "/api/v1/health",
		},
		{
			name: "500_error_level",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"err":"x"}`))
			},
			method:         http.MethodPost,
			path:           "/api/v1/compress",
			remoteAddr:     "10.0.0.1:1111",
			userAgent:      "ua",
			wantLevel:      "ERROR",
			wantStatus:     500,
			wantBytes:      11,
			wantRemoteIP:   "10.0.0.1",
			wantPathLogged: "/api/v1/compress",
		},
		{
			name: "404_warn_level",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			method:       http.MethodGet,
			path:         "/missing",
			remoteAddr:   "127.0.0.1:9999",
			wantLevel:    "WARN",
			wantStatus:   404,
			wantBytes:    0,
			wantRemoteIP: "127.0.0.1",
		},
		{
			name: "x_forwarded_for_takes_precedence",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			method:        http.MethodGet,
			path:          "/x",
			remoteAddr:    "127.0.0.1:9999",
			xForwardedFor: "203.0.113.10, 10.0.0.1",
			wantLevel:     "INFO",
			wantStatus:    200,
			wantRemoteIP:  "203.0.113.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			// RequestID も流して伝播を確認する。
			handler := chimw.RequestID(NewLogging(logger)(tt.handler))

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			line := strings.TrimSpace(buf.String())
			if line == "" {
				t.Fatal("expected log line, got empty")
			}
			var entry map[string]any
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				t.Fatalf("invalid json log: %v\n%s", err, line)
			}

			if got := entry["level"]; got != tt.wantLevel {
				t.Errorf("level = %v, want %v", got, tt.wantLevel)
			}
			if got := entry["msg"]; got != "http_request" {
				t.Errorf("msg = %v, want http_request", got)
			}
			if got := entry["method"]; got != tt.method {
				t.Errorf("method = %v, want %v", got, tt.method)
			}
			if tt.wantPathLogged != "" {
				if got := entry["path"]; got != tt.wantPathLogged {
					t.Errorf("path = %v, want %v", got, tt.wantPathLogged)
				}
			}
			if got := int(entry["status"].(float64)); got != tt.wantStatus {
				t.Errorf("status = %v, want %v", got, tt.wantStatus)
			}
			if got := int64(entry["bytes_out"].(float64)); got != tt.wantBytes {
				t.Errorf("bytes_out = %v, want %v", got, tt.wantBytes)
			}
			if got := entry["remote_ip"]; got != tt.wantRemoteIP {
				t.Errorf("remote_ip = %v, want %v", got, tt.wantRemoteIP)
			}
			if _, ok := entry["duration_ms"]; !ok {
				t.Error("duration_ms missing")
			}
			rid, _ := entry["request_id"].(string)
			if rid == "" {
				t.Error("request_id is empty")
			}
			if got := rr.Header().Get("X-Request-Id"); got != rid {
				t.Errorf("X-Request-Id header = %q, want %q", got, rid)
			}
		})
	}
}

func TestResponseWriterWrapper_Flush(t *testing.T) {
	rec := httptest.NewRecorder()
	w := &responseWriterWrapper{ResponseWriter: rec, status: http.StatusOK}
	w.Flush() // should delegate to the recorder's Flush

	if !rec.Flushed {
		t.Error("Flush() did not delegate to underlying ResponseWriter")
	}

	// 非 Flusher な ResponseWriter でも panic しないこと。
	w2 := &responseWriterWrapper{ResponseWriter: nonFlusherWriter{}}
	w2.Flush()
}

type nonFlusherWriter struct{}

func (nonFlusherWriter) Header() http.Header         { return http.Header{} }
func (nonFlusherWriter) Write(b []byte) (int, error) { return len(b), nil }
func (nonFlusherWriter) WriteHeader(int)             {}

func TestResponseWriterWrapper_DoubleWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	w := &responseWriterWrapper{ResponseWriter: rec, status: http.StatusOK}
	w.WriteHeader(http.StatusCreated)
	w.WriteHeader(http.StatusBadRequest) // should be ignored

	if w.status != http.StatusCreated {
		t.Errorf("status = %d, want 201 (second WriteHeader should be ignored)", w.status)
	}
}
