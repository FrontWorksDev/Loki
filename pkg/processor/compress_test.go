package processor

import (
	"bytes"
	"context"
	"errors"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/chai2010/webp"
)

// errReader is an io.Reader that always returns an error.
type errReader struct {
	err error
}

func (r *errReader) Read([]byte) (int, error) {
	return 0, r.err
}

// errWriter is an io.Writer that always returns an error.
type errWriter struct {
	err error
}

func (w *errWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestCompress_JPEGCompressionLevels(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestJPEG(t, 200, 200, 100)

	levels := []CompressionLevel{
		CompressionLow,
		CompressionMedium,
		CompressionHigh,
	}

	sizes := make(map[CompressionLevel]int64)
	for _, level := range levels {
		reader := bytes.NewReader(input)
		var output bytes.Buffer

		opts := CompressOptions{Level: level}
		result, err := p.Compress(context.Background(), reader, &output, opts)
		if err != nil {
			t.Fatalf("Compress() with level %v error = %v", level, err)
		}

		if result.CompressedSize <= 0 {
			t.Errorf("CompressedSize for level %v should be positive, got %d", level, result.CompressedSize)
		}
		if result.OriginalSize != int64(len(input)) {
			t.Errorf("OriginalSize = %d, want %d", result.OriginalSize, len(input))
		}
		if result.Format != FormatJPEG {
			t.Errorf("Format = %v, want %v", result.Format, FormatJPEG)
		}

		sizes[level] = result.CompressedSize
	}

	// JPEG: Higher CompressionLevel maps to higher quality (Low=60, Medium=75, High=90),
	// so higher levels produce LARGER files.
	if sizes[CompressionLow] >= sizes[CompressionMedium] {
		t.Errorf("Low (%d) should be smaller than Medium (%d)", sizes[CompressionLow], sizes[CompressionMedium])
	}
	if sizes[CompressionMedium] >= sizes[CompressionHigh] {
		t.Errorf("Medium (%d) should be smaller than High (%d)", sizes[CompressionMedium], sizes[CompressionHigh])
	}
}

func TestCompress_PNGCompressionLevels_OutputSize(t *testing.T) {
	p := NewPNGProcessor()
	input := createTestPNG(t, 200, 200)

	levels := []CompressionLevel{
		CompressionLow,
		CompressionMedium,
		CompressionHigh,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			reader := bytes.NewReader(input)
			var output bytes.Buffer

			opts := CompressOptions{Level: level}
			result, err := p.Compress(context.Background(), reader, &output, opts)
			if err != nil {
				t.Fatalf("Compress() with level %v error = %v", level, err)
			}

			if result.CompressedSize <= 0 {
				t.Errorf("CompressedSize for level %v should be positive, got %d", level, result.CompressedSize)
			}
			if result.OriginalSize != int64(len(input)) {
				t.Errorf("OriginalSize = %d, want %d", result.OriginalSize, len(input))
			}
			if result.Format != FormatPNG {
				t.Errorf("Format = %v, want %v", result.Format, FormatPNG)
			}
			if result.CompressedSize != int64(output.Len()) {
				t.Errorf("CompressedSize (%d) != output length (%d)", result.CompressedSize, output.Len())
			}
		})
	}
}

func TestCompress_JPEG_ReadError(t *testing.T) {
	p := NewJPEGProcessor()
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

func TestCompress_JPEG_WriteError(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestJPEG(t, 50, 50, 95)
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

func TestCompress_PNG_ReadError(t *testing.T) {
	p := NewPNGProcessor()
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

func TestCompress_PNG_WriteError(t *testing.T) {
	p := NewPNGProcessor()
	input := createTestPNG(t, 50, 50)
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

func TestConvert_JPEG_InvalidInput(t *testing.T) {
	p := NewJPEGProcessor()
	reader := bytes.NewReader([]byte("this is not a valid image"))
	var output bytes.Buffer

	_, err := p.Convert(context.Background(), reader, &output, DefaultConvertOptions(FormatJPEG))
	if err == nil {
		t.Fatal("Convert() should return error for invalid input")
	}
}

func TestConvert_PNG_InvalidInput(t *testing.T) {
	p := NewPNGProcessor()
	reader := bytes.NewReader([]byte("this is not a valid image"))
	var output bytes.Buffer

	_, err := p.Convert(context.Background(), reader, &output, DefaultConvertOptions(FormatPNG))
	if err == nil {
		t.Fatal("Convert() should return error for invalid input")
	}
}

func TestConvert_JPEG_ReadError(t *testing.T) {
	p := NewJPEGProcessor()
	readErr := errors.New("simulated read error")
	r := &errReader{err: readErr}
	var output bytes.Buffer

	_, err := p.Convert(context.Background(), r, &output, DefaultConvertOptions(FormatJPEG))
	if err == nil {
		t.Fatal("Convert() should return error for read failure")
	}
	if !errors.Is(err, readErr) {
		t.Errorf("error should wrap readErr, got: %v", err)
	}
}

func TestConvert_JPEG_WriteError(t *testing.T) {
	p := NewJPEGProcessor()
	input := createTestPNG(t, 50, 50)
	reader := bytes.NewReader(input)
	writeErr := errors.New("simulated write error")
	w := &errWriter{err: writeErr}

	_, err := p.Convert(context.Background(), reader, w, DefaultConvertOptions(FormatJPEG))
	if err == nil {
		t.Fatal("Convert() should return error for write failure")
	}
	if !errors.Is(err, writeErr) {
		t.Errorf("error should wrap writeErr, got: %v", err)
	}
}

func TestConvert_PNG_ReadError(t *testing.T) {
	p := NewPNGProcessor()
	readErr := errors.New("simulated read error")
	r := &errReader{err: readErr}
	var output bytes.Buffer

	_, err := p.Convert(context.Background(), r, &output, DefaultConvertOptions(FormatPNG))
	if err == nil {
		t.Fatal("Convert() should return error for read failure")
	}
	if !errors.Is(err, readErr) {
		t.Errorf("error should wrap readErr, got: %v", err)
	}
}

func TestConvert_PNG_WriteError(t *testing.T) {
	p := NewPNGProcessor()
	input := createTestJPEG(t, 50, 50, 95)
	reader := bytes.NewReader(input)
	writeErr := errors.New("simulated write error")
	w := &errWriter{err: writeErr}

	_, err := p.Convert(context.Background(), reader, w, DefaultConvertOptions(FormatPNG))
	if err == nil {
		t.Fatal("Convert() should return error for write failure")
	}
	if !errors.Is(err, writeErr) {
		t.Errorf("error should wrap writeErr, got: %v", err)
	}
}

func TestCompress_AllFormats(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		processor Processor
		input     func(t *testing.T) []byte
		level     CompressionLevel
		validate  func(t *testing.T, data []byte)
	}{
		{
			name:      "JPEG/Low",
			format:    "jpeg",
			processor: NewJPEGProcessor(),
			input:     func(t *testing.T) []byte { return createTestJPEG(t, 100, 100, 95) },
			level:     CompressionLow,
			validate: func(t *testing.T, data []byte) {
				_, err := jpeg.Decode(bytes.NewReader(data))
				if err != nil {
					t.Errorf("output is not valid JPEG: %v", err)
				}
			},
		},
		{
			name:      "JPEG/Medium",
			format:    "jpeg",
			processor: NewJPEGProcessor(),
			input:     func(t *testing.T) []byte { return createTestJPEG(t, 100, 100, 95) },
			level:     CompressionMedium,
			validate: func(t *testing.T, data []byte) {
				_, err := jpeg.Decode(bytes.NewReader(data))
				if err != nil {
					t.Errorf("output is not valid JPEG: %v", err)
				}
			},
		},
		{
			name:      "JPEG/High",
			format:    "jpeg",
			processor: NewJPEGProcessor(),
			input:     func(t *testing.T) []byte { return createTestJPEG(t, 100, 100, 95) },
			level:     CompressionHigh,
			validate: func(t *testing.T, data []byte) {
				_, err := jpeg.Decode(bytes.NewReader(data))
				if err != nil {
					t.Errorf("output is not valid JPEG: %v", err)
				}
			},
		},
		{
			name:      "PNG/Low",
			format:    "png",
			processor: NewPNGProcessor(),
			input:     func(t *testing.T) []byte { return createTestPNG(t, 100, 100) },
			level:     CompressionLow,
			validate: func(t *testing.T, data []byte) {
				_, err := png.Decode(bytes.NewReader(data))
				if err != nil {
					t.Errorf("output is not valid PNG: %v", err)
				}
			},
		},
		{
			name:      "PNG/Medium",
			format:    "png",
			processor: NewPNGProcessor(),
			input:     func(t *testing.T) []byte { return createTestPNG(t, 100, 100) },
			level:     CompressionMedium,
			validate: func(t *testing.T, data []byte) {
				_, err := png.Decode(bytes.NewReader(data))
				if err != nil {
					t.Errorf("output is not valid PNG: %v", err)
				}
			},
		},
		{
			name:      "PNG/High",
			format:    "png",
			processor: NewPNGProcessor(),
			input:     func(t *testing.T) []byte { return createTestPNG(t, 100, 100) },
			level:     CompressionHigh,
			validate: func(t *testing.T, data []byte) {
				_, err := png.Decode(bytes.NewReader(data))
				if err != nil {
					t.Errorf("output is not valid PNG: %v", err)
				}
			},
		},
		{
			name:      "WebP/Low",
			format:    "webp",
			processor: NewWEBPProcessor(),
			input:     func(t *testing.T) []byte { return createTestWEBP(t, 100, 100, 95) },
			level:     CompressionLow,
			validate: func(t *testing.T, data []byte) {
				_, err := webp.Decode(bytes.NewReader(data))
				if err != nil {
					t.Errorf("output is not valid WebP: %v", err)
				}
			},
		},
		{
			name:      "WebP/Medium",
			format:    "webp",
			processor: NewWEBPProcessor(),
			input:     func(t *testing.T) []byte { return createTestWEBP(t, 100, 100, 95) },
			level:     CompressionMedium,
			validate: func(t *testing.T, data []byte) {
				_, err := webp.Decode(bytes.NewReader(data))
				if err != nil {
					t.Errorf("output is not valid WebP: %v", err)
				}
			},
		},
		{
			name:      "WebP/High",
			format:    "webp",
			processor: NewWEBPProcessor(),
			input:     func(t *testing.T) []byte { return createTestWEBP(t, 100, 100, 95) },
			level:     CompressionHigh,
			validate: func(t *testing.T, data []byte) {
				_, err := webp.Decode(bytes.NewReader(data))
				if err != nil {
					t.Errorf("output is not valid WebP: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input(t)
			reader := bytes.NewReader(input)
			var output bytes.Buffer

			opts := CompressOptions{Level: tt.level}
			result, err := tt.processor.Compress(context.Background(), reader, &output, opts)
			if err != nil {
				t.Fatalf("Compress() error = %v", err)
			}

			if result.OriginalSize != int64(len(input)) {
				t.Errorf("OriginalSize = %d, want %d", result.OriginalSize, len(input))
			}
			if result.CompressedSize <= 0 {
				t.Errorf("CompressedSize should be positive, got %d", result.CompressedSize)
			}
			if output.Len() == 0 {
				t.Error("output should not be empty")
			}

			tt.validate(t, output.Bytes())
		})
	}
}
