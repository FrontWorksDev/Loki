package image

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestProcessor_DetectFormatByPath(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		path     string
		expected Format
	}{
		{"image.jpg", FormatJPEG},
		{"image.jpeg", FormatJPEG},
		{"image.JPG", FormatJPEG},
		{"image.png", FormatPNG},
		{"image.PNG", FormatPNG},
		{"image.gif", FormatUnknown},
		{"image.webp", FormatUnknown},
		{"image", FormatUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := p.DetectFormatByPath(tt.path)
			if got != tt.expected {
				t.Errorf("DetectFormatByPath(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestProcessor_GenerateOutputPath(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		input    string
		suffix   string
		expected string
	}{
		{"/path/to/image.jpg", "_compressed", "/path/to/image_compressed.jpg"},
		{"/path/to/image.png", "_min", "/path/to/image_min.png"},
		{"image.jpg", "_compressed", "image_compressed.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := p.GenerateOutputPath(tt.input, tt.suffix)
			if got != tt.expected {
				t.Errorf("GenerateOutputPath(%q, %q) = %v, want %v", tt.input, tt.suffix, got, tt.expected)
			}
		})
	}
}

func TestProcessor_LoadAndSave(t *testing.T) {
	p := NewProcessor()

	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "imgcompress-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// テスト画像を作成
	img := createTestImage()

	// JPEGテスト
	t.Run("JPEG", func(t *testing.T) {
		jpegPath := filepath.Join(tmpDir, "test.jpg")
		err := p.Save(img, jpegPath, FormatJPEG, 80)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		loadedImg, format, err := p.Load(jpegPath)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if format != FormatJPEG {
			t.Errorf("format = %v, want %v", format, FormatJPEG)
		}
		if loadedImg == nil {
			t.Error("loaded image is nil")
		}
	})

	// PNGテスト
	t.Run("PNG", func(t *testing.T) {
		pngPath := filepath.Join(tmpDir, "test.png")
		err := p.Save(img, pngPath, FormatPNG, 80)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		loadedImg, format, err := p.Load(pngPath)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if format != FormatPNG {
			t.Errorf("format = %v, want %v", format, FormatPNG)
		}
		if loadedImg == nil {
			t.Error("loaded image is nil")
		}
	})
}

func TestProcessor_Load_FileNotFound(t *testing.T) {
	p := NewProcessor()

	_, _, err := p.Load("/nonexistent/file.jpg")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func createTestImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 5), G: uint8(y * 5), B: 100, A: 255})
		}
	}
	return img
}
