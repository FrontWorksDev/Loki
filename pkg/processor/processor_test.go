package processor

import (
	"errors"
	"testing"
)

func TestDefaultCompressOptions(t *testing.T) {
	opts := DefaultCompressOptions()

	if opts.Quality != 0 {
		t.Errorf("DefaultCompressOptions().Quality = %v, want 0", opts.Quality)
	}
	if opts.Level != CompressionMedium {
		t.Errorf("DefaultCompressOptions().Level = %v, want %v", opts.Level, CompressionMedium)
	}
	if opts.PreserveMetadata != false {
		t.Errorf("DefaultCompressOptions().PreserveMetadata = %v, want false", opts.PreserveMetadata)
	}
}

func TestDefaultConvertOptions(t *testing.T) {
	tests := []struct {
		name   string
		format ImageFormat
	}{
		{"JPEG format", FormatJPEG},
		{"PNG format", FormatPNG},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultConvertOptions(tt.format)

			if opts.Format != tt.format {
				t.Errorf("DefaultConvertOptions().Format = %v, want %v", opts.Format, tt.format)
			}
			if opts.Quality != 0 {
				t.Errorf("DefaultConvertOptions().Quality = %v, want 0", opts.Quality)
			}
			if opts.Level != CompressionMedium {
				t.Errorf("DefaultConvertOptions().Level = %v, want %v", opts.Level, CompressionMedium)
			}
		})
	}
}

func TestResult_CompressionRatio(t *testing.T) {
	tests := []struct {
		name           string
		originalSize   int64
		compressedSize int64
		expected       float64
	}{
		{"50% compression", 1000, 500, 50.0},
		{"100% (no compression)", 1000, 1000, 100.0},
		{"25% compression", 1000, 250, 25.0},
		{"Zero original size", 0, 0, 0.0},
		{"Larger than original", 1000, 1200, 120.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{
				OriginalSize:   tt.originalSize,
				CompressedSize: tt.compressedSize,
			}
			if got := r.CompressionRatio(); got != tt.expected {
				t.Errorf("Result.CompressionRatio() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResult_SavedBytes(t *testing.T) {
	tests := []struct {
		name           string
		originalSize   int64
		compressedSize int64
		expected       int64
	}{
		{"Positive savings", 1000, 600, 400},
		{"No savings", 1000, 1000, 0},
		{"Negative savings (larger)", 1000, 1200, -200},
		{"Zero sizes", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{
				OriginalSize:   tt.originalSize,
				CompressedSize: tt.compressedSize,
			}
			if got := r.SavedBytes(); got != tt.expected {
				t.Errorf("Result.SavedBytes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResult_SavedPercentage(t *testing.T) {
	tests := []struct {
		name           string
		originalSize   int64
		compressedSize int64
		expected       float64
	}{
		{"40% savings", 1000, 600, 40.0},
		{"No savings", 1000, 1000, 0.0},
		{"Zero original size", 0, 0, 0.0},
		{"Negative savings", 1000, 1200, -20.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{
				OriginalSize:   tt.originalSize,
				CompressedSize: tt.compressedSize,
			}
			if got := r.SavedPercentage(); got != tt.expected {
				t.Errorf("Result.SavedPercentage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBatchResult_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		result   *Result
		err      error
		expected bool
	}{
		{
			name: "Success with result",
			result: &Result{
				OriginalSize:   1000,
				CompressedSize: 500,
				Format:         FormatJPEG,
			},
			err:      nil,
			expected: true,
		},
		{
			name:     "Failure with error",
			result:   nil,
			err:      errors.New("processing failed"),
			expected: false,
		},
		{
			name:     "Failure with nil result and nil error",
			result:   nil,
			err:      nil,
			expected: false,
		},
		{
			name: "Failure with result and error",
			result: &Result{
				OriginalSize:   1000,
				CompressedSize: 500,
			},
			err:      errors.New("partial failure"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			br := &BatchResult{
				Item:   BatchItem{InputPath: "/test/input.jpg", OutputPath: "/test/output.jpg"},
				Result: tt.result,
				Error:  tt.err,
			}
			if got := br.IsSuccess(); got != tt.expected {
				t.Errorf("BatchResult.IsSuccess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompressOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    CompressOptions
		wantErr error
	}{
		{
			name:    "Default options are valid",
			opts:    DefaultCompressOptions(),
			wantErr: nil,
		},
		{
			name: "PreserveMetadata false is valid",
			opts: CompressOptions{
				Quality:          80,
				Level:            CompressionMedium,
				PreserveMetadata: false,
			},
			wantErr: nil,
		},
		{
			name: "PreserveMetadata true returns error",
			opts: CompressOptions{
				Quality:          80,
				Level:            CompressionMedium,
				PreserveMetadata: true,
			},
			wantErr: ErrPreserveMetadataNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompressOptions_Customization(t *testing.T) {
	opts := CompressOptions{
		Quality:          85,
		Level:            CompressionHigh,
		PreserveMetadata: true,
	}

	if opts.Quality != 85 {
		t.Errorf("CompressOptions.Quality = %v, want 85", opts.Quality)
	}
	if opts.Level != CompressionHigh {
		t.Errorf("CompressOptions.Level = %v, want %v", opts.Level, CompressionHigh)
	}
	if opts.PreserveMetadata != true {
		t.Errorf("CompressOptions.PreserveMetadata = %v, want true", opts.PreserveMetadata)
	}
}

func TestConvertOptions_Customization(t *testing.T) {
	opts := ConvertOptions{
		Format: FormatPNG,
		CompressOptions: CompressOptions{
			Quality:          90,
			Level:            CompressionLow,
			PreserveMetadata: false,
		},
	}

	if opts.Format != FormatPNG {
		t.Errorf("ConvertOptions.Format = %v, want %v", opts.Format, FormatPNG)
	}
	if opts.Quality != 90 {
		t.Errorf("ConvertOptions.Quality = %v, want 90", opts.Quality)
	}
	if opts.Level != CompressionLow {
		t.Errorf("ConvertOptions.Level = %v, want %v", opts.Level, CompressionLow)
	}
}

func TestBatchItem_Fields(t *testing.T) {
	item := BatchItem{
		InputPath:  "/path/to/input.jpg",
		OutputPath: "/path/to/output.jpg",
		Options: CompressOptions{
			Quality: 80,
			Level:   CompressionMedium,
		},
	}

	if item.InputPath != "/path/to/input.jpg" {
		t.Errorf("BatchItem.InputPath = %v, want /path/to/input.jpg", item.InputPath)
	}
	if item.OutputPath != "/path/to/output.jpg" {
		t.Errorf("BatchItem.OutputPath = %v, want /path/to/output.jpg", item.OutputPath)
	}
	if item.Options.Quality != 80 {
		t.Errorf("BatchItem.Options.Quality = %v, want 80", item.Options.Quality)
	}
}
