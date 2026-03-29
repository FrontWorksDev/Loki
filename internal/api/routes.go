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

	compressOp := huma.Operation{
		OperationID:  "compress-image",
		Summary:      "画像を圧縮する",
		Description:  "画像ファイルをアップロードして圧縮する。JPEG/PNG/WebP対応。\n\n圧縮の強さはqualityまたはlevelで指定できる。qualityは1-100の数値で直接指定、levelはlow/medium/highの3段階から選択。両方指定した場合はqualityが優先される。\n\nレスポンスヘッダーに圧縮結果のメタデータ（元サイズ、圧縮後サイズ、圧縮率）を付与する。",
		Method:       http.MethodPost,
		Path:         "/api/v1/compress",
		Tags:         []string{"Image"},
		MaxBodyBytes: 50 * 1024 * 1024, // 50MB
		Responses: map[string]*huma.Response{
			"200": {
				Description: "圧縮された画像バイナリ",
				Headers: map[string]*huma.Param{
					"X-Original-Size":     {Description: "元のファイルサイズ（バイト）", Schema: &huma.Schema{Type: "string"}},
					"X-Compressed-Size":   {Description: "圧縮後のファイルサイズ（バイト）", Schema: &huma.Schema{Type: "string"}},
					"X-Compression-Ratio": {Description: "圧縮率（パーセンテージ）", Schema: &huma.Schema{Type: "string"}},
				},
				Content: map[string]*huma.MediaType{
					"image/*": {
						Schema: &huma.Schema{Type: "string", Format: "binary"},
					},
				},
			},
		},
	}
	huma.Register(api, compressOp, compressHandler.Handle)
}
