package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/FrontWorksDev/Loki/internal/api"
	"github.com/spf13/viper"
)

const (
	configName = "default"
	configType = "yaml"
	envPrefix  = "LOKI"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "設定の読み込みに失敗しました: %v\n", err)
		os.Exit(1)
	}

	srv := api.NewServer(cfg)

	// シグナルによるGraceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)

	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

	exitCode := 0

	select {
	case <-ctx.Done():
		fmt.Println("\nShutting down server...")
	case err := <-errCh:
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		fmt.Println("\nShutting down server...")
		exitCode = 1
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Shutdown error: %v\n", err)
		exitCode = 1
	} else {
		fmt.Println("Server stopped gracefully.")
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// loadConfig は configs/default.yaml と環境変数（LOKI_API_*）を読み込み、
// api.Config に詰めて返す。設定ファイルが存在しない場合はデフォルト値で起動する。
func loadConfig() (api.Config, error) {
	v := viper.New()

	cfg := api.DefaultConfig()
	setDefaults(v, cfg)

	v.SetConfigName(configName)
	v.SetConfigType(configType)
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return api.Config{}, fmt.Errorf("config file: %w", err)
		}
	}

	cfg.Port = v.GetInt("api.port")
	cfg.CORS.AllowedOrigins = v.GetStringSlice("api.cors.allowed_origins")
	cfg.CORS.AllowedMethods = v.GetStringSlice("api.cors.allowed_methods")
	cfg.CORS.AllowedHeaders = v.GetStringSlice("api.cors.allowed_headers")
	cfg.CORS.AllowCredentials = v.GetBool("api.cors.allow_credentials")
	cfg.CORS.MaxAge = v.GetInt("api.cors.max_age")
	cfg.BodyLimitBytes = v.GetInt64("api.body_limit_bytes")
	cfg.RateLimit.RequestsPerMinute = v.GetInt("api.rate_limit.requests_per_minute")
	cfg.RateLimit.Burst = v.GetInt("api.rate_limit.burst")
	cfg.Logging.Level = v.GetString("api.logging.level")

	return cfg, nil
}

func setDefaults(v *viper.Viper, cfg api.Config) {
	v.SetDefault("api.port", cfg.Port)
	v.SetDefault("api.cors.allowed_origins", cfg.CORS.AllowedOrigins)
	v.SetDefault("api.cors.allowed_methods", cfg.CORS.AllowedMethods)
	v.SetDefault("api.cors.allowed_headers", cfg.CORS.AllowedHeaders)
	v.SetDefault("api.cors.allow_credentials", cfg.CORS.AllowCredentials)
	v.SetDefault("api.cors.max_age", cfg.CORS.MaxAge)
	v.SetDefault("api.body_limit_bytes", cfg.BodyLimitBytes)
	v.SetDefault("api.rate_limit.requests_per_minute", cfg.RateLimit.RequestsPerMinute)
	v.SetDefault("api.rate_limit.burst", cfg.RateLimit.Burst)
	v.SetDefault("api.logging.level", cfg.Logging.Level)
}
