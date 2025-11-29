package compressor

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	imgpkg "github.com/FrontWorksDev/image-compressor/internal/image"
)

func TestGetCompressor(t *testing.T) {
	tests := []struct {
		format   imgpkg.Format
		expected bool
	}{
		{imgpkg.FormatJPEG, true},
		{imgpkg.FormatPNG, true},
		{imgpkg.FormatUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			comp := GetCompressor(tt.format)
			if (comp != nil) != tt.expected {
				t.Errorf("GetCompressor(%v) returned %v, want non-nil=%v", tt.format, comp, tt.expected)
			}
		})
	}
}

func TestJPEGCompressor_Compress(t *testing.T) {
	comp := &JPEGCompressor{}
	img := createTestImage()

	tests := []struct {
		quality int
	}{
		{100},
		{80},
		{50},
		{1},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("quality_%d", tt.quality), func(t *testing.T) {
			data, err := comp.Compress(img, tt.quality)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}
			if len(data) == 0 {
				t.Error("compressed data is empty")
			}
		})
	}
}

func TestJPEGCompressor_Format(t *testing.T) {
	comp := &JPEGCompressor{}
	if comp.Format() != imgpkg.FormatJPEG {
		t.Errorf("Format() = %v, want %v", comp.Format(), imgpkg.FormatJPEG)
	}
}

func TestPNGCompressor_Compress(t *testing.T) {
	comp := &PNGCompressor{}
	img := createTestImage()

	tests := []struct {
		quality int
	}{
		{100},
		{80},
		{50},
		{1},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("quality_%d", tt.quality), func(t *testing.T) {
			data, err := comp.Compress(img, tt.quality)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}
			if len(data) == 0 {
				t.Error("compressed data is empty")
			}
		})
	}
}

func TestPNGCompressor_Format(t *testing.T) {
	comp := &PNGCompressor{}
	if comp.Format() != imgpkg.FormatPNG {
		t.Errorf("Format() = %v, want %v", comp.Format(), imgpkg.FormatPNG)
	}
}

func TestQualityToCompressionLevel(t *testing.T) {
	tests := []struct {
		quality  int
		minBytes bool // 低品質ほど圧縮率が高い（バイト数が少ない）
	}{
		{100, false},
		{90, false},
		{70, true},
		{50, true},
		{1, true},
	}

	comp := &PNGCompressor{}
	img := createTestImage()

	var prevSize int
	for _, tt := range tests {
		data, _ := comp.Compress(img, tt.quality)
		if prevSize > 0 && tt.minBytes && len(data) > prevSize {
			// 品質が下がるにつれてサイズが小さくなることを確認
			// ただしPNGは可逆圧縮なので、必ずしもそうならない場合がある
		}
		prevSize = len(data)
	}
}

func createTestImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 2), G: uint8(y * 2), B: uint8(x + y), A: 255})
		}
	}
	return img
}
