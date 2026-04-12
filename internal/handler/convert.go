package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/FrontWorksDev/Loki/pkg/processor"
	"github.com/danielgtaylor/huma/v2"
)

// ConvertFormData はフォーマット変換のmultipart/form-dataを表す。
type ConvertFormData struct {
	File    huma.FormFile `form:"file" contentType:"image/jpeg,image/png,image/webp" required:"true" doc:"変換する画像ファイル（JPEG/PNG/WebP）"`
	Format  string        `form:"format" enum:"jpeg,png,webp" required:"true" doc:"出力フォーマット（jpeg/png/webp）"`
	Quality int           `form:"quality" minimum:"0" maximum:"100" required:"false" doc:"出力品質（1-100）。0または未指定の場合はlevelに基づくデフォルト値を使用"`
	Level   string        `form:"level" enum:"low,medium,high," required:"false" doc:"圧縮レベル。low=圧縮優先, medium=バランス(デフォルト), high=品質優先。quality指定時はqualityが優先"`
}

// ConvertInput はフォーマット変換エンドポイントのリクエストを表す。
type ConvertInput struct {
	RawBody huma.MultipartFormFiles[ConvertFormData]
}

// ConvertHandler は画像フォーマット変換ハンドラーを表す。
type ConvertHandler struct {
	processors map[processor.ImageFormat]processor.Processor
}

// NewConvertHandler は新しいConvertHandlerを生成する。
func NewConvertHandler(processors map[processor.ImageFormat]processor.Processor) *ConvertHandler {
	return &ConvertHandler{processors: processors}
}

// Handle は画像フォーマット変換リクエストを処理する。
func (h *ConvertHandler) Handle(ctx context.Context, input *ConvertInput) (*huma.StreamResponse, error) {
	data := input.RawBody.Data()

	if !data.File.IsSet {
		return nil, huma.Error422UnprocessableEntity("ファイルが指定されていません")
	}

	inputFormat, err := detectFormatFromMIME(data.File.ContentType)
	if err != nil {
		return nil, huma.Error400BadRequest("非対応の画像フォーマットです", err)
	}

	outputFormat, err := parseImageFormat(data.Format)
	if err != nil {
		return nil, huma.Error400BadRequest("非対応の出力フォーマットです", err)
	}

	// 同一フォーマットの場合は圧縮にフォールバック
	if inputFormat == outputFormat {
		return h.handleCompress(ctx, *data, inputFormat)
	}

	proc, ok := h.processors[outputFormat]
	if !ok {
		return nil, huma.Error400BadRequest("非対応の出力フォーマットです")
	}

	opts := processor.ConvertOptions{
		Format: outputFormat,
		CompressOptions: processor.CompressOptions{
			Quality:     data.Quality,
			Level:       parseCompressionLevel(data.Level),
			MaxFileSize: maxFileSize,
		},
	}

	var buf bytes.Buffer
	result, err := proc.Convert(ctx, data.File, &buf, opts)
	if err != nil {
		return nil, handleProcessorError(err)
	}

	return buildConvertResponse(&buf, result, inputFormat, outputFormat), nil
}

// handleCompress は同一フォーマット変換時の圧縮フォールバックを処理する。
func (h *ConvertHandler) handleCompress(ctx context.Context, data ConvertFormData, format processor.ImageFormat) (*huma.StreamResponse, error) {
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
		return nil, handleProcessorError(err)
	}

	return buildConvertResponse(&buf, result, format, format), nil
}

// handleProcessorError はプロセッサエラーをHTTPエラーに変換する。
func handleProcessorError(err error) error {
	if errors.Is(err, processor.ErrFileTooLarge) {
		return huma.Error413RequestEntityTooLarge("ファイルサイズが上限を超えています", err)
	}
	if isDecodeError(err) {
		return huma.Error400BadRequest("画像データが不正です", err)
	}
	return huma.Error500InternalServerError("変換処理に失敗しました", err)
}

// buildConvertResponse は変換結果からStreamResponseを生成する。
func buildConvertResponse(buf *bytes.Buffer, result *processor.Result, inputFormat, outputFormat processor.ImageFormat) *huma.StreamResponse {
	converted := buf.Bytes()
	mimeType := outputFormat.MIMEType()
	origSize := fmt.Sprintf("%d", result.OriginalSize)
	convSize := fmt.Sprintf("%d", result.CompressedSize)
	origFmt := inputFormat.String()
	outFmt := outputFormat.String()

	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Content-Type", mimeType)
			ctx.SetHeader("X-Original-Size", origSize)
			ctx.SetHeader("X-Converted-Size", convSize)
			ctx.SetHeader("X-Original-Format", origFmt)
			ctx.SetHeader("X-Output-Format", outFmt)
			_, _ = ctx.BodyWriter().Write(converted)
		},
	}
}

// parseImageFormat は文字列からImageFormatに変換する。
func parseImageFormat(s string) (processor.ImageFormat, error) {
	switch s {
	case "jpeg":
		return processor.FormatJPEG, nil
	case "png":
		return processor.FormatPNG, nil
	case "webp":
		return processor.FormatWEBP, nil
	default:
		return 0, fmt.Errorf("非対応のフォーマット: %s", s)
	}
}
