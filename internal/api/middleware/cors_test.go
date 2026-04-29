package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newCORSConfig(origins ...string) CORSConfig {
	return CORSConfig{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}
}

func TestCORS(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	tests := []struct {
		name             string
		cfg              CORSConfig
		method           string
		origin           string
		reqMethodHeader  string
		wantStatus       int
		wantAllowOrigin  string
		wantAllowMethods bool
	}{
		{
			name:            "allowed_origin_get",
			cfg:             newCORSConfig("https://example.com"),
			method:          http.MethodGet,
			origin:          "https://example.com",
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "https://example.com",
		},
		{
			name:            "disallowed_origin_no_header",
			cfg:             newCORSConfig("https://example.com"),
			method:          http.MethodGet,
			origin:          "https://evil.example",
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "",
		},
		{
			name:            "wildcard_origin",
			cfg:             newCORSConfig("*"),
			method:          http.MethodGet,
			origin:          "https://anywhere.example",
			wantStatus:      http.StatusOK,
			wantAllowOrigin: "*",
		},
		{
			name:             "preflight",
			cfg:              newCORSConfig("https://example.com"),
			method:           http.MethodOptions,
			origin:           "https://example.com",
			reqMethodHeader:  "POST",
			wantStatus:       http.StatusOK,
			wantAllowOrigin:  "https://example.com",
			wantAllowMethods: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewCORS(tt.cfg)(okHandler)
			req := httptest.NewRequest(tt.method, "/api/v1/health", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.reqMethodHeader != "" {
				req.Header.Set("Access-Control-Request-Method", tt.reqMethodHeader)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if got := rr.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllowOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.wantAllowOrigin)
			}
			if tt.wantAllowMethods {
				if rr.Header().Get("Access-Control-Allow-Methods") == "" {
					t.Error("Access-Control-Allow-Methods missing on preflight")
				}
			}
		})
	}
}
