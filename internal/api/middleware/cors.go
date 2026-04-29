package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

// CORSConfig はCORSミドルウェアの設定を表す。
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// NewCORS はCORSミドルウェアを返す。設定値を go-chi/cors に橋渡しする。
// プリフライトには 200 OK が返る（go-chi/cors のデフォルト動作）。
func NewCORS(cfg CORSConfig) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   cfg.AllowedMethods,
		AllowedHeaders:   cfg.AllowedHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	})
}
