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
		{
			name: "Convert WebP to JPEG",
			createInput: func(t *testing.T) []byte {
				return createTestWEBP(t, 100, 100, 95)
			},
			opts: DefaultConvertOptions(FormatJPEG),
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

// cancelAfterReadReader wraps a reader and cancels the context after all data is read.
type cancelAfterReadReader struct {
	r      *bytes.Reader
	cancel context.CancelFunc
}

func (r *cancelAfterReadReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if err != nil {
		r.cancel()
	}
	return n, err
}

func TestJPEGProcessor_Compress_ContextCanceledAfterRead(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestJPEG(t, 50, 50, 95)
	ctx, cancel := context.WithCancel(context.Background())
	reader := &cancelAfterReadReader{r: bytes.NewReader(input), cancel: cancel}
	var output bytes.Buffer

	_, err := p.Compress(ctx, reader, &output, DefaultCompressOptions())
	if err == nil {
		t.Error("Compress() should return error when context is canceled after read")
	}
}

func TestJPEGProcessor_Convert_ContextCanceledAfterRead(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestPNG(t, 50, 50)
	ctx, cancel := context.WithCancel(context.Background())
	reader := &cancelAfterReadReader{r: bytes.NewReader(input), cancel: cancel}
	var output bytes.Buffer

	_, err := p.Convert(ctx, reader, &output, DefaultConvertOptions(FormatJPEG))
	if err == nil {
		t.Error("Convert() should return error when context is canceled after read")
	}
}

// slowReader wraps a reader and introduces a delay after a certain number of bytes,
// then cancels the context. This allows testing context cancellation between decode and encode.
type slowReader struct {
	r         *bytes.Reader
	cancel    context.CancelFunc
	threshold int
	read      int
	canceled  bool
}

func (r *slowReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.read += n
	if !r.canceled && r.read >= r.threshold {
		r.cancel()
		r.canceled = true
	}
	return n, err
}

func TestJPEGProcessor_Compress_ContextCanceledAfterDecode(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestJPEG(t, 50, 50, 95)
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after reading most of the data (after decode will succeed)
	reader := &slowReader{r: bytes.NewReader(input), cancel: cancel, threshold: len(input) - 1}
	var output bytes.Buffer

	_, err := p.Compress(ctx, reader, &output, DefaultCompressOptions())
	if err == nil {
		t.Error("Compress() should return error when context is canceled after decode")
	}
}

func TestJPEGProcessor_Convert_ContextCanceledAfterDecode(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestPNG(t, 50, 50)
	ctx, cancel := context.WithCancel(context.Background())
	reader := &slowReader{r: bytes.NewReader(input), cancel: cancel, threshold: len(input) - 1}
	var output bytes.Buffer

	_, err := p.Convert(ctx, reader, &output, DefaultConvertOptions(FormatJPEG))
	if err == nil {
		t.Error("Convert() should return error when context is canceled after decode")
	}
}

func TestJPEGProcessor_Convert_QualityBounds(t *testing.T) {
	p := NewJPEGProcessor()

	tests := []struct {
		name    string
		quality int
	}{
		{"Quality below minimum", -10},
		{"Quality above maximum", 150},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestPNG(t, 50, 50)
			reader := bytes.NewReader(input)
			var output bytes.Buffer

			opts := ConvertOptions{
				Format: FormatJPEG,
				CompressOptions: CompressOptions{
					Quality: tt.quality,
				},
			}
			result, err := p.Convert(context.Background(), reader, &output, opts)
			if err != nil {
				t.Errorf("Convert() error = %v", err)
				return
			}
			if result.CompressedSize == 0 {
				t.Error("CompressedSize should not be 0")
			}
		})
	}
}

func TestJPEGProcessor_Compress_MaxFileSize(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestJPEG(t, 100, 100, 95)

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

func TestJPEGProcessor_Compress_PreserveMetadata(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestJPEG(t, 50, 50, 95)
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

func TestJPEGProcessor_Convert_MaxFileSize(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestPNG(t, 50, 50)

	t.Run("Exceeds limit", func(t *testing.T) {
		reader := bytes.NewReader(input)
		var output bytes.Buffer
		opts := ConvertOptions{
			Format: FormatJPEG,
			CompressOptions: CompressOptions{
				MaxFileSize: 1,
			},
		}
		_, err := p.Convert(context.Background(), reader, &output, opts)
		if err == nil {
			t.Fatal("Convert() should return error when file exceeds MaxFileSize")
		}
	})
}

func TestJPEGProcessor_Convert_PreserveMetadata(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestPNG(t, 50, 50)
	reader := bytes.NewReader(input)
	var output bytes.Buffer

	opts := ConvertOptions{
		Format: FormatJPEG,
		CompressOptions: CompressOptions{
			Quality:          80,
			PreserveMetadata: true,
		},
	}
	_, err := p.Convert(context.Background(), reader, &output, opts)
	if err == nil {
		t.Error("Convert() should return error when PreserveMetadata is true")
	}
	if err != ErrPreserveMetadataNotSupported {
		t.Errorf("Convert() error = %v, want %v", err, ErrPreserveMetadataNotSupported)
	}
}

func TestJPEGProcessor_Convert_FormatMismatch(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestPNG(t, 100, 100)
	reader := bytes.NewReader(input)
	var output bytes.Buffer

	// Try to convert with PNG format (should fail for JPEG processor)
	opts := ConvertOptions{
		Format:          FormatPNG,
		CompressOptions: DefaultCompressOptions(),
	}

	_, err := p.Convert(context.Background(), reader, &output, opts)
	if err == nil {
		t.Error("Convert() should return error for mismatched format")
	}
}
