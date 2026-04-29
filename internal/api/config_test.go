package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig_AllFields(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.BodyLimitBytes != 50*1024*1024 {
		t.Errorf("BodyLimitBytes = %d, want 50MiB", cfg.BodyLimitBytes)
	}
	if cfg.RateLimit.RequestsPerMinute != 30 {
		t.Errorf("RequestsPerMinute = %d, want 30", cfg.RateLimit.RequestsPerMinute)
	}
	if cfg.RateLimit.Burst != 10 {
		t.Errorf("Burst = %d, want 10", cfg.RateLimit.Burst)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q, want info", cfg.Logging.Level)
	}
	if len(cfg.CORS.AllowedOrigins) == 0 || cfg.CORS.AllowedOrigins[0] != "*" {
		t.Errorf("CORS.AllowedOrigins = %v, want [*]", cfg.CORS.AllowedOrigins)
	}
	if cfg.CORS.AllowCredentials {
		t.Error("CORS.AllowCredentials should default to false")
	}
}

func TestLoadConfig_DefaultsWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(LoadConfigOptions{
		ConfigName:  "nonexistent",
		ConfigPaths: []string{dir},
		EnvPrefix:   "LOKI_TEST_MISSING",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	def := DefaultConfig()
	if cfg.Port != def.Port {
		t.Errorf("Port = %d, want default %d", cfg.Port, def.Port)
	}
	if cfg.BodyLimitBytes != def.BodyLimitBytes {
		t.Errorf("BodyLimitBytes = %d, want default %d", cfg.BodyLimitBytes, def.BodyLimitBytes)
	}
}

func TestLoadConfig_FromYAML(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "default.yaml")
	yaml := `
api:
  port: 9090
  body_limit_bytes: 1048576
  rate_limit:
    requests_per_minute: 120
    burst: 5
  logging:
    level: "warn"
  cors:
    allowed_origins: ["https://example.com"]
    allowed_methods: ["GET", "POST"]
    allowed_headers: ["X-Test"]
    allow_credentials: true
    max_age: 600
`
	if err := os.WriteFile(yamlPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(LoadConfigOptions{
		ConfigPaths: []string{dir},
		EnvPrefix:   "LOKI_TEST_YAML",
	})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.BodyLimitBytes != 1048576 {
		t.Errorf("BodyLimitBytes = %d, want 1048576", cfg.BodyLimitBytes)
	}
	if cfg.RateLimit.RequestsPerMinute != 120 {
		t.Errorf("RequestsPerMinute = %d, want 120", cfg.RateLimit.RequestsPerMinute)
	}
	if cfg.RateLimit.Burst != 5 {
		t.Errorf("Burst = %d, want 5", cfg.RateLimit.Burst)
	}
	if cfg.Logging.Level != "warn" {
		t.Errorf("Logging.Level = %q, want warn", cfg.Logging.Level)
	}
	if len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "https://example.com" {
		t.Errorf("CORS.AllowedOrigins = %v, want [https://example.com]", cfg.CORS.AllowedOrigins)
	}
	if !cfg.CORS.AllowCredentials {
		t.Error("CORS.AllowCredentials should be true from YAML")
	}
	if cfg.CORS.MaxAge != 600 {
		t.Errorf("CORS.MaxAge = %d, want 600", cfg.CORS.MaxAge)
	}
}

func TestLoadConfig_EnvOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `
api:
  port: 9090
  body_limit_bytes: 1048576
`
	if err := os.WriteFile(filepath.Join(dir, "default.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LOKI_TEST_OVR_API_PORT", "7000")
	t.Setenv("LOKI_TEST_OVR_API_BODY_LIMIT_BYTES", "2048")

	cfg, err := LoadConfig(LoadConfigOptions{
		ConfigPaths: []string{dir},
		EnvPrefix:   "LOKI_TEST_OVR",
	})
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Port != 7000 {
		t.Errorf("Port = %d, want 7000 (env override)", cfg.Port)
	}
	if cfg.BodyLimitBytes != 2048 {
		t.Errorf("BodyLimitBytes = %d, want 2048 (env override)", cfg.BodyLimitBytes)
	}
}

func TestLoadConfig_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "default.yaml"), []byte("api: : invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(LoadConfigOptions{
		ConfigPaths: []string{dir},
		EnvPrefix:   "LOKI_TEST_BAD",
	})
	if err == nil {
		t.Error("expected error for malformed yaml")
	}
}
