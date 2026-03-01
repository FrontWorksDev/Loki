package processor

import (
	"image/png"
	"testing"
)

func TestImageFormat_String(t *testing.T) {
	tests := []struct {
		name     string
		format   ImageFormat
		expected string
	}{
		{"JPEG format", FormatJPEG, "jpeg"},
		{"PNG format", FormatPNG, "png"},
		{"WEBP format", FormatWEBP, "webp"},
		{"Unknown format", ImageFormat(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format.String(); got != tt.expected {
				t.Errorf("ImageFormat.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestImageFormat_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		format   ImageFormat
		expected bool
	}{
		{"JPEG is valid", FormatJPEG, true},
		{"PNG is valid", FormatPNG, true},
		{"WEBP is valid", FormatWEBP, true},
		{"Unknown is invalid", ImageFormat(99), false},
		{"Negative is invalid", ImageFormat(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format.IsValid(); got != tt.expected {
				t.Errorf("ImageFormat.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestImageFormat_Extension(t *testing.T) {
	tests := []struct {
		name     string
		format   ImageFormat
		expected string
	}{
		{"JPEG extension", FormatJPEG, ".jpg"},
		{"PNG extension", FormatPNG, ".png"},
		{"WEBP extension", FormatWEBP, ".webp"},
		{"Unknown extension", ImageFormat(99), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format.Extension(); got != tt.expected {
				t.Errorf("ImageFormat.Extension() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestImageFormat_MIMEType(t *testing.T) {
	tests := []struct {
		name     string
		format   ImageFormat
		expected string
	}{
		{"JPEG MIME type", FormatJPEG, "image/jpeg"},
		{"PNG MIME type", FormatPNG, "image/png"},
		{"WEBP MIME type", FormatWEBP, "image/webp"},
		{"Unknown MIME type", ImageFormat(99), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format.MIMEType(); got != tt.expected {
				t.Errorf("ImageFormat.MIMEType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompressionLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    CompressionLevel
		expected string
	}{
		{"Low compression", CompressionLow, "low"},
		{"Medium compression", CompressionMedium, "medium"},
		{"High compression", CompressionHigh, "high"},
		{"Unknown compression", CompressionLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("CompressionLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompressionLevel_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		level    CompressionLevel
		expected bool
	}{
		{"Low is valid", CompressionLow, true},
		{"Medium is valid", CompressionMedium, true},
		{"High is valid", CompressionHigh, true},
		{"Unknown is invalid", CompressionLevel(99), false},
		{"Negative is invalid", CompressionLevel(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.IsValid(); got != tt.expected {
				t.Errorf("CompressionLevel.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompressionLevel_ToPNGCompressionLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    CompressionLevel
		expected png.CompressionLevel
	}{
		{"Low to BestSpeed", CompressionLow, png.BestSpeed},
		{"Medium to DefaultCompression", CompressionMedium, png.DefaultCompression},
		{"High to BestCompression", CompressionHigh, png.BestCompression},
		{"Unknown to DefaultCompression", CompressionLevel(99), png.DefaultCompression},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.ToPNGCompressionLevel(); got != tt.expected {
				t.Errorf("CompressionLevel.ToPNGCompressionLevel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompressionLevel_ToWebPQuality(t *testing.T) {
	tests := []struct {
		name     string
		level    CompressionLevel
		expected float32
	}{
		{"Low quality", CompressionLow, 60},
		{"Medium quality", CompressionMedium, 75},
		{"High quality", CompressionHigh, 90},
		{"Unknown defaults to medium", CompressionLevel(99), 75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.ToWebPQuality(); got != tt.expected {
				t.Errorf("CompressionLevel.ToWebPQuality() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompressionLevel_ToJPEGQuality(t *testing.T) {
	tests := []struct {
		name     string
		level    CompressionLevel
		expected int
	}{
		{"Low quality", CompressionLow, 60},
		{"Medium quality", CompressionMedium, 75},
		{"High quality", CompressionHigh, 90},
		{"Unknown defaults to medium", CompressionLevel(99), 75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.ToJPEGQuality(); got != tt.expected {
				t.Errorf("CompressionLevel.ToJPEGQuality() = %v, want %v", got, tt.expected)
			}
		})
	}
}
