package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FrontWorksDev/Loki/pkg/processor"
	"github.com/spf13/viper"
)

func TestParseImageFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    processor.ImageFormat
		wantErr bool
	}{
		{name: "jpeg", input: "jpeg", want: processor.FormatJPEG},
		{name: "jpg", input: "jpg", want: processor.FormatJPEG},
		{name: "png", input: "png", want: processor.FormatPNG},
		{name: "webp", input: "webp", want: processor.FormatWEBP},
		{name: "大文字JPEG", input: "JPEG", want: processor.FormatJPEG},
		{name: "大文字JPG", input: "JPG", want: processor.FormatJPEG},
		{name: "大文字PNG", input: "PNG", want: processor.FormatPNG},
		{name: "大文字WEBP", input: "WEBP", want: processor.FormatWEBP},
		{name: "混合大文字Jpeg", input: "Jpeg", want: processor.FormatJPEG},
		{name: "不正な値", input: "bmp", wantErr: true},
		{name: "空文字", input: "", wantErr: true},
		{name: "数値", input: "123", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseImageFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseImageFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseImageFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultConvertOutputPath(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		format processor.ImageFormat
		want   string
	}{
		{name: "PNG→JPEG", input: "photo.png", format: processor.FormatJPEG, want: "photo.jpg"},
		{name: "JPEG→PNG", input: "photo.jpg", format: processor.FormatPNG, want: "photo.png"},
		{name: "JPEG→WebP", input: "photo.jpg", format: processor.FormatWEBP, want: "photo.webp"},
		{name: "PNG→WebP", input: "icon.png", format: processor.FormatWEBP, want: "icon.webp"},
		{name: "WebP→JPEG", input: "image.webp", format: processor.FormatJPEG, want: "image.jpg"},
		{name: "WebP→PNG", input: "image.webp", format: processor.FormatPNG, want: "image.png"},
		{name: "パス付き", input: "/path/to/photo.png", format: processor.FormatJPEG, want: "/path/to/photo.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultConvertOutputPath(tt.input, tt.format)
			if got != tt.want {
				t.Errorf("defaultConvertOutputPath(%q, %v) = %q, want %q", tt.input, tt.format, got, tt.want)
			}
		})
	}
}

func TestConvertSingleFile_PNG_to_JPEG(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.png")
	pngData := createTestPNG(t, 100, 100)
	if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "output.jpg")
	viper.Set("convert.output", outputPath)

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatJPEG,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	err := convertSingleFile(cmd, inputPath, processor.FormatJPEG, opts)
	if err != nil {
		t.Fatalf("convertSingleFile() error = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("出力ファイルが作成されていません")
	}
}

func TestConvertSingleFile_JPEG_to_PNG(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	jpegData := createTestJPEG(t, 100, 100, 95)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "output.png")
	viper.Set("convert.output", outputPath)

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatPNG,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	err := convertSingleFile(cmd, inputPath, processor.FormatPNG, opts)
	if err != nil {
		t.Fatalf("convertSingleFile() error = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("出力ファイルが作成されていません")
	}
}

func TestConvertSingleFile_PNG_to_WebP(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.png")
	pngData := createTestPNG(t, 100, 100)
	if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "output.webp")
	viper.Set("convert.output", outputPath)

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatWEBP,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	err := convertSingleFile(cmd, inputPath, processor.FormatWEBP, opts)
	if err != nil {
		t.Fatalf("convertSingleFile() error = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("出力ファイルが作成されていません")
	}
}

func TestConvertSingleFile_JPEG_to_WebP(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	jpegData := createTestJPEG(t, 100, 100, 95)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "output.webp")
	viper.Set("convert.output", outputPath)

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatWEBP,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	err := convertSingleFile(cmd, inputPath, processor.FormatWEBP, opts)
	if err != nil {
		t.Fatalf("convertSingleFile() error = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("出力ファイルが作成されていません")
	}
}

func TestConvertSingleFile_同一フォーマットエラー(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	jpegData := createTestJPEG(t, 100, 100, 95)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatJPEG,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	err := convertSingleFile(cmd, inputPath, processor.FormatJPEG, opts)
	if err == nil {
		t.Fatal("同一フォーマットでエラーが返されるべき")
	}
	if !strings.Contains(err.Error(), "compressコマンドを使用してください") {
		t.Errorf("エラーメッセージに案内が含まれていません: %v", err)
	}
}

func TestRunConvert_存在しないファイル(t *testing.T) {
	resetGlobals(t)

	rootCmd.SetArgs([]string{"convert", "/nonexistent/file.jpg", "-f", "png"})

	err := Execute()
	if err == nil {
		t.Error("存在しないファイルでエラーが返されるべき")
	}
}

func TestRunConvert_不正フォーマット(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	jpegData := createTestJPEG(t, 10, 10, 80)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"convert", inputPath, "-f", "bmp"})

	err := Execute()
	if err == nil {
		t.Error("不正フォーマットでエラーが返されるべき")
	}
}

func TestRunConvert_品質範囲外(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	jpegData := createTestJPEG(t, 10, 10, 80)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"convert", inputPath, "-f", "png", "--quality", "150"})

	err := Execute()
	if err == nil {
		t.Error("品質範囲外でエラーが返されるべき")
	}
}
