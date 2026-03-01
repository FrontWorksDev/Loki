package processor

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/chai2010/webp"
)

// createTestWEBP creates a test WebP image with the specified dimensions.
func createTestWEBP(t *testing.T, width, height int, quality float32) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / max(width, 1)),
				G: uint8(y * 255 / max(height, 1)),
				B: 128,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Quality: quality}); err != nil {
		t.Fatalf("failed to create test WebP: %v", err)
	}
	return buf.Bytes()
}

func TestNewWEBPProcessor(t *testing.T) {
	p := NewWEBPProcessor()
	if p == nil {
		t.Error("NewWEBPProcessor() returned nil")
	}
}

func TestWEBPProcessor_SupportedFormats(t *testing.T) {
	p := NewWEBPProcessor()
	formats := p.SupportedFormats()

	if len(formats) != 1 {
		t.Errorf("SupportedFormats() returned %d formats, want 1", len(formats))
	}
	if formats[0] != FormatWEBP {
		t.Errorf("SupportedFormats()[0] = %v, want %v", formats[0], FormatWEBP)
	}
}

func TestWEBPProcessor_Compress(t *testing.T) {
	p := NewWEBPProcessor()

	tests := []struct {
		name    string
		width   int
		height  int
		inQual  float32
		opts    CompressOptions
		wantErr bool
	}{
		{
			name:   "Default compression",
			width:  100,
			height: 100,
			inQual: 95,
			opts:   DefaultCompressOptions(),
		},
		{
			name:   "High quality",
			width:  100,
			height: 100,
			inQual: 95,
			opts: CompressOptions{
				Quality: 90,
				Level:   CompressionHigh,
			},
		},
		{
			name:   "Low quality",
			width:  100,
			height: 100,
			inQual: 95,
			opts: CompressOptions{
				Quality: 30,
				Level:   CompressionLow,
			},
		},
		{
			name:   "Use level when quality is 0",
			width:  100,
			height: 100,
			inQual: 95,
			opts: CompressOptions{
				Quality: 0,
				Level:   CompressionLow,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestWEBP(t, tt.width, tt.height, tt.inQual)
			reader := bytes.NewReader(input)
			var output bytes.Buffer

			result, err := p.Compress(context.Background(), reader, &output, tt.opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("Compress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if result.OriginalSize != int64(len(input)) {
					t.Errorf("OriginalSize = %d, want %d", result.OriginalSize, len(input))
				}
				if result.CompressedSize != int64(output.Len()) {
					t.Errorf("CompressedSize = %d, want %d", result.CompressedSize, output.Len())
				}
				if result.Format != FormatWEBP {
					t.Errorf("Format = %v, want %v", result.Format, FormatWEBP)
				}
				if output.Len() == 0 {
					t.Error("Output is empty")
				}

				// Verify output is valid WebP
				_, err := webp.Decode(bytes.NewReader(output.Bytes()))
				if err != nil {
					t.Errorf("Output is not valid WebP: %v", err)
				}
			}
		})
	}
}

func TestWEBPProcessor_Compress_InvalidInput(t *testing.T) {
	p := NewWEBPProcessor()
	reader := bytes.NewReader([]byte("not a valid image"))
	var output bytes.Buffer

	_, err := p.Compress(context.Background(), reader, &output, DefaultCompressOptions())
	if err == nil {
		t.Error("Compress() should return error for invalid input")
	}
}

func TestWEBPProcessor_Compress_ContextCanceled(t *testing.T) {
	p := NewWEBPProcessor()
	input := createTestWEBP(t, 100, 100, 95)
	reader := bytes.NewReader(input)
	var output bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := p.Compress(ctx, reader, &output, DefaultCompressOptions())
	if err == nil {
		t.Error("Compress() should return error for canceled context")
	}
}

func TestWEBPProcessor_Compress_ContextTimeout(t *testing.T) {
	p := NewWEBPProcessor()
	input := createTestWEBP(t, 100, 100, 95)
	reader := bytes.NewReader(input)
	var output bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure timeout

	_, err := p.Compress(ctx, reader, &output, DefaultCompressOptions())
	if err == nil {
		t.Error("Compress() should return error for timed out context")
	}
}

func TestWEBPProcessor_Convert(t *testing.T) {
	p := NewWEBPProcessor()

	tests := []struct {
		name        string
		createInput func(t *testing.T) []byte
		opts        ConvertOptions
		wantErr     bool
	}{
		{
			name: "Convert JPEG to WebP",
			createInput: func(t *testing.T) []byte {
				return createTestJPEG(t, 100, 100, 95)
			},
			opts: DefaultConvertOptions(FormatWEBP),
		},
		{
			name: "Convert PNG to WebP",
			createInput: func(t *testing.T) []byte {
				return createTestPNG(t, 100, 100)
			},
			opts: DefaultConvertOptions(FormatWEBP),
		},
		{
			name: "Convert WebP to WebP (re-encode)",
			createInput: func(t *testing.T) []byte {
				return createTestWEBP(t, 100, 100, 95)
			},
			opts: ConvertOptions{
				Format: FormatWEBP,
				CompressOptions: CompressOptions{
					Quality: 80,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.createInput(t)
			reader := bytes.NewReader(input)
			var output bytes.Buffer

			result, err := p.Convert(context.Background(), reader, &output, tt.opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("Convert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if result.OriginalSize != int64(len(input)) {
					t.Errorf("OriginalSize = %d, want %d", result.OriginalSize, len(input))
				}
				if result.Format != FormatWEBP {
					t.Errorf("Format = %v, want %v", result.Format, FormatWEBP)
				}
				if output.Len() == 0 {
					t.Error("Output is empty")
				}

				// Verify output is valid WebP
				_, err := webp.Decode(bytes.NewReader(output.Bytes()))
				if err != nil {
					t.Errorf("Output is not valid WebP: %v", err)
				}
			}
		})
	}
}

func TestWEBPProcessor_Convert_InvalidInput(t *testing.T) {
	p := NewWEBPProcessor()
	reader := bytes.NewReader([]byte("this is not a valid image"))
	var output bytes.Buffer

	_, err := p.Convert(context.Background(), reader, &output, DefaultConvertOptions(FormatWEBP))
	if err == nil {
		t.Fatal("Convert() should return error for invalid input")
	}
}

func TestWEBPProcessor_Convert_FormatMismatch(t *testing.T) {
	p := NewWEBPProcessor()
	input := createTestJPEG(t, 100, 100, 95)
	reader := bytes.NewReader(input)
	var output bytes.Buffer

	// Try to convert with JPEG format (should fail for WebP processor)
	opts := ConvertOptions{
		Format:          FormatJPEG,
		CompressOptions: DefaultCompressOptions(),
	}

	_, err := p.Convert(context.Background(), reader, &output, opts)
	if err == nil {
		t.Error("Convert() should return error for mismatched format")
	}
}

func TestWEBPProcessor_Compress_ReadError(t *testing.T) {
	p := NewWEBPProcessor()
	readErr := errors.New("simulated read error")
	r := &errReader{err: readErr}
	var output bytes.Buffer

	_, err := p.Compress(context.Background(), r, &output, DefaultCompressOptions())
	if err == nil {
		t.Fatal("Compress() should return error for read failure")
	}
	if !errors.Is(err, readErr) {
		t.Errorf("error should wrap readErr, got: %v", err)
	}
}

func TestWEBPProcessor_Compress_WriteError(t *testing.T) {
	p := NewWEBPProcessor()
	input := createTestWEBP(t, 50, 50, 95)
	reader := bytes.NewReader(input)
	writeErr := errors.New("simulated write error")
	w := &errWriter{err: writeErr}

	_, err := p.Compress(context.Background(), reader, w, DefaultCompressOptions())
	if err == nil {
		t.Fatal("Compress() should return error for write failure")
	}
	if !errors.Is(err, writeErr) {
		t.Errorf("error should wrap writeErr, got: %v", err)
	}
}

func TestWEBPProcessor_Convert_ReadError(t *testing.T) {
	p := NewWEBPProcessor()
	readErr := errors.New("simulated read error")
	r := &errReader{err: readErr}
	var output bytes.Buffer

	_, err := p.Convert(context.Background(), r, &output, DefaultConvertOptions(FormatWEBP))
	if err == nil {
		t.Fatal("Convert() should return error for read failure")
	}
	if !errors.Is(err, readErr) {
		t.Errorf("error should wrap readErr, got: %v", err)
	}
}

func TestWEBPProcessor_Convert_WriteError(t *testing.T) {
	p := NewWEBPProcessor()
	input := createTestJPEG(t, 50, 50, 95)
	reader := bytes.NewReader(input)
	writeErr := errors.New("simulated write error")
	w := &errWriter{err: writeErr}

	_, err := p.Convert(context.Background(), reader, w, DefaultConvertOptions(FormatWEBP))
	if err == nil {
		t.Fatal("Convert() should return error for write failure")
	}
	if !errors.Is(err, writeErr) {
		t.Errorf("error should wrap writeErr, got: %v", err)
	}
}

func TestWEBPProcessor_Compress_QualityBounds(t *testing.T) {
	p := NewWEBPProcessor()

	tests := []struct {
		name    string
		quality int
	}{
		{"Quality below minimum", -10},
		{"Quality at minimum", 1},
		{"Quality above maximum", 150},
		{"Quality at maximum", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestWEBP(t, 50, 50, 95)
			reader := bytes.NewReader(input)
			var output bytes.Buffer

			opts := CompressOptions{Quality: tt.quality}
			result, err := p.Compress(context.Background(), reader, &output, opts)

			if err != nil {
				t.Errorf("Compress() error = %v", err)
				return
			}

			if result.CompressedSize == 0 {
				t.Error("CompressedSize should not be 0")
			}
		})
	}
}

func TestWEBPProcessor_Compress_PreserveMetadata(t *testing.T) {
	p := NewWEBPProcessor()
	input := createTestWEBP(t, 50, 50, 95)
	reader := bytes.NewReader(input)
	var output bytes.Buffer

	opts := CompressOptions{
		Quality:          80,
		PreserveMetadata: true,
	}
	_, err := p.Compress(context.Background(), reader, &output, opts)
	if err == nil {
		t.Error("Compress() should return error when PreserveMetadata is true")
	}
	if err != ErrPreserveMetadataNotSupported {
		t.Errorf("Compress() error = %v, want %v", err, ErrPreserveMetadataNotSupported)
	}
}

func TestWEBPProcessor_Compress_MaxFileSize(t *testing.T) {
	p := NewWEBPProcessor()
	input := createTestWEBP(t, 100, 100, 95)

	t.Run("Within limit", func(t *testing.T) {
		reader := bytes.NewReader(input)
		var output bytes.Buffer
		opts := CompressOptions{
			MaxFileSize: int64(len(input) + 1),
		}
		result, err := p.Compress(context.Background(), reader, &output, opts)
		if err != nil {
			t.Fatalf("Compress() error = %v", err)
		}
		if result.OriginalSize != int64(len(input)) {
			t.Errorf("OriginalSize = %d, want %d", result.OriginalSize, len(input))
		}
	})

	t.Run("Exceeds limit", func(t *testing.T) {
		reader := bytes.NewReader(input)
		var output bytes.Buffer
		opts := CompressOptions{
			MaxFileSize: 1, // Very small limit
		}
		_, err := p.Compress(context.Background(), reader, &output, opts)
		if err == nil {
			t.Fatal("Compress() should return error when file exceeds MaxFileSize")
		}
	})
}
