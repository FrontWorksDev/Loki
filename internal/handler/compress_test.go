package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/FrontWorksDev/Loki/pkg/processor"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func newTestProcessors() map[processor.ImageFormat]processor.Processor {
	return map[processor.ImageFormat]processor.Processor{
		processor.FormatJPEG: processor.NewJPEGProcessor(),
		processor.FormatPNG:  processor.NewPNGProcessor(),
		processor.FormatWEBP: processor.NewWEBPProcessor(),
	}
}

func setupTestAPI(t *testing.T) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	h := NewCompressHandler(newTestProcessors())
	huma.Register(api, huma.Operation{
		OperationID:  "compress-image",
		Method:       http.MethodPost,
		Path:         "/api/v1/compress",
		MaxBodyBytes: 50 * 1024 * 1024,
	}, h.Handle)
	return api
}

func createTestJPEG(t *testing.T, width, height, quality int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / width),
				G: uint8(y * 255 / height),
				B: 128, A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatalf("failed to create test JPEG: %v", err)
	}
	return buf.Bytes()
}

func createTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / width),
				G: uint8(y * 255 / height),
				B: 128, A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	return buf.Bytes()
}

func buildMultipartRequest(t *testing.T, fields map[string]string, fileName, fileContentType string, fileData []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if fileData != nil {
		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": {fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName)},
			"Content-Type":        {fileContentType},
		})
		if err != nil {
			t.Fatalf("failed to create file part: %v", err)
		}
		if _, err := part.Write(fileData); err != nil {
			t.Fatalf("failed to write file data: %v", err)
		}
	}

	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			t.Fatalf("failed to write field %s: %v", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	return &body, writer.FormDataContentType()
}

func doMultipartRequest(t *testing.T, api humatest.TestAPI, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	return api.Do(http.MethodPost, "/api/v1/compress",
		"Content-Type: "+contentType,
		body,
	)
}

func TestCompressJPEG(t *testing.T) {
	api := setupTestAPI(t)
	jpegData := createTestJPEG(t, 100, 100, 95)
	body, ct := buildMultipartRequest(t, nil, "test.jpg", "image/jpeg", jpegData)

	resp := doMultipartRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if resp.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("expected Content-Type image/jpeg, got %s", resp.Header().Get("Content-Type"))
	}
	if resp.Body.Len() == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestCompressPNG(t *testing.T) {
	api := setupTestAPI(t)
	pngData := createTestPNG(t, 100, 100)
	body, ct := buildMultipartRequest(t, nil, "test.png", "image/png", pngData)

	resp := doMultipartRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if resp.Header().Get("Content-Type") != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", resp.Header().Get("Content-Type"))
	}
}

func TestCompressWebP(t *testing.T) {
	api := setupTestAPI(t)
	webpData := encodeTestWebP(t, 100, 100)
	body, ct := buildMultipartRequest(t, nil, "test.webp", "image/webp", webpData)

	resp := doMultipartRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if resp.Header().Get("Content-Type") != "image/webp" {
		t.Errorf("expected Content-Type image/webp, got %s", resp.Header().Get("Content-Type"))
	}
}

func encodeTestWebP(t *testing.T, width, height int) []byte {
	t.Helper()
	// WebPプロセッサのCompressにPNG入力を渡して出力を得る
	pngData := createTestPNG(t, width, height)
	proc := processor.NewWEBPProcessor()
	var buf bytes.Buffer
	_, err := proc.Compress(context.Background(), bytes.NewReader(pngData), &buf, processor.DefaultCompressOptions())
	if err != nil {
		t.Fatalf("failed to create test WebP: %v", err)
	}
	return buf.Bytes()
}

func TestCompressWithQuality(t *testing.T) {
	api := setupTestAPI(t)
	jpegData := createTestJPEG(t, 100, 100, 95)
	body, ct := buildMultipartRequest(t, map[string]string{"quality": "50"}, "test.jpg", "image/jpeg", jpegData)

	resp := doMultipartRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCompressWithLevel(t *testing.T) {
	api := setupTestAPI(t)
	jpegData := createTestJPEG(t, 100, 100, 95)
	body, ct := buildMultipartRequest(t, map[string]string{"level": "high"}, "test.jpg", "image/jpeg", jpegData)

	resp := doMultipartRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCompressUnsupportedFormat(t *testing.T) {
	api := setupTestAPI(t)
	body, ct := buildMultipartRequest(t, nil, "test.gif", "image/gif", []byte("GIF89a"))

	resp := doMultipartRequest(t, api, body, ct)

	if resp.Code == http.StatusOK {
		t.Error("expected error for unsupported format, got 200")
	}
}

func TestCompressResponseHeaders(t *testing.T) {
	api := setupTestAPI(t)
	jpegData := createTestJPEG(t, 100, 100, 95)
	body, ct := buildMultipartRequest(t, nil, "test.jpg", "image/jpeg", jpegData)

	resp := doMultipartRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	origSize := resp.Header().Get("X-Original-Size")
	compSize := resp.Header().Get("X-Compressed-Size")
	ratio := resp.Header().Get("X-Compression-Ratio")

	if origSize == "" {
		t.Error("X-Original-Size header is missing")
	}
	if compSize == "" {
		t.Error("X-Compressed-Size header is missing")
	}
	if ratio == "" {
		t.Error("X-Compression-Ratio header is missing")
	}

	origSizeInt, err := strconv.ParseInt(origSize, 10, 64)
	if err != nil {
		t.Errorf("X-Original-Size is not a valid integer: %s", origSize)
	}
	compSizeInt, err := strconv.ParseInt(compSize, 10, 64)
	if err != nil {
		t.Errorf("X-Compressed-Size is not a valid integer: %s", compSize)
	}

	if origSizeInt <= 0 {
		t.Errorf("X-Original-Size should be positive, got %d", origSizeInt)
	}
	if compSizeInt <= 0 {
		t.Errorf("X-Compressed-Size should be positive, got %d", compSizeInt)
	}
}

func TestDetectFormatFromMIME(t *testing.T) {
	tests := []struct {
		mimeType string
		expected processor.ImageFormat
		wantErr  bool
	}{
		{"image/jpeg", processor.FormatJPEG, false},
		{"image/png", processor.FormatPNG, false},
		{"image/webp", processor.FormatWEBP, false},
		{"image/gif", 0, true},
		{"text/plain", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			got, err := detectFormatFromMIME(tt.mimeType)
			if (err != nil) != tt.wantErr {
				t.Errorf("detectFormatFromMIME(%q) error = %v, wantErr %v", tt.mimeType, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("detectFormatFromMIME(%q) = %v, want %v", tt.mimeType, got, tt.expected)
			}
		})
	}
}

func TestCompressNoFile(t *testing.T) {
	api := setupTestAPI(t)
	body, ct := buildMultipartRequest(t, nil, "", "", nil)

	resp := doMultipartRequest(t, api, body, ct)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCompressProcessorNotFound(t *testing.T) {
	// JPEGプロセッサのみ登録し、PNGを送信
	_, api := humatest.New(t)
	h := NewCompressHandler(map[processor.ImageFormat]processor.Processor{
		processor.FormatJPEG: processor.NewJPEGProcessor(),
	})
	huma.Register(api, huma.Operation{
		OperationID:  "compress-image",
		Method:       http.MethodPost,
		Path:         "/api/v1/compress",
		MaxBodyBytes: 50 * 1024 * 1024,
	}, h.Handle)

	pngData := createTestPNG(t, 100, 100)
	body, ct := buildMultipartRequest(t, nil, "test.png", "image/png", pngData)

	resp := doMultipartRequest(t, api, body, ct)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCompressInvalidImageData(t *testing.T) {
	api := setupTestAPI(t)
	// 壊れたJPEGデータを送信してデコードエラーを発生させる
	body, ct := buildMultipartRequest(t, nil, "broken.jpg", "image/jpeg", []byte("not a real jpeg"))

	resp := doMultipartRequest(t, api, body, ct)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

// mockProcessor はテスト用のモックプロセッサ。
type mockProcessor struct {
	compressErr error
	convertErr  error
}

func (m *mockProcessor) Compress(_ context.Context, _ io.Reader, _ io.Writer, _ processor.CompressOptions) (*processor.Result, error) {
	return nil, m.compressErr
}

func (m *mockProcessor) Convert(_ context.Context, _ io.Reader, _ io.Writer, _ processor.ConvertOptions) (*processor.Result, error) {
	if m.convertErr != nil {
		return nil, m.convertErr
	}
	return nil, errors.New("not implemented")
}

func (m *mockProcessor) SupportedFormats() []processor.ImageFormat {
	return nil
}

func TestCompressFileTooLarge(t *testing.T) {
	_, api := humatest.New(t)
	mock := &mockProcessor{compressErr: processor.ErrFileTooLarge}
	h := NewCompressHandler(map[processor.ImageFormat]processor.Processor{
		processor.FormatJPEG: mock,
	})
	huma.Register(api, huma.Operation{
		OperationID:  "compress-image",
		Method:       http.MethodPost,
		Path:         "/api/v1/compress",
		MaxBodyBytes: 50 * 1024 * 1024,
	}, h.Handle)

	jpegData := createTestJPEG(t, 10, 10, 50)
	body, ct := buildMultipartRequest(t, nil, "test.jpg", "image/jpeg", jpegData)

	resp := doMultipartRequest(t, api, body, ct)

	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestParseCompressionLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected processor.CompressionLevel
	}{
		{"low", processor.CompressionLow},
		{"medium", processor.CompressionMedium},
		{"high", processor.CompressionHigh},
		{"", processor.CompressionMedium},
		{"unknown", processor.CompressionMedium},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseCompressionLevel(tt.input)
			if got != tt.expected {
				t.Errorf("parseCompressionLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
