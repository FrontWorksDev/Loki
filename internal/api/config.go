package api

import "time"

const (
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

// DefaultConfig はデフォルト設定を返す。
func DefaultConfig() Config {
	return Config{
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
