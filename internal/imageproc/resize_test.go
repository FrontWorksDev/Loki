package imageproc

import (
	"image"
	"image/color"
	"testing"
)

// createTestImage はテスト用の画像を生成します
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// 単色で塗りつぶす
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	return img
}

func TestResizeImage(t *testing.T) {
	tests := []struct {
		name          string
		sourceWidth   int
		sourceHeight  int
		targetWidth   int
		targetHeight  int
		expectedWidth int
		expectedHeight int
	}{
		{
			name:          "100x100を50x50にリサイズ",
			sourceWidth:   100,
			sourceHeight:  100,
			targetWidth:   50,
			targetHeight:  50,
			expectedWidth: 50,
			expectedHeight: 50,
		},
		{
			name:          "200x100を100x50にリサイズ",
			sourceWidth:   200,
			sourceHeight:  100,
			targetWidth:   100,
			targetHeight:  50,
			expectedWidth: 100,
			expectedHeight: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := createTestImage(tt.sourceWidth, tt.sourceHeight)
			resized := ResizeImage(img, tt.targetWidth, tt.targetHeight)

			bounds := resized.Bounds()
			if bounds.Dx() != tt.expectedWidth {
				t.Errorf("幅が期待値と異なります: got %d, want %d", bounds.Dx(), tt.expectedWidth)
			}
			if bounds.Dy() != tt.expectedHeight {
				t.Errorf("高さが期待値と異なります: got %d, want %d", bounds.Dy(), tt.expectedHeight)
			}
		})
	}
}

func TestResizeImageByWidth(t *testing.T) {
	tests := []struct {
		name          string
		sourceWidth   int
		sourceHeight  int
		targetWidth   int
		expectedWidth int
		expectedHeight int
	}{
		{
			name:          "100x100を幅50にリサイズ（アスペクト比維持）",
			sourceWidth:   100,
			sourceHeight:  100,
			targetWidth:   50,
			expectedWidth: 50,
			expectedHeight: 50,
		},
		{
			name:          "200x100を幅100にリサイズ（アスペクト比維持）",
			sourceWidth:   200,
			sourceHeight:  100,
			targetWidth:   100,
			expectedWidth: 100,
			expectedHeight: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := createTestImage(tt.sourceWidth, tt.sourceHeight)
			resized := ResizeImageByWidth(img, tt.targetWidth)

			bounds := resized.Bounds()
			if bounds.Dx() != tt.expectedWidth {
				t.Errorf("幅が期待値と異なります: got %d, want %d", bounds.Dx(), tt.expectedWidth)
			}
			if bounds.Dy() != tt.expectedHeight {
				t.Errorf("高さが期待値と異なります: got %d, want %d", bounds.Dy(), tt.expectedHeight)
			}
		})
	}
}

func TestResizeImageByHeight(t *testing.T) {
	tests := []struct {
		name           string
		sourceWidth    int
		sourceHeight   int
		targetHeight   int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "100x100を高さ50にリサイズ（アスペクト比維持）",
			sourceWidth:    100,
			sourceHeight:   100,
			targetHeight:   50,
			expectedWidth:  50,
			expectedHeight: 50,
		},
		{
			name:           "100x200を高さ100にリサイズ（アスペクト比維持）",
			sourceWidth:    100,
			sourceHeight:   200,
			targetHeight:   100,
			expectedWidth:  50,
			expectedHeight: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := createTestImage(tt.sourceWidth, tt.sourceHeight)
			resized := ResizeImageByHeight(img, tt.targetHeight)

			bounds := resized.Bounds()
			if bounds.Dx() != tt.expectedWidth {
				t.Errorf("幅が期待値と異なります: got %d, want %d", bounds.Dx(), tt.expectedWidth)
			}
			if bounds.Dy() != tt.expectedHeight {
				t.Errorf("高さが期待値と異なります: got %d, want %d", bounds.Dy(), tt.expectedHeight)
			}
		})
	}
}
