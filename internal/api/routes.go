package api

import (
	"context"
	"net/http"

	"github.com/FrontWorksDev/Loki/internal/handler"
	"github.com/danielgtaylor/huma/v2"
)

// HealthOutput はヘルスチェックのレスポンスを表す。
type HealthOutput struct {
	Body struct {
		Status string `json:"status" example:"ok" doc:"サーバーの稼働状態"`
	}
}

// RegisterRoutes はAPIルートを登録する。
func RegisterRoutes(api huma.API, compressHandler *handler.CompressHandler) {
	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Summary:     "ヘルスチェック",
		Description: "サーバーの稼働状態を確認する",
		Method:      http.MethodGet,
		Path:        "/api/v1/health",
		Tags:        []string{"System"},
	}, func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		resp := &HealthOutput{}
		resp.Body.Status = "ok"
		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:  "compress-image",
		Summary:      "画像を圧縮する",
		Description:  "画像ファイルをアップロードして圧縮する。JPEG/PNG/WebP対応。",
		Method:       http.MethodPost,
		Path:         "/api/v1/compress",
		Tags:         []string{"Image"},
		MaxBodyBytes: 50 * 1024 * 1024, // 50MB
	}, compressHandler.Handle)
}
