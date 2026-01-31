package processor

import (
	"bytes"
	"context"
	"image/png"
	"testing"
	"time"
)

func TestNewPNGProcessor(t *testing.T) {
	p := NewPNGProcessor()
	if p == nil {
		t.Error("NewPNGProcessor() returned nil")
	}
}

func TestPNGProcessor_SupportedFormats(t *testing.T) {
	p := NewPNGProcessor()
	formats := p.SupportedFormats()

	if len(formats) != 1 {
		t.Errorf("SupportedFormats() returned %d formats, want 1", len(formats))
	}
	if formats[0] != FormatPNG {
		t.Errorf("SupportedFormats()[0] = %v, want %v", formats[0], FormatPNG)
	}
}

func TestPNGProcessor_Compress(t *testing.T) {
	p := NewPNGProcessor()

	tests := []struct {
		name    string
		width   int
		height  int
		opts    CompressOptions
		wantErr bool
	}{
		{
			name:   "Default compression",
			width:  100,
			height: 100,
			opts:   DefaultCompressOptions(),
		},
		{
			name:   "Best compression",
			width:  100,
			height: 100,
			opts: CompressOptions{
				Level: CompressionHigh,
			},
		},
		{
			name:   "Best speed",
			width:  100,
			height: 100,
			opts: CompressOptions{
				Level: CompressionLow,
			},
		},
		{
			name:   "Small image",
			width:  10,
			height: 10,
			opts:   DefaultCompressOptions(),
		},
		{
			name:   "Large image",
			width:  500,
			height: 500,
			opts: CompressOptions{
				Level: CompressionMedium,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestPNG(t, tt.width, tt.height)
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
				if result.Format != FormatPNG {
					t.Errorf("Format = %v, want %v", result.Format, FormatPNG)
				}
				if output.Len() == 0 {
					t.Error("Output is empty")
				}

				// Verify output is valid PNG
				_, err := png.Decode(&output)
				if err != nil {
					t.Errorf("Output is not valid PNG: %v", err)
				}
			}
		})
	}
}

func TestPNGProcessor_Compress_InvalidInput(t *testing.T) {
	p := NewPNGProcessor()
	reader := bytes.NewReader([]byte("not a valid image"))
	var output bytes.Buffer

	_, err := p.Compress(context.Background(), reader, &output, DefaultCompressOptions())
	if err == nil {
		t.Error("Compress() should return error for invalid input")
	}
}

func TestPNGProcessor_Compress_ContextCanceled(t *testing.T) {
	p := NewPNGProcessor()
	input := createTestPNG(t, 100, 100)
	reader := bytes.NewReader(input)
	var output bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := p.Compress(ctx, reader, &output, DefaultCompressOptions())
	if err == nil {
		t.Error("Compress() should return error for canceled context")
	}
}

func TestPNGProcessor_Convert(t *testing.T) {
	p := NewPNGProcessor()

	tests := []struct {
		name        string
		createInput func(t *testing.T) []byte
		opts        ConvertOptions
		wantErr     bool
	}{
		{
			name: "Convert JPEG to PNG",
			createInput: func(t *testing.T) []byte {
				return createTestJPEG(t, 100, 100, 95)
			},
			opts: DefaultConvertOptions(FormatPNG),
		},
		{
			name: "Convert PNG to PNG (re-encode)",
			createInput: func(t *testing.T) []byte {
				return createTestPNG(t, 100, 100)
			},
			opts: ConvertOptions{
				Format: FormatPNG,
				CompressOptions: CompressOptions{
					Level: CompressionHigh,
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
				if result.Format != FormatPNG {
					t.Errorf("Format = %v, want %v", result.Format, FormatPNG)
				}
				if output.Len() == 0 {
					t.Error("Output is empty")
				}

				// Verify output is valid PNG
				outputCopy := bytes.NewReader(output.Bytes())
				_, err := png.Decode(outputCopy)
				if err != nil {
					t.Errorf("Output is not valid PNG: %v", err)
				}
			}
		})
	}
}

func TestPNGProcessor_Convert_ContextTimeout(t *testing.T) {
	p := NewPNGProcessor()
	input := createTestJPEG(t, 100, 100, 95)
	reader := bytes.NewReader(input)
	var output bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure timeout

	_, err := p.Convert(ctx, reader, &output, DefaultConvertOptions(FormatPNG))
	if err == nil {
		t.Error("Convert() should return error for timed out context")
	}
}

func TestPNGProcessor_Compress_CompressionLevels(t *testing.T) {
	p := NewPNGProcessor()
	input := createTestPNG(t, 200, 200)

	levels := []CompressionLevel{
		CompressionLow,
		CompressionMedium,
		CompressionHigh,
	}

	var sizes []int64
	for _, level := range levels {
		reader := bytes.NewReader(input)
		var output bytes.Buffer

		opts := CompressOptions{Level: level}
		result, err := p.Compress(context.Background(), reader, &output, opts)
		if err != nil {
			t.Errorf("Compress() with level %v error = %v", level, err)
			continue
		}
		sizes = append(sizes, result.CompressedSize)
	}

	// Higher compression should generally produce smaller files
	// (though this isn't always guaranteed with PNG)
	t.Logf("Compression sizes: Low=%d, Medium=%d, High=%d", sizes[0], sizes[1], sizes[2])
}
