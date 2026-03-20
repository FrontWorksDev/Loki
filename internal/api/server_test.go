package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

func getFreePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func TestNewServer(t *testing.T) {
	cfg := DefaultConfig()
	srv := NewServer(cfg)

	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	if srv.API() == nil {
		t.Fatal("API() returned nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.ShutdownTimeout != 5*time.Second {
		t.Errorf("expected 5s shutdown timeout, got %v", cfg.ShutdownTimeout)
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}

	cfg := Config{
		Port:            port,
		ShutdownTimeout: 5 * time.Second,
	}
	srv := NewServer(cfg)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// サーバーが起動するまで待機
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	if !waitForServer(baseURL, 3*time.Second) {
		t.Fatal("server did not start in time")
	}

	// ヘルスチェックエンドポイントのテスト
	resp, err := http.Get(baseURL + "/api/v1/health")
	if err != nil {
		t.Fatalf("health check request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", body.Status)
	}

	// Graceful shutdown テスト
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	// Start が正常終了したことを確認
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("unexpected server error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop in time")
	}
}

func TestOpenAPIDocsEndpoint(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}

	cfg := Config{
		Port:            port,
		ShutdownTimeout: 5 * time.Second,
	}
	srv := NewServer(cfg)

	go func() {
		_ = srv.Start()
	}()

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	if !waitForServer(baseURL, 3*time.Second) {
		t.Fatal("server did not start in time")
	}
	defer func() { _ = srv.Shutdown(context.Background()) }()

	// OpenAPIドキュメントが /docs で取得できることを確認
	resp, err := http.Get(baseURL + "/docs")
	if err != nil {
		t.Fatalf("docs request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for /docs, got %d", resp.StatusCode)
	}

	// OpenAPI JSON が取得できることも確認
	resp2, err := http.Get(baseURL + "/openapi.json")
	if err != nil {
		t.Fatalf("openapi.json request failed: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for /openapi.json, got %d", resp2.StatusCode)
	}
}

func waitForServer(baseURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/v1/health")
		if err == nil {
			_ = resp.Body.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
