package cli

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/FrontWorksDev/Loki/pkg/processor"
	"github.com/spf13/cobra"
)

// createTestJPEG creates a test JPEG image with the specified dimensions.
func createTestJPEG(t *testing.T, width, height int, quality int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / max(width, 1)),
				G: uint8(y * 255 / max(height, 1)),
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
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / max(width, 1)),
				G: uint8(y * 255 / max(height, 1)),
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

// newTestCmd creates a cobra.Command suitable for testing (output discarded).
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetContext(context.Background())
	return cmd
}

// resetGlobals resets all global flag variables to their defaults.
func resetGlobals(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		quality = 0
		level = "medium"
		output = ""
		recursive = false
		rootCmd.SetArgs([]string{})
	})
}

func TestParseCompressionLevel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    processor.CompressionLevel
		wantErr bool
	}{
		{name: "low", input: "low", want: processor.CompressionLow},
		{name: "medium", input: "medium", want: processor.CompressionMedium},
		{name: "high", input: "high", want: processor.CompressionHigh},
		{name: "大文字LOW", input: "LOW", want: processor.CompressionLow},
		{name: "大文字MEDIUM", input: "MEDIUM", want: processor.CompressionMedium},
		{name: "大文字HIGH", input: "HIGH", want: processor.CompressionHigh},
		{name: "混合大文字小文字", input: "Medium", want: processor.CompressionMedium},
		{name: "不正な値", input: "wrong", wantErr: true},
		{name: "空文字", input: "", wantErr: true},
		{name: "数値", input: "123", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCompressionLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCompressionLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseCompressionLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    processor.ImageFormat
		wantErr bool
	}{
		{name: ".jpg拡張子", path: "photo.jpg", want: processor.FormatJPEG},
		{name: ".jpeg拡張子", path: "photo.jpeg", want: processor.FormatJPEG},
		{name: ".png拡張子", path: "icon.png", want: processor.FormatPNG},
		{name: ".JPG大文字", path: "PHOTO.JPG", want: processor.FormatJPEG},
		{name: ".JPEG大文字", path: "PHOTO.JPEG", want: processor.FormatJPEG},
		{name: ".PNG大文字", path: "ICON.PNG", want: processor.FormatPNG},
		{name: "パス付き", path: "/path/to/photo.jpg", want: processor.FormatJPEG},
		{name: "非対応_bmp", path: "file.bmp", wantErr: true},
		{name: "非対応_gif", path: "file.gif", wantErr: true},
		{name: "拡張子なし", path: "noext", wantErr: true},
		{name: "空文字", path: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detectFormat(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("detectFormat(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("detectFormat(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDefaultOutputPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "JPEGファイル", input: "photo.jpg", want: "photo_compressed.jpg"},
		{name: "PNGファイル", input: "icon.png", want: "icon_compressed.png"},
		{name: "パス付き", input: "/path/to/photo.jpg", want: "/path/to/photo_compressed.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultOutputPath(tt.input)
			if got != tt.want {
				t.Errorf("defaultOutputPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCompressSingleFile_JPEG(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	jpegData := createTestJPEG(t, 100, 100, 95)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "output.jpg")
	output = outputPath

	cmd := newTestCmd()
	err := compressSingleFile(cmd, inputPath, processor.DefaultCompressOptions())
	if err != nil {
		t.Fatalf("compressSingleFile() error = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("出力ファイルが作成されていません")
	}
}

func TestCompressSingleFile_PNG(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.png")
	pngData := createTestPNG(t, 100, 100)
	if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "output.png")
	output = outputPath

	cmd := newTestCmd()
	err := compressSingleFile(cmd, inputPath, processor.DefaultCompressOptions())
	if err != nil {
		t.Fatalf("compressSingleFile() error = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("出力ファイルが作成されていません")
	}
}

func TestCompressDirectory(t *testing.T) {
	resetGlobals(t)

	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 50, 50, 90)
	pngData := createTestPNG(t, 50, 50)

	if err := os.WriteFile(filepath.Join(inputDir, "photo.jpg"), jpegData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "icon.png"), pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	output = outputDir
	recursive = true

	cmd := newTestCmd()
	err := compressDirectory(cmd, inputDir, processor.DefaultCompressOptions())
	if err != nil {
		t.Fatalf("compressDirectory() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputDir, "photo.jpg")); os.IsNotExist(err) {
		t.Error("photo.jpg が出力されていません")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "icon.png")); os.IsNotExist(err) {
		t.Error("icon.png が出力されていません")
	}
}

func TestRunCompress_引数なし(t *testing.T) {
	resetGlobals(t)

	rootCmd.SetArgs([]string{"compress"})

	err := Execute()
	if err == nil {
		t.Error("引数なしでエラーが返されるべき")
	}
}

func TestRunCompress_存在しないファイル(t *testing.T) {
	resetGlobals(t)

	rootCmd.SetArgs([]string{"compress", "/nonexistent/file.jpg"})

	err := Execute()
	if err == nil {
		t.Error("存在しないファイルでエラーが返されるべき")
	}
}

func TestRunCompress_ディレクトリにrecursiveなし(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	rootCmd.SetArgs([]string{"compress", tmpDir})

	err := Execute()
	if err == nil {
		t.Error("ディレクトリに--recursiveなしでエラーが返されるべき")
	}
}

func TestRunCompress_不正レベル(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	jpegData := createTestJPEG(t, 10, 10, 80)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"compress", inputPath, "--level", "wrong"})

	err := Execute()
	if err == nil {
		t.Error("不正なレベルでエラーが返されるべき")
	}
}

func TestRunCompress_品質範囲外(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	jpegData := createTestJPEG(t, 10, 10, 80)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"compress", inputPath, "--quality", "150"})

	err := Execute()
	if err == nil {
		t.Error("品質範囲外でエラーが返されるべき")
	}
}
