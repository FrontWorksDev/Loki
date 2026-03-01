package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// executeConvert resets globals, sets args, executes the root command and
// captures stdout/stderr output. Returns combined output and any error.
func executeConvert(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetGlobals(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	rootCmd.SetArgs(args)
	err := Execute()
	return buf.String(), err
}

// --- E2E: All 6 format conversion patterns ---

func TestE2E_Convert_JPEG_to_PNG(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	outputPath := filepath.Join(tmpDir, "output.png")

	jpegData := createTestJPEG(t, 100, 100, 95)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeConvert(t, "convert", inputPath, "-f", "png", "-o", outputPath)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	verifyImageFile(t, outputPath, "png")

	if !strings.Contains(out, "変換完了") {
		t.Errorf("出力に「変換完了」が含まれていません: %s", out)
	}
}

func TestE2E_Convert_PNG_to_JPEG(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.png")
	outputPath := filepath.Join(tmpDir, "output.jpg")

	pngData := createTestPNG(t, 100, 100)
	if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeConvert(t, "convert", inputPath, "-f", "jpeg", "-o", outputPath)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	verifyImageFile(t, outputPath, "jpeg")

	if !strings.Contains(out, "変換完了") {
		t.Errorf("出力に「変換完了」が含まれていません: %s", out)
	}
}

func TestE2E_Convert_JPEG_to_WebP(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	outputPath := filepath.Join(tmpDir, "output.webp")

	jpegData := createTestJPEG(t, 100, 100, 95)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := executeConvert(t, "convert", inputPath, "-f", "webp", "-o", outputPath)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("出力ファイルが作成されていません")
	}
}

func TestE2E_Convert_WebP_to_JPEG(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.webp")
	outputPath := filepath.Join(tmpDir, "output.jpg")

	webpData := createTestWEBP(t, 100, 100, 95)
	if err := os.WriteFile(inputPath, webpData, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeConvert(t, "convert", inputPath, "-f", "jpeg", "-o", outputPath)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	verifyImageFile(t, outputPath, "jpeg")

	if !strings.Contains(out, "変換完了") {
		t.Errorf("出力に「変換完了」が含まれていません: %s", out)
	}
}

func TestE2E_Convert_PNG_to_WebP(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.png")
	outputPath := filepath.Join(tmpDir, "output.webp")

	pngData := createTestPNG(t, 100, 100)
	if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := executeConvert(t, "convert", inputPath, "-f", "webp", "-o", outputPath)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("出力ファイルが作成されていません")
	}
}

func TestE2E_Convert_WebP_to_PNG(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.webp")
	outputPath := filepath.Join(tmpDir, "output.png")

	webpData := createTestWEBP(t, 100, 100, 95)
	if err := os.WriteFile(inputPath, webpData, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeConvert(t, "convert", inputPath, "-f", "png", "-o", outputPath)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	verifyImageFile(t, outputPath, "png")

	if !strings.Contains(out, "変換完了") {
		t.Errorf("出力に「変換完了」が含まれていません: %s", out)
	}
}

// --- E2E: Auto output path ---

func TestE2E_Convert_出力パス自動生成(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "auto.png")

	pngData := createTestPNG(t, 50, 50)
	if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := executeConvert(t, "convert", inputPath, "-f", "webp")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "auto.webp")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("自動生成された出力ファイルが存在しません: %s", expectedPath)
	}
}

// --- E2E: Same format error ---

func TestE2E_Convert_同一フォーマットエラー(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")

	jpegData := createTestJPEG(t, 50, 50, 90)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := executeConvert(t, "convert", inputPath, "-f", "jpeg")
	if err == nil {
		t.Fatal("同一フォーマットでエラーが返されるべき")
	}
	if !strings.Contains(err.Error(), "compressコマンドを使用してください") {
		t.Errorf("エラーメッセージに案内が含まれていません: %v", err)
	}
}

// --- E2E: Directory conversion ---

func TestE2E_Convert_ディレクトリ変換_全ファイル検証(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	setupTestDir(t, inputDir, map[string][]byte{
		"photo.jpg": createTestJPEG(t, 50, 50, 90),
		"icon.png":  createTestPNG(t, 50, 50),
	})

	out, err := executeConvert(t, "convert", inputDir, "-f", "webp", "-r", "-o", outputDir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Both files should be converted to webp.
	if _, err := os.Stat(filepath.Join(outputDir, "photo.webp")); os.IsNotExist(err) {
		t.Error("photo.webp が出力されていません")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "icon.webp")); os.IsNotExist(err) {
		t.Error("icon.webp が出力されていません")
	}

	if !strings.Contains(out, "成功 2") {
		t.Errorf("出力に「成功 2」が含まれていません: %s", out)
	}
}

func TestE2E_Convert_ディレクトリ変換_サブディレクトリ構造維持(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	setupTestDir(t, inputDir, map[string][]byte{
		"root.jpg":           createTestJPEG(t, 30, 30, 80),
		"sub1/a.png":         createTestPNG(t, 30, 30),
		"sub1/sub2/deep.jpg": createTestJPEG(t, 30, 30, 80),
	})

	_, err := executeConvert(t, "convert", inputDir, "-f", "webp", "-r", "-o", outputDir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, relPath := range []string{"root.webp", "sub1/a.webp", "sub1/sub2/deep.webp"} {
		p := filepath.Join(outputDir, relPath)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("出力ファイルが存在しません: %s", relPath)
		}
	}
}

func TestE2E_Convert_ディレクトリ変換_同一フォーマットスキップ(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	setupTestDir(t, inputDir, map[string][]byte{
		"photo.jpg":  createTestJPEG(t, 30, 30, 80),
		"icon.png":   createTestPNG(t, 30, 30),
		"image.webp": createTestWEBP(t, 30, 30, 80),
	})

	out, err := executeConvert(t, "convert", inputDir, "-f", "webp", "-r", "-o", outputDir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// image.webp should be skipped (already webp).
	if !strings.Contains(out, "成功 2") {
		t.Errorf("出力に「成功 2」が含まれていません (webpスキップ): %s", out)
	}

	// Verify the webp file was not converted.
	if _, err := os.Stat(filepath.Join(outputDir, "image.webp")); !os.IsNotExist(err) {
		t.Error("同一フォーマットのファイルがスキップされていません")
	}
}

func TestE2E_Convert_空ディレクトリ(t *testing.T) {
	inputDir := t.TempDir()

	out, err := executeConvert(t, "convert", inputDir, "-f", "webp", "-r")
	if err != nil {
		t.Fatalf("空ディレクトリでエラーが返されました: %v", err)
	}

	if !strings.Contains(out, "変換対象の画像ファイルが見つかりませんでした") {
		t.Errorf("出力に期待メッセージが含まれていません: %s", out)
	}
}

func TestE2E_Convert_ディレクトリ変換_出力パス自動生成(t *testing.T) {
	base := t.TempDir()
	inputDir := filepath.Join(base, "images")
	if err := os.Mkdir(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	setupTestDir(t, inputDir, map[string][]byte{
		"photo.jpg": createTestJPEG(t, 30, 30, 80),
	})

	_, err := executeConvert(t, "convert", inputDir, "-f", "png", "-r")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expectedDir := inputDir + "_converted"
	if _, err := os.Stat(filepath.Join(expectedDir, "photo.png")); os.IsNotExist(err) {
		t.Errorf("自動生成されたディレクトリに出力されていません: %s", expectedDir)
	}
}

// --- E2E: Directory conversion with partial failures ---

func TestE2E_Convert_ディレクトリ変換_一部失敗(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	setupTestDir(t, inputDir, map[string][]byte{
		"good.jpg":    createTestJPEG(t, 50, 50, 90),
		"corrupt.jpg": []byte("this is not a valid jpeg"),
	})

	out, err := executeConvert(t, "convert", inputDir, "-f", "png", "-r", "-o", outputDir)

	if err == nil {
		t.Fatal("一部失敗時にエラーが返されるべきです")
	}

	if !strings.Contains(out, "成功 1") {
		t.Errorf("出力に「成功 1」が含まれていません: %s", out)
	}
	if !strings.Contains(out, "失敗 1") {
		t.Errorf("出力に「失敗 1」が含まれていません: %s", out)
	}

	// The valid file should still be converted.
	if _, err := os.Stat(filepath.Join(outputDir, "good.png")); os.IsNotExist(err) {
		t.Error("good.png が出力されていません")
	}
}

// --- E2E: Flag combination tests ---

func TestE2E_Convert_フラグ組み合わせ(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "quality_50", args: []string{"--quality", "50"}},
		{name: "level_low", args: []string{"--level", "low"}},
		{name: "level_high", args: []string{"--level", "high"}},
		{name: "quality優先_quality70_level_high", args: []string{"--quality", "70", "--level", "high"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "test.png")
			outputPath := filepath.Join(tmpDir, "out.jpg")

			pngData := createTestPNG(t, 80, 80)
			if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
				t.Fatal(err)
			}

			args := append([]string{"convert", inputPath, "-f", "jpeg", "-o", outputPath}, tt.args...)
			_, err := executeConvert(t, args...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			verifyImageFile(t, outputPath, "jpeg")
		})
	}
}

// --- E2E: Error handling tests ---

func TestE2E_Convert_エラーハンドリング(t *testing.T) {
	tmpDir := t.TempDir()
	validJPEG := filepath.Join(tmpDir, "valid.jpg")
	if err := os.WriteFile(validJPEG, createTestJPEG(t, 10, 10, 80), 0o644); err != nil {
		t.Fatal(err)
	}
	bmpFile := filepath.Join(tmpDir, "test.bmp")
	if err := os.WriteFile(bmpFile, []byte("not a bmp"), 0o644); err != nil {
		t.Fatal(err)
	}
	emptyDir := filepath.Join(tmpDir, "emptydir")
	if err := os.Mkdir(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		args      []string
		wantInErr string
	}{
		{
			name: "存在しないファイル",
			args: []string{"convert", "/nonexistent/path/file.jpg", "-f", "png"},
		},
		{
			name: "存在しないディレクトリ",
			args: []string{"convert", "/nonexistent/dir/", "-f", "png", "-r"},
		},
		{
			name: "recursiveなしのディレクトリ",
			args: []string{"convert", emptyDir, "-f", "png"},
		},
		{
			name:      "不正なフォーマット",
			args:      []string{"convert", validJPEG, "-f", "bmp"},
			wantInErr: "不正なフォーマット",
		},
		{
			name:      "quality範囲外_101",
			args:      []string{"convert", validJPEG, "-f", "png", "--quality", "101"},
			wantInErr: "品質は1〜100の範囲",
		},
		{
			name:      "非対応フォーマットbmp",
			args:      []string{"convert", bmpFile, "-f", "png"},
			wantInErr: "サポートされていない画像形式",
		},
		{
			name:      "同一フォーマット",
			args:      []string{"convert", validJPEG, "-f", "jpeg"},
			wantInErr: "compressコマンドを使用してください",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeConvert(t, tt.args...)
			if err == nil {
				t.Fatal("エラーが返されるべきですが、nilでした")
			}
			if tt.wantInErr != "" && !strings.Contains(err.Error(), tt.wantInErr) {
				t.Errorf("エラーメッセージに %q が含まれていません: %v", tt.wantInErr, err)
			}
		})
	}
}
