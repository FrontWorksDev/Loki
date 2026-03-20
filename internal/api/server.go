package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

const (
	defaultPort            = 8080
	defaultShutdownTimeout = 5 * time.Second
)

// Config はAPIサーバーの設定を保持する。
type Config struct {
	Port            int
	ShutdownTimeout time.Duration
}

// DefaultConfig はデフォルト設定を返す。
func DefaultConfig() Config {
	return Config{
		Port:            defaultPort,
		ShutdownTimeout: defaultShutdownTimeout,
	}
}

// Server はAPIサーバーを表す。
type Server struct {
	config     Config
	router     chi.Router
	api        huma.API
	httpServer *http.Server
}

// NewServer は新しいAPIサーバーを生成する。
func NewServer(cfg Config) *Server {
	router := chi.NewMux()

	humaConfig := huma.DefaultConfig("Loki Image API", "1.0.0")
	humaConfig.Info.Description = "画像圧縮・変換API"

	api := humachi.New(router, humaConfig)

	s := &Server{
		config: cfg,
		router: router,
		api:    api,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: router,
		},
	}

	RegisterRoutes(api)

	return s
}

// API はHuma APIインスタンスを返す。
func (s *Server) API() huma.API {
	return s.api
}

// Start はサーバーを起動する。
func (s *Server) Start() error {
	fmt.Printf("Server starting on port %d...\n", s.config.Port)
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
