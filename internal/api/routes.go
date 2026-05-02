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

// RegisterHealth はヘルスチェックエンドポイントを登録する。
// このエンドポイントは Cloud Run のヘルスチェック互換を想定しており、
// レートリミットやボディサイズ制限の影響を受けないようミドルウェア側の
// exempt path 設定で除外される前提で利用される。
// 想定外エラーが発生しない設計のため commonErrorCodes() は適用しない。
func RegisterHealth(api huma.API) {
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
}

// RegisterRoutes はAPIルートを登録する。
func RegisterRoutes(api huma.API, compressHandler *handler.CompressHandler, convertHandler *handler.ConvertHandler) {
	compressOp := huma.Operation{
		OperationID:  "compress-image",
		Summary:      "画像を圧縮する",
		Description:  "画像ファイルをアップロードして圧縮する。JPEG/PNG/WebP対応。\n\n圧縮の強さはqualityまたはlevelで指定できる。qualityは1-100の数値で直接指定、levelはlow/medium/highの3段階から選択。両方指定した場合はqualityが優先される。\n\nレスポンスヘッダーに圧縮結果のメタデータ（元サイズ、圧縮後サイズ、圧縮率）を付与する。",
		Method:       http.MethodPost,
		Path:         "/api/v1/compress",
		Tags:         []string{"Image"},
		MaxBodyBytes: 50 * 1024 * 1024, // 50MB
		Errors:       commonErrorCodes(),
		Responses: map[string]*huma.Response{
			"200": {
				Description: "圧縮された画像バイナリ",
				Headers: map[string]*huma.Param{
					"X-Original-Size":     {Description: "元のファイルサイズ（バイト）", Schema: &huma.Schema{Type: "integer", Format: "int64"}},
					"X-Compressed-Size":   {Description: "圧縮後のファイルサイズ（バイト）", Schema: &huma.Schema{Type: "integer", Format: "int64"}},
					"X-Compression-Ratio": {Description: "圧縮率（パーセンテージ）", Schema: &huma.Schema{Type: "number"}},
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

	convertOp := huma.Operation{
		OperationID:  "convert-image",
		Summary:      "画像フォーマットを変換する",
		Description:  "画像ファイルをアップロードして指定フォーマットに変換する。JPEG/PNG/WebP間の相互変換に対応。\n\n出力品質はqualityまたはlevelで指定できる。qualityは1-100の数値で直接指定、levelはlow/medium/highの3段階から選択。両方指定した場合はqualityが優先される。\n\n同一フォーマットを指定した場合は圧縮処理にフォールバックする。\n\nレスポンスヘッダーに変換結果のメタデータ（元サイズ、変換後サイズ、元フォーマット、出力フォーマット）を付与する。",
		Method:       http.MethodPost,
		Path:         "/api/v1/convert",
		Tags:         []string{"Image"},
		MaxBodyBytes: 50 * 1024 * 1024, // 50MB
		Errors:       commonErrorCodes(),
		Responses: map[string]*huma.Response{
			"200": {
				Description: "変換された画像バイナリ",
				Headers: map[string]*huma.Param{
					"X-Original-Size":   {Description: "元のファイルサイズ（バイト）", Schema: &huma.Schema{Type: "integer", Format: "int64"}},
					"X-Converted-Size":  {Description: "変換後のファイルサイズ（バイト）", Schema: &huma.Schema{Type: "integer", Format: "int64"}},
					"X-Original-Format": {Description: "元の画像フォーマット", Schema: &huma.Schema{Type: "string"}},
					"X-Output-Format":   {Description: "出力画像フォーマット", Schema: &huma.Schema{Type: "string"}},
				},
				Content: map[string]*huma.MediaType{
					"image/*": {
						Schema: &huma.Schema{Type: "string", Format: "binary"},
					},
				},
			},
		},
	}
	huma.Register(api, convertOp, convertHandler.Handle)
}
