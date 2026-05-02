package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultHost              = "0.0.0.0"
	defaultPort              = 8080
	defaultShutdownTimeout   = 5 * time.Second
	defaultReadHeaderTimeout = 10 * time.Second
	defaultReadTimeout       = 30 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 120 * time.Second
	defaultBodyLimitBytes    = int64(50 * 1024 * 1024) // 50 MiB
	defaultRateRPM           = 30
	defaultRateBurst         = 10
	defaultLogLevel          = "info"
)

// Config はAPIサーバーの設定を保持する。
type Config struct {
	Host              string
	Port              int
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	CORS           CORSConfig
	BodyLimitBytes int64
	RateLimit      RateLimitConfig
	Logging        LoggingConfig
}

// CORSConfig はCORSミドルウェアの設定を表す。
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// RateLimitConfig はレートリミットミドルウェアの設定を表す。
type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

// LoggingConfig はロギングミドルウェアの設定を表す。
type LoggingConfig struct {
	Level string // debug/info/warn/error
}

// LoadConfigOptions は LoadConfig の挙動をカスタマイズするためのオプション。
type LoadConfigOptions struct {
	// ConfigName はファイル名（拡張子なし）。デフォルトは "default"。
	ConfigName string
	// ConfigType はファイル形式。デフォルトは "yaml"。
	ConfigType string
	// ConfigPaths は設定ファイルの探索パス。デフォルトは ["./configs", "."]。
	ConfigPaths []string
	// EnvPrefix は環境変数のプレフィックス。デフォルトは "LOKI"。
	EnvPrefix string
}

// LoadConfig は設定ファイル（YAML）と環境変数（{EnvPrefix}_API_*）を読み込み、
// Config を返す。設定ファイルが存在しない場合はデフォルト値で構築する。
func LoadConfig(opts LoadConfigOptions) (Config, error) {
	if opts.ConfigName == "" {
		opts.ConfigName = "default"
	}
	if opts.ConfigType == "" {
		opts.ConfigType = "yaml"
	}
	if len(opts.ConfigPaths) == 0 {
		opts.ConfigPaths = []string{"./configs", "."}
	}
	if opts.EnvPrefix == "" {
		opts.EnvPrefix = "LOKI"
	}

	v := viper.New()
	cfg := DefaultConfig()
	setDefaults(v, cfg)

	v.SetConfigName(opts.ConfigName)
	v.SetConfigType(opts.ConfigType)
	for _, p := range opts.ConfigPaths {
		v.AddConfigPath(p)
	}

	v.SetEnvPrefix(opts.EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("config file: %w", err)
		}
	}

	cfg.Host = v.GetString("api.host")
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

func setDefaults(v *viper.Viper, cfg Config) {
	v.SetDefault("api.host", cfg.Host)
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

// DefaultConfig はデフォルト設定を返す。
func DefaultConfig() Config {
	return Config{
		Host:              defaultHost,
		Port:              defaultPort,
		ShutdownTimeout:   defaultShutdownTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		CORS: CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: false,
			MaxAge:           300,
		},
		BodyLimitBytes: defaultBodyLimitBytes,
		RateLimit: RateLimitConfig{
			RequestsPerMinute: defaultRateRPM,
			Burst:             defaultRateBurst,
		},
		Logging: LoggingConfig{
			Level: defaultLogLevel,
		},
	}
}
