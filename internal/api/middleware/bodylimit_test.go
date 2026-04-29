package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBodyLimit(t *testing.T) {
	echoHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})

	tests := []struct {
		name         string
		maxBytes     int64
		body         []byte
		path         string
		exemptPaths  []string
		wantStatus   int
		wantHandler  bool
		wantContains string
	}{
		{
			name:        "under_limit",
			maxBytes:    100,
			body:        []byte("hello"),
			path:        "/api/v1/compress",
			wantStatus:  http.StatusOK,
			wantHandler: true,
		},
		{
			name:         "over_limit_content_length",
			maxBytes:     10,
			body:         bytes.Repeat([]byte("a"), 50),
			path:         "/api/v1/compress",
			wantStatus:   http.StatusRequestEntityTooLarge,
			wantHandler:  false,
			wantContains: "Payload Too Large",
		},
		{
			name:         "exact_limit_plus_one",
			maxBytes:     5,
			body:         []byte("hello!"),
			path:         "/api/v1/compress",
			wantStatus:   http.StatusRequestEntityTooLarge,
			wantHandler:  false,
			wantContains: "max_bytes",
		},
		{
			name:        "exempt_path_skips_limit",
			maxBytes:    1,
			body:        bytes.Repeat([]byte("x"), 100),
			path:        "/api/v1/health",
			exemptPaths: []string{"/api/v1/health"},
			wantStatus:  http.StatusOK,
			wantHandler: true,
		},
		{
			name:        "zero_max_bytes_disabled",
			maxBytes:    0,
			body:        bytes.Repeat([]byte("x"), 100),
			path:        "/api/v1/compress",
			wantStatus:  http.StatusOK,
			wantHandler: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			tracker := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				echoHandler.ServeHTTP(w, r)
			})

			var opts []BodyLimitOption
			if len(tt.exemptPaths) > 0 {
				opts = append(opts, WithBodyLimitExemptPaths(tt.exemptPaths...))
			}
			h := NewBodyLimit(tt.maxBytes, opts...)(tracker)

			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(tt.body))
			req.ContentLength = int64(len(tt.body))
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if handlerCalled != tt.wantHandler {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.wantHandler)
			}
			if tt.wantContains != "" {
				body := rr.Body.String()
				if !bytes.Contains([]byte(body), []byte(tt.wantContains)) {
					t.Errorf("body does not contain %q\nbody=%s", tt.wantContains, body)
				}
				// JSON でパースできること
				var m map[string]any
				if err := json.Unmarshal([]byte(body), &m); err != nil {
					t.Errorf("body is not JSON: %v", err)
				}
			}
		})
	}
}
