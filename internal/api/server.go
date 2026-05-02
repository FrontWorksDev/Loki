package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"

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

// apiDescription は OpenAPI スペックの info.description に設定する API 概要。
const apiDescription = `画像の圧縮・フォーマット変換を提供する HTTP API。

## 機能

- ` + "`POST /api/v1/compress`" + ` — 画像ファイルを圧縮する。
- ` + "`POST /api/v1/convert`" + ` — 画像のフォーマットを変換する（JPEG / PNG / WebP 相互変換）。
- ` + "`GET  /api/v1/health`" + ` — サーバー稼働状態を返す。

## 対応フォーマット

JPEG (` + "`image/jpeg`" + `)、PNG (` + "`image/png`" + `)、WebP (` + "`image/webp`" + `)。

## 認証

本 API は認証を要求しない。アクセス制御はネットワーク層 / リバースプロキシ側で行う想定。

## レート制限

クライアント IP ごとに既定で 1 分あたり 30 リクエスト・バースト 10 を許容する。
超過時は ` + "`429 Too Many Requests`" + ` を返し、` + "`Retry-After`" + ` ヘッダーを付与する。
ヘルスチェックエンドポイントはレート制限の対象外。

## エラーレスポンス

エラーは RFC 9457 (Problem Details for HTTP APIs) に準拠した
` + "`application/problem+json`" + ` 形式で返す。`

// Server はAPIサーバーを表す。
type Server struct {
	config      Config
	router      chi.Router
	api         huma.API
	httpServer  *http.Server
	logger      *slog.Logger
	rateLimiter middleware.RateLimiter
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
	humaConfig.Info.Description = apiDescription
	humaConfig.Info.Contact = &huma.Contact{
		Name: "FrontWorksDev",
		URL:  "https://github.com/FrontWorksDev/Loki",
	}
	humaConfig.Info.License = &huma.License{
		Name: "MIT",
		URL:  "https://github.com/FrontWorksDev/Loki/blob/main/LICENSE",
	}

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
		config:      cfg,
		router:      router,
		api:         api,
		logger:      logger,
		rateLimiter: rateLimiter,
		httpServer: &http.Server{
			Addr:              net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
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
	s.logger.Info("server starting",
		slog.String("host", s.config.Host),
		slog.Int("port", s.config.Port),
		slog.String("addr", s.httpServer.Addr),
	)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Shutdown はサーバーをGracefulに停止する。
// レートリミッタの内部 goroutine もここで停止する。
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()
	err := s.httpServer.Shutdown(shutdownCtx)
	if s.rateLimiter != nil {
		// Close は冪等。エラーは現実装では発生しないが将来の実装に備えて
		// HTTP 側のエラーを優先しつつログにとどめる。
		if cerr := s.rateLimiter.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
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
