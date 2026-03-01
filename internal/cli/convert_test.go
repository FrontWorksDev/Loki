package cli

import (
	"bytes"
	"fmt"
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

func TestConvertSingleFile_WebP_to_JPEG(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.webp")
	webpData := createTestWEBP(t, 100, 100, 95)
	if err := os.WriteFile(inputPath, webpData, 0o644); err != nil {
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

func TestConvertSingleFile_WebP_to_PNG(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.webp")
	webpData := createTestWEBP(t, 100, 100, 95)
	if err := os.WriteFile(inputPath, webpData, 0o644); err != nil {
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

func TestConvertSingleFile_破損画像(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "corrupt.jpg")
	if err := os.WriteFile(inputPath, []byte("not a valid jpeg"), 0o644); err != nil {
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
	if err == nil {
		t.Fatal("破損画像でエラーが返されるべき")
	}
	if !strings.Contains(err.Error(), "変換に失敗しました") {
		t.Errorf("エラーメッセージに「変換に失敗しました」が含まれていません: %v", err)
	}

	// Output file should be cleaned up.
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Error("失敗時に出力ファイルが削除されていません")
	}
}

func TestConvertSingleFile_出力パス自動生成(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.png")
	pngData := createTestPNG(t, 50, 50)
	if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	// convert.output を空にして自動生成を確認
	viper.Set("convert.output", "")

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatJPEG,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	err := convertSingleFile(cmd, inputPath, processor.FormatJPEG, opts)
	if err != nil {
		t.Fatalf("convertSingleFile() error = %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "test.jpg")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("自動生成された出力ファイルが存在しません: %s", expectedPath)
	}
}

func TestConvertSingleFile_PNG同一フォーマット(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.png")
	pngData := createTestPNG(t, 50, 50)
	if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatPNG,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	err := convertSingleFile(cmd, inputPath, processor.FormatPNG, opts)
	if err == nil {
		t.Fatal("同一フォーマットでエラーが返されるべき")
	}
}

func TestConvertSingleFile_WebP同一フォーマット(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.webp")
	webpData := createTestWEBP(t, 50, 50, 80)
	if err := os.WriteFile(inputPath, webpData, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatWEBP,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	err := convertSingleFile(cmd, inputPath, processor.FormatWEBP, opts)
	if err == nil {
		t.Fatal("同一フォーマットでエラーが返されるべき")
	}
}

func TestRunConvert_不正レベル(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	jpegData := createTestJPEG(t, 10, 10, 80)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"convert", inputPath, "-f", "png", "--level", "wrong"})

	err := Execute()
	if err == nil {
		t.Error("不正なレベルでエラーが返されるべき")
	}
}

func TestRunConvert_ディレクトリにrecursiveなし(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	rootCmd.SetArgs([]string{"convert", tmpDir, "-f", "webp"})

	err := Execute()
	if err == nil {
		t.Error("ディレクトリに--recursiveなしでエラーが返されるべき")
	}
}

func TestRunConvert_単一ファイル成功(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.png")
	pngData := createTestPNG(t, 100, 100)
	if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "output.jpg")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	rootCmd.SetArgs([]string{"convert", inputPath, "-f", "jpeg", "-o", outputPath})

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("出力ファイルが作成されていません")
	}

	out := buf.String()
	if !strings.Contains(out, "変換完了") {
		t.Errorf("出力に「変換完了」が含まれていません: %s", out)
	}
}

func TestRunConvert_ディレクトリ成功(t *testing.T) {
	resetGlobals(t)

	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	jpegData := createTestJPEG(t, 50, 50, 90)
	pngData := createTestPNG(t, 50, 50)

	if err := os.WriteFile(filepath.Join(inputDir, "photo.jpg"), jpegData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "icon.png"), pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	rootCmd.SetArgs([]string{"convert", inputDir, "-f", "webp", "-r", "-o", outputDir})

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "完了") {
		t.Errorf("出力に「完了」が含まれていません: %s", out)
	}
}

func TestConvertResultsToBatchResults(t *testing.T) {
	results := []processor.BatchConvertResult{
		{
			Item: processor.BatchConvertItem{
				InputPath:  "/input/a.jpg",
				OutputPath: "/output/a.png",
			},
			Result: &processor.Result{
				OriginalSize:   1000,
				CompressedSize: 800,
				Format:         processor.FormatPNG,
			},
		},
		{
			Item: processor.BatchConvertItem{
				InputPath:  "/input/b.png",
				OutputPath: "/output/b.webp",
			},
			Error: fmt.Errorf("test error"),
		},
	}

	batchResults := convertResultsToBatchResults(results)

	if len(batchResults) != 2 {
		t.Fatalf("convertResultsToBatchResults() returned %d results, want 2", len(batchResults))
	}

	if batchResults[0].Item.InputPath != "/input/a.jpg" {
		t.Errorf("InputPath = %q, want %q", batchResults[0].Item.InputPath, "/input/a.jpg")
	}
	if batchResults[0].Item.OutputPath != "/output/a.png" {
		t.Errorf("OutputPath = %q, want %q", batchResults[0].Item.OutputPath, "/output/a.png")
	}
	if batchResults[0].Result == nil {
		t.Error("Result should not be nil")
	}
	if batchResults[0].Error != nil {
		t.Errorf("Error should be nil, got %v", batchResults[0].Error)
	}

	if batchResults[1].Error == nil {
		t.Error("Error should not be nil")
	}
	if batchResults[1].Result != nil {
		t.Error("Result should be nil for failed item")
	}
}

func TestConvertResultsToBatchResults_空(t *testing.T) {
	batchResults := convertResultsToBatchResults([]processor.BatchConvertResult{})
	if len(batchResults) != 0 {
		t.Errorf("convertResultsToBatchResults() returned %d results, want 0", len(batchResults))
	}
}

func TestConvertDirectory_recursiveなしエラー(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	viper.Set("convert.recursive", false)

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatWEBP,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	err := convertDirectory(cmd, tmpDir, processor.FormatWEBP, opts)
	if err == nil {
		t.Fatal("recursiveなしでエラーが返されるべき")
	}
	if !strings.Contains(err.Error(), "--recursive") {
		t.Errorf("エラーメッセージに--recursiveが含まれていません: %v", err)
	}
}

func TestConvertDirectory_空ディレクトリ(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	viper.Set("convert.recursive", true)
	viper.Set("convert.output", filepath.Join(t.TempDir(), "output"))

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatWEBP,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	err := convertDirectory(cmd, tmpDir, processor.FormatWEBP, opts)
	if err != nil {
		t.Fatalf("空ディレクトリでエラーが返されるべきではない: %v", err)
	}
}

func TestConvertDirectory_出力パス自動生成(t *testing.T) {
	resetGlobals(t)

	base := t.TempDir()
	inputDir := filepath.Join(base, "images")
	if err := os.Mkdir(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	jpegData := createTestJPEG(t, 30, 30, 80)
	if err := os.WriteFile(filepath.Join(inputDir, "test.jpg"), jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	viper.Set("convert.recursive", true)
	viper.Set("convert.output", "") // 自動生成

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatPNG,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	err := convertDirectory(cmd, inputDir, processor.FormatPNG, opts)
	if err != nil {
		t.Fatalf("convertDirectory() error = %v", err)
	}

	expectedDir := inputDir + "_converted"
	if _, err := os.Stat(filepath.Join(expectedDir, "test.png")); os.IsNotExist(err) {
		t.Errorf("自動生成ディレクトリに出力されていません: %s", expectedDir)
	}
}

func TestConvertDirectory_同一フォーマットのみ(t *testing.T) {
	resetGlobals(t)

	inputDir := t.TempDir()
	webpData := createTestWEBP(t, 30, 30, 80)
	if err := os.WriteFile(filepath.Join(inputDir, "image.webp"), webpData, 0o644); err != nil {
		t.Fatal(err)
	}

	viper.Set("convert.recursive", true)
	viper.Set("convert.output", filepath.Join(t.TempDir(), "output"))

	cmd := newTestCmd()
	opts := processor.ConvertOptions{
		Format:          processor.FormatWEBP,
		CompressOptions: processor.DefaultCompressOptions(),
	}
	// All files are webp and target is webp, so all should be skipped.
	err := convertDirectory(cmd, inputDir, processor.FormatWEBP, opts)
	if err != nil {
		t.Fatalf("convertDirectory() error = %v", err)
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
