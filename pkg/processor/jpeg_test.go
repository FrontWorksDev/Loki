package processor

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
	"time"
)

// createTestJPEG creates a test JPEG image with the specified dimensions.
func createTestJPEG(t *testing.T, width, height int, quality int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / width),
				G: uint8(y * 255 / height),
				B: 128,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatalf("failed to create test JPEG: %v", err)
	}
	return buf.Bytes()
}

// createTestPNG creates a test PNG image with the specified dimensions.
func createTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / width),
				G: uint8(y * 255 / height),
				B: 128,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	return buf.Bytes()
}

func TestNewJPEGProcessor(t *testing.T) {
	p := NewJPEGProcessor()
	if p == nil {
		t.Error("NewJPEGProcessor() returned nil")
	}
}

func TestJPEGProcessor_SupportedFormats(t *testing.T) {
	p := NewJPEGProcessor()
	formats := p.SupportedFormats()

	if len(formats) != 1 {
		t.Errorf("SupportedFormats() returned %d formats, want 1", len(formats))
	}
	if formats[0] != FormatJPEG {
		t.Errorf("SupportedFormats()[0] = %v, want %v", formats[0], FormatJPEG)
	}
}

func TestJPEGProcessor_Compress(t *testing.T) {
	p := NewJPEGProcessor()

	tests := []struct {
		name    string
		width   int
		height  int
		inQual  int
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
			input := createTestJPEG(t, tt.width, tt.height, tt.inQual)
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
				if result.Format != FormatJPEG {
					t.Errorf("Format = %v, want %v", result.Format, FormatJPEG)
				}
				if output.Len() == 0 {
					t.Error("Output is empty")
				}

				// Verify output is valid JPEG
				_, err := jpeg.Decode(&output)
				if err != nil {
					t.Errorf("Output is not valid JPEG: %v", err)
				}
			}
		})
	}
}

func TestJPEGProcessor_Compress_InvalidInput(t *testing.T) {
	p := NewJPEGProcessor()
	reader := bytes.NewReader([]byte("not a valid image"))
	var output bytes.Buffer

	_, err := p.Compress(context.Background(), reader, &output, DefaultCompressOptions())
	if err == nil {
		t.Error("Compress() should return error for invalid input")
	}
}

func TestJPEGProcessor_Compress_ContextCanceled(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestJPEG(t, 100, 100, 95)
	reader := bytes.NewReader(input)
	var output bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := p.Compress(ctx, reader, &output, DefaultCompressOptions())
	if err == nil {
		t.Error("Compress() should return error for canceled context")
	}
}

func TestJPEGProcessor_Convert(t *testing.T) {
	p := NewJPEGProcessor()

	tests := []struct {
		name        string
		createInput func(t *testing.T) []byte
		opts        ConvertOptions
		wantErr     bool
	}{
		{
			name: "Convert PNG to JPEG",
			createInput: func(t *testing.T) []byte {
				return createTestPNG(t, 100, 100)
			},
			opts: DefaultConvertOptions(FormatJPEG),
		},
		{
			name: "Convert JPEG to JPEG (re-encode)",
			createInput: func(t *testing.T) []byte {
				return createTestJPEG(t, 100, 100, 95)
			},
			opts: ConvertOptions{
				Format: FormatJPEG,
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
				if result.Format != FormatJPEG {
					t.Errorf("Format = %v, want %v", result.Format, FormatJPEG)
				}
				if output.Len() == 0 {
					t.Error("Output is empty")
				}

				// Verify output is valid JPEG
				outputCopy := bytes.NewReader(output.Bytes())
				_, err := jpeg.Decode(outputCopy)
				if err != nil {
					t.Errorf("Output is not valid JPEG: %v", err)
				}
			}
		})
	}
}

func TestJPEGProcessor_Convert_ContextTimeout(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestPNG(t, 100, 100)
	reader := bytes.NewReader(input)
	var output bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure timeout

	_, err := p.Convert(ctx, reader, &output, DefaultConvertOptions(FormatJPEG))
	if err == nil {
		t.Error("Convert() should return error for timed out context")
	}
}

func TestJPEGProcessor_Compress_QualityBounds(t *testing.T) {
	p := NewJPEGProcessor()

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
			input := createTestJPEG(t, 50, 50, 95)
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
