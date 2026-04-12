package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/FrontWorksDev/Loki/pkg/processor"
	"github.com/danielgtaylor/huma/v2"
)

const maxFileSize = 50 * 1024 * 1024 // 50MB

// CompressFormData はmultipart/form-dataのフォームデータを表す。
type CompressFormData struct {
	File    huma.FormFile `form:"file" contentType:"image/jpeg,image/png,image/webp" required:"true" doc:"圧縮する画像ファイル（JPEG/PNG/WebP）"`
	Quality int           `form:"quality" minimum:"0" maximum:"100" required:"false" doc:"圧縮品質（1-100）。0または未指定の場合はlevelに基づくデフォルト値を使用"`
	Level   string        `form:"level" enum:"low,medium,high," required:"false" doc:"圧縮レベル。low=圧縮優先(JPEG:60), medium=バランス(JPEG:75,デフォルト), high=品質優先(JPEG:90)。quality指定時はqualityが優先"`
}

// CompressInput は圧縮エンドポイントのリクエストを表す。
type CompressInput struct {
	RawBody huma.MultipartFormFiles[CompressFormData]
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
func (h *CompressHandler) Handle(ctx context.Context, input *CompressInput) (*huma.StreamResponse, error) {
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
		if errors.Is(err, processor.ErrFileTooLarge) {
			return nil, huma.Error413RequestEntityTooLarge("ファイルサイズが上限を超えています", err)
		}
		if isDecodeError(err) {
			return nil, huma.Error400BadRequest("画像データが不正です", err)
		}
		return nil, huma.Error500InternalServerError("圧縮処理に失敗しました", err)
	}

	compressed := buf.Bytes()
	mimeType := format.MIMEType()
	origSize := fmt.Sprintf("%d", result.OriginalSize)
	compSize := fmt.Sprintf("%d", result.CompressedSize)
	ratio := fmt.Sprintf("%.1f", result.CompressionRatio())

	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Content-Type", mimeType)
			ctx.SetHeader("X-Original-Size", origSize)
			ctx.SetHeader("X-Compressed-Size", compSize)
			ctx.SetHeader("X-Compression-Ratio", ratio)
			_, _ = ctx.BodyWriter().Write(compressed)
		},
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

// isDecodeError はデコード関連のエラーかどうかを判定する。
func isDecodeError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "decode") || strings.Contains(msg, "unknown format") || strings.Contains(msg, "invalid")
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
