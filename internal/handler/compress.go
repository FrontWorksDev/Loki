package handler

import (
	"bytes"
	"context"
	"fmt"

	"github.com/FrontWorksDev/Loki/pkg/processor"
	"github.com/danielgtaylor/huma/v2"
)

const maxFileSize = 50 * 1024 * 1024 // 50MB

// CompressFormData はmultipart/form-dataのフォームデータを表す。
type CompressFormData struct {
	File    huma.FormFile `form:"file" contentType:"image/jpeg,image/png,image/webp" required:"true"`
	Quality int           `form:"quality" minimum:"0" maximum:"100" required:"false"`
	Level   string        `form:"level" enum:"low,medium,high," required:"false"`
}

// CompressInput は圧縮エンドポイントのリクエストを表す。
type CompressInput struct {
	RawBody huma.MultipartFormFiles[CompressFormData]
}

// CompressOutput は圧縮エンドポイントのレスポンスを表す。
type CompressOutput struct {
	ContentType       string `header:"Content-Type"`
	XOriginalSize     string `header:"X-Original-Size"`
	XCompressedSize   string `header:"X-Compressed-Size"`
	XCompressionRatio string `header:"X-Compression-Ratio"`
	Body              []byte
}

// CompressHandler は画像圧縮ハンドラーを表す。
type CompressHandler struct {
	processors map[processor.ImageFormat]processor.Processor
}

// NewCompressHandler は新しいCompressHandlerを生成する。
func NewCompressHandler(processors map[processor.ImageFormat]processor.Processor) *CompressHandler {
	return &CompressHandler{processors: processors}
}

// Handle は画像圧縮リクエストを処理する。
func (h *CompressHandler) Handle(ctx context.Context, input *CompressInput) (*CompressOutput, error) {
	data := input.RawBody.Data()

	if !data.File.IsSet {
		return nil, huma.Error422UnprocessableEntity("ファイルが指定されていません")
	}

	format, err := detectFormatFromMIME(data.File.ContentType)
	if err != nil {
		return nil, huma.Error400BadRequest("非対応の画像フォーマットです", err)
	}

	proc, ok := h.processors[format]
	if !ok {
		return nil, huma.Error400BadRequest("非対応の画像フォーマットです")
	}

	opts := processor.CompressOptions{
		Quality:     data.Quality,
		Level:       parseCompressionLevel(data.Level),
		MaxFileSize: maxFileSize,
	}

	var buf bytes.Buffer
	result, err := proc.Compress(ctx, data.File, &buf, opts)
	if err != nil {
		return nil, huma.Error500InternalServerError("圧縮処理に失敗しました", err)
	}

	return &CompressOutput{
		ContentType:       format.MIMEType(),
		XOriginalSize:     fmt.Sprintf("%d", result.OriginalSize),
		XCompressedSize:   fmt.Sprintf("%d", result.CompressedSize),
		XCompressionRatio: fmt.Sprintf("%.1f", result.CompressionRatio()),
		Body:              buf.Bytes(),
	}, nil
}

// detectFormatFromMIME はMIMEタイプからImageFormatを判定する。
func detectFormatFromMIME(mimeType string) (processor.ImageFormat, error) {
	switch mimeType {
	case "image/jpeg":
		return processor.FormatJPEG, nil
	case "image/png":
		return processor.FormatPNG, nil
	case "image/webp":
		return processor.FormatWEBP, nil
	default:
		return 0, fmt.Errorf("非対応のMIMEタイプ: %s", mimeType)
	}
}

// parseCompressionLevel は文字列からCompressionLevelに変換する。
func parseCompressionLevel(s string) processor.CompressionLevel {
	switch s {
	case "low":
		return processor.CompressionLow
	case "high":
		return processor.CompressionHigh
	default:
		return processor.CompressionMedium
	}
}
