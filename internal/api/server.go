package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/FrontWorksDev/Loki/internal/api/middleware"
	"github.com/FrontWorksDev/Loki/internal/handler"
	"github.com/FrontWorksDev/Loki/pkg/processor"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// healthPath はレートリミットとボディサイズ制限の対象外とするパス。
const healthPath = "/api/v1/health"

// Server はAPIサーバーを表す。
type Server struct {
	config     Config
	router     chi.Router
	api        huma.API
	httpServer *http.Server
	logger     *slog.Logger
}

// NewServer は新しいAPIサーバーを生成する。
func NewServer(cfg Config) *Server {
	logger := newLogger(cfg.Logging.Level)
	router := chi.NewMux()

	rateLimiter := middleware.NewInMemoryRateLimiter(cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.Burst)

	// ミドルウェア登録順（外側 → 内側）。
	// Chi の制約により Use はルート登録前にすべて呼ぶ必要があるため、
	// レートリミット・ボディサイズ制限はミドルウェア側で healthPath を除外する。
	router.Use(chimw.RequestID)
	router.Use(chimw.Recoverer)
	router.Use(middleware.NewLogging(logger))
	router.Use(middleware.NewCORS(toMiddlewareCORS(cfg.CORS)))
	router.Use(middleware.NewRateLimit(rateLimiter, middleware.WithExemptPaths(healthPath)))
	router.Use(middleware.NewBodyLimit(cfg.BodyLimitBytes, middleware.WithBodyLimitExemptPaths(healthPath)))

	humaConfig := huma.DefaultConfig("Loki Image API", "1.0.0")
	humaConfig.Info.Description = "画像圧縮・変換API"

	api := humachi.New(router, humaConfig)

	processors := map[processor.ImageFormat]processor.Processor{
		processor.FormatJPEG: processor.NewJPEGProcessor(),
		processor.FormatPNG:  processor.NewPNGProcessor(),
		processor.FormatWEBP: processor.NewWEBPProcessor(),
	}
	compressHandler := handler.NewCompressHandler(processors)
	convertHandler := handler.NewConvertHandler(processors)

	RegisterHealth(api)
	RegisterRoutes(api, compressHandler, convertHandler)

	return &Server{
		config: cfg,
		router: router,
		api:    api,
		logger: logger,
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           router,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},
	}
}

// API はHuma APIインスタンスを返す。
func (s *Server) API() huma.API {
	return s.api
}

// Start はサーバーを起動する。
func (s *Server) Start() error {
	s.logger.Info("server starting", slog.Int("port", s.config.Port))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Shutdown はサーバーをGracefulに停止する。
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()
	return s.httpServer.Shutdown(shutdownCtx)
}

// newLogger は構造化JSONロガーを生成する。
func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(handler)
}

func toMiddlewareCORS(c CORSConfig) middleware.CORSConfig {
	return middleware.CORSConfig{
		AllowedOrigins:   c.AllowedOrigins,
		AllowedMethods:   c.AllowedMethods,
		AllowedHeaders:   c.AllowedHeaders,
		AllowCredentials: c.AllowCredentials,
		MaxAge:           c.MaxAge,
	}
}
