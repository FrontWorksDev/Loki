package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FrontWorksDev/Loki/pkg/processor"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func setupConvertTestAPI(t *testing.T) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	h := NewConvertHandler(newTestProcessors())
	huma.Register(api, huma.Operation{
		OperationID:  "convert-image",
		Method:       http.MethodPost,
		Path:         "/api/v1/convert",
		MaxBodyBytes: 50 * 1024 * 1024,
	}, h.Handle)
	return api
}

func doConvertRequest(t *testing.T, api humatest.TestAPI, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	return api.Do(http.MethodPost, "/api/v1/convert",
		"Content-Type: "+contentType,
		body,
	)
}

func TestConvertJPEGToWebP(t *testing.T) {
	api := setupConvertTestAPI(t)
	jpegData := createTestJPEG(t, 100, 100, 95)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "webp"}, "test.jpg", "image/jpeg", jpegData)

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if resp.Header().Get("Content-Type") != "image/webp" {
		t.Errorf("expected Content-Type image/webp, got %s", resp.Header().Get("Content-Type"))
	}
	if resp.Body.Len() == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestConvertPNGToJPEG(t *testing.T) {
	api := setupConvertTestAPI(t)
	pngData := createTestPNG(t, 100, 100)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "jpeg"}, "test.png", "image/png", pngData)

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if resp.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("expected Content-Type image/jpeg, got %s", resp.Header().Get("Content-Type"))
	}
}

func TestConvertWebPToPNG(t *testing.T) {
	api := setupConvertTestAPI(t)
	webpData := encodeTestWebP(t, 100, 100)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "png"}, "test.webp", "image/webp", webpData)

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if resp.Header().Get("Content-Type") != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", resp.Header().Get("Content-Type"))
	}
}

func TestConvertWithQuality(t *testing.T) {
	api := setupConvertTestAPI(t)
	pngData := createTestPNG(t, 100, 100)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "jpeg", "quality": "80"}, "test.png", "image/png", pngData)

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestConvertWithLevel(t *testing.T) {
	api := setupConvertTestAPI(t)
	jpegData := createTestJPEG(t, 100, 100, 95)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "webp", "level": "high"}, "test.jpg", "image/jpeg", jpegData)

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestConvertSameFormatFallback(t *testing.T) {
	api := setupConvertTestAPI(t)
	jpegData := createTestJPEG(t, 100, 100, 95)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "jpeg"}, "test.jpg", "image/jpeg", jpegData)

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 for same-format fallback, got %d: %s", resp.Code, resp.Body.String())
	}
	if resp.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("expected Content-Type image/jpeg, got %s", resp.Header().Get("Content-Type"))
	}
}

func TestConvertResponseHeaders(t *testing.T) {
	api := setupConvertTestAPI(t)
	jpegData := createTestJPEG(t, 100, 100, 95)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "webp"}, "test.jpg", "image/jpeg", jpegData)

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	headers := []string{"X-Original-Size", "X-Converted-Size", "X-Original-Format", "X-Output-Format"}
	for _, h := range headers {
		if resp.Header().Get(h) == "" {
			t.Errorf("%s header is missing", h)
		}
	}

	if resp.Header().Get("X-Original-Format") != "jpeg" {
		t.Errorf("expected X-Original-Format jpeg, got %s", resp.Header().Get("X-Original-Format"))
	}
	if resp.Header().Get("X-Output-Format") != "webp" {
		t.Errorf("expected X-Output-Format webp, got %s", resp.Header().Get("X-Output-Format"))
	}
}

func TestConvertUnsupportedOutputFormat(t *testing.T) {
	api := setupConvertTestAPI(t)
	jpegData := createTestJPEG(t, 100, 100, 95)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "gif"}, "test.jpg", "image/jpeg", jpegData)

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code == http.StatusOK {
		t.Error("expected error for unsupported output format, got 200")
	}
}

func TestConvertNoFile(t *testing.T) {
	api := setupConvertTestAPI(t)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "webp"}, "", "", nil)

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestConvertNoFormat(t *testing.T) {
	api := setupConvertTestAPI(t)
	jpegData := createTestJPEG(t, 100, 100, 95)
	body, ct := buildMultipartRequest(t, nil, "test.jpg", "image/jpeg", jpegData)

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code == http.StatusOK {
		t.Error("expected error for missing format, got 200")
	}
}

func TestConvertUnsupportedInputFormat(t *testing.T) {
	api := setupConvertTestAPI(t)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "webp"}, "test.gif", "image/gif", []byte("GIF89a"))

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code == http.StatusOK {
		t.Error("expected error for unsupported input format, got 200")
	}
}

func TestConvertInvalidImageData(t *testing.T) {
	api := setupConvertTestAPI(t)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "webp"}, "broken.jpg", "image/jpeg", []byte("not a real jpeg"))

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestConvertFileTooLarge(t *testing.T) {
	_, api := humatest.New(t)
	mock := &mockProcessor{convertErr: processor.ErrFileTooLarge}
	h := NewConvertHandler(map[processor.ImageFormat]processor.Processor{
		processor.FormatJPEG: mock,
		processor.FormatWEBP: mock,
	})
	huma.Register(api, huma.Operation{
		OperationID:  "convert-image",
		Method:       http.MethodPost,
		Path:         "/api/v1/convert",
		MaxBodyBytes: 50 * 1024 * 1024,
	}, h.Handle)

	jpegData := createTestJPEG(t, 10, 10, 50)
	body, ct := buildMultipartRequest(t, map[string]string{"format": "webp"}, "test.jpg", "image/jpeg", jpegData)

	resp := doConvertRequest(t, api, body, ct)

	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestParseImageFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected processor.ImageFormat
		wantErr  bool
	}{
		{"jpeg", processor.FormatJPEG, false},
		{"png", processor.FormatPNG, false},
		{"webp", processor.FormatWEBP, false},
		{"gif", 0, true},
		{"", 0, true},
		{"bmp", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseImageFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseImageFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("parseImageFormat(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
