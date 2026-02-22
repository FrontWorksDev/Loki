package cli

import (
	"bytes"
	"image"
	// Register decoders for verification.
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Helper Functions ---

// executeCompress resets globals, sets args, executes the root command and
// captures stdout/stderr output. Returns combined output and any error.
func executeCompress(t *testing.T, args ...string) (string, error) {
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

// verifyImageFile opens the file at path, decodes it as an image, and asserts
// that the decoded format matches expectedFormat ("jpeg" or "png").
func verifyImageFile(t *testing.T, path, expectedFormat string) {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("出力ファイルを開けません: %v", err)
	}
	defer f.Close()

	_, format, err := image.Decode(f)
	if err != nil {
		t.Fatalf("画像のデコードに失敗しました (%s): %v", path, err)
	}
	if format != expectedFormat {
		t.Errorf("画像フォーマット = %q, want %q", format, expectedFormat)
	}
}

// setupTestDir creates the given directory tree with files. The files map is
// keyed by relative path (e.g. "sub/image.jpg") and valued by file contents.
func setupTestDir(t *testing.T, dir string, files map[string][]byte) {
	t.Helper()
	for relPath, data := range files {
		writeTestFile(t, dir, relPath, data)
	}
}

// writeTestFile writes data to dir/name, creating parent directories as needed.
func writeTestFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("ディレクトリ作成失敗: %v", err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("ファイル書き込み失敗: %v", err)
	}
}

// --- Step 3: E2E Basic Tests (Single File) ---

func TestE2E_JPEG圧縮_出力ファイル検証(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.jpg")
	outputPath := filepath.Join(tmpDir, "output.jpg")

	jpegData := createTestJPEG(t, 100, 100, 95)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeCompress(t, "compress", inputPath, "-o", outputPath)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	verifyImageFile(t, outputPath, "jpeg")

	if !strings.Contains(out, "圧縮完了") {
		t.Errorf("出力に「圧縮完了」が含まれていません: %s", out)
	}
	if !strings.Contains(out, "削減率") {
		t.Errorf("出力に「削減率」が含まれていません: %s", out)
	}
}

func TestE2E_PNG圧縮_出力ファイル検証(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.png")
	outputPath := filepath.Join(tmpDir, "output.png")

	pngData := createTestPNG(t, 100, 100)
	if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeCompress(t, "compress", inputPath, "-o", outputPath)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	verifyImageFile(t, outputPath, "png")

	if !strings.Contains(out, "圧縮完了") {
		t.Errorf("出力に「圧縮完了」が含まれていません: %s", out)
	}
	if !strings.Contains(out, "削減率") {
		t.Errorf("出力に「削減率」が含まれていません: %s", out)
	}
}

func TestE2E_大きな画像の圧縮(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "large.jpg")
	outputPath := filepath.Join(tmpDir, "large_out.jpg")

	jpegData := createTestJPEG(t, 500, 500, 95)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := executeCompress(t, "compress", inputPath, "-o", outputPath, "--quality", "50")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	verifyImageFile(t, outputPath, "jpeg")

	origInfo, _ := os.Stat(inputPath)
	compInfo, _ := os.Stat(outputPath)
	if compInfo.Size() >= origInfo.Size() {
		t.Errorf("圧縮後サイズ(%d) >= 元サイズ(%d)", compInfo.Size(), origInfo.Size())
	}
}

func TestE2E_出力パス自動生成(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "auto.jpg")

	jpegData := createTestJPEG(t, 50, 50, 90)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := executeCompress(t, "compress", inputPath)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "auto_compressed.jpg")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("自動生成された出力ファイルが存在しません: %s", expectedPath)
	}
	verifyImageFile(t, expectedPath, "jpeg")
}

// --- Step 4: E2E Directory Tests ---

func TestE2E_ディレクトリ圧縮_全ファイル検証(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	setupTestDir(t, inputDir, map[string][]byte{
		"photo.jpg": createTestJPEG(t, 50, 50, 90),
		"icon.png":  createTestPNG(t, 50, 50),
	})

	out, err := executeCompress(t, "compress", inputDir, "-r", "-o", outputDir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	verifyImageFile(t, filepath.Join(outputDir, "photo.jpg"), "jpeg")
	verifyImageFile(t, filepath.Join(outputDir, "icon.png"), "png")

	if !strings.Contains(out, "成功 2") {
		t.Errorf("出力に「成功 2」が含まれていません: %s", out)
	}
}

func TestE2E_ディレクトリ圧縮_サブディレクトリ構造維持(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	setupTestDir(t, inputDir, map[string][]byte{
		"root.jpg":            createTestJPEG(t, 30, 30, 80),
		"sub1/a.png":          createTestPNG(t, 30, 30),
		"sub1/sub2/deep.jpg":  createTestJPEG(t, 30, 30, 80),
	})

	_, err := executeCompress(t, "compress", inputDir, "-r", "-o", outputDir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, relPath := range []string{"root.jpg", "sub1/a.png", "sub1/sub2/deep.jpg"} {
		p := filepath.Join(outputDir, relPath)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("出力ファイルが存在しません: %s", relPath)
		}
	}

	verifyImageFile(t, filepath.Join(outputDir, "root.jpg"), "jpeg")
	verifyImageFile(t, filepath.Join(outputDir, "sub1/a.png"), "png")
	verifyImageFile(t, filepath.Join(outputDir, "sub1/sub2/deep.jpg"), "jpeg")
}

func TestE2E_ディレクトリ圧縮_画像以外スキップ(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	setupTestDir(t, inputDir, map[string][]byte{
		"photo.jpg":  createTestJPEG(t, 30, 30, 80),
		"readme.txt": []byte("this is not an image"),
		"data.csv":   []byte("a,b,c"),
	})

	_, err := executeCompress(t, "compress", inputDir, "-r", "-o", outputDir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	verifyImageFile(t, filepath.Join(outputDir, "photo.jpg"), "jpeg")

	for _, skip := range []string{"readme.txt", "data.csv"} {
		p := filepath.Join(outputDir, skip)
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("画像以外のファイルが出力されています: %s", skip)
		}
	}
}

func TestE2E_ディレクトリ圧縮_出力パス自動生成(t *testing.T) {
	// Create a named directory inside TempDir so the _compressed path is predictable.
	base := t.TempDir()
	inputDir := filepath.Join(base, "images")
	if err := os.Mkdir(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	setupTestDir(t, inputDir, map[string][]byte{
		"photo.jpg": createTestJPEG(t, 30, 30, 80),
	})

	_, err := executeCompress(t, "compress", inputDir, "-r")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expectedDir := inputDir + "_compressed"
	if _, err := os.Stat(filepath.Join(expectedDir, "photo.jpg")); os.IsNotExist(err) {
		t.Errorf("自動生成されたディレクトリに出力されていません: %s", expectedDir)
	}
}

func TestE2E_空ディレクトリ圧縮(t *testing.T) {
	inputDir := t.TempDir()

	out, err := executeCompress(t, "compress", inputDir, "-r")
	if err != nil {
		t.Fatalf("空ディレクトリでエラーが返されました: %v", err)
	}

	if !strings.Contains(out, "対象の画像ファイルが見つかりませんでした") {
		t.Errorf("出力に期待メッセージが含まれていません: %s", out)
	}
}

// --- Step 5: Flag Combination Tests (Table-Driven) ---

func TestE2E_フラグ組み合わせ(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "quality_50", args: []string{"--quality", "50"}},
		{name: "level_low", args: []string{"--level", "low"}},
		{name: "level_high", args: []string{"--level", "high"}},
		{name: "quality優先_quality70_level_high", args: []string{"--quality", "70", "--level", "high"}},
		{name: "quality境界値_1", args: []string{"--quality", "1"}},
		{name: "quality境界値_100", args: []string{"--quality", "100"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "test.jpg")
			outputPath := filepath.Join(tmpDir, "out.jpg")

			jpegData := createTestJPEG(t, 80, 80, 95)
			if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
				t.Fatal(err)
			}

			args := append([]string{"compress", inputPath, "-o", outputPath}, tt.args...)
			_, err := executeCompress(t, args...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			verifyImageFile(t, outputPath, "jpeg")
		})
	}
}

func TestE2E_フラグ組み合わせ_ディレクトリ(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "recursive_quality_60", args: []string{"-r", "--quality", "60"}},
		{name: "recursive_level_low", args: []string{"-r", "--level", "low"}},
		{name: "recursive_level_high", args: []string{"-r", "--level", "high"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputDir := t.TempDir()
			outputDir := filepath.Join(t.TempDir(), "output")

			setupTestDir(t, inputDir, map[string][]byte{
				"a.jpg": createTestJPEG(t, 50, 50, 90),
				"b.png": createTestPNG(t, 50, 50),
			})

			args := append([]string{"compress", inputDir, "-o", outputDir}, tt.args...)
			_, err := executeCompress(t, args...)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			verifyImageFile(t, filepath.Join(outputDir, "a.jpg"), "jpeg")
			verifyImageFile(t, filepath.Join(outputDir, "b.png"), "png")
		})
	}
}

// --- Step 6: Error Handling Tests (Table-Driven) ---

func TestE2E_エラーハンドリング(t *testing.T) {
	// Prepare a valid JPEG for tests that need a real file.
	tmpDir := t.TempDir()
	validJPEG := filepath.Join(tmpDir, "valid.jpg")
	if err := os.WriteFile(validJPEG, createTestJPEG(t, 10, 10, 80), 0o644); err != nil {
		t.Fatal(err)
	}
	bmpFile := filepath.Join(tmpDir, "test.bmp")
	if err := os.WriteFile(bmpFile, []byte("not a bmp"), 0o644); err != nil {
		t.Fatal(err)
	}
	corruptJPEG := filepath.Join(tmpDir, "corrupt.jpg")
	if err := os.WriteFile(corruptJPEG, []byte("this is not a jpeg"), 0o644); err != nil {
		t.Fatal(err)
	}
	emptyDir := filepath.Join(tmpDir, "emptydir")
	if err := os.Mkdir(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		args      []string
		wantInErr string // substring expected in error message (empty = just check error is non-nil)
	}{
		{
			name: "引数なし",
			args: []string{"compress"},
		},
		{
			name: "引数過多",
			args: []string{"compress", "file1.jpg", "file2.jpg"},
		},
		{
			name: "存在しないファイル",
			args: []string{"compress", "/nonexistent/path/file.jpg"},
		},
		{
			name: "存在しないディレクトリ",
			args: []string{"compress", "/nonexistent/dir/", "-r"},
		},
		{
			name: "recursiveなしのディレクトリ",
			args: []string{"compress", emptyDir},
		},
		{
			name:      "不正なlevel値",
			args:      []string{"compress", validJPEG, "--level", "ultra"},
			wantInErr: "不正な圧縮レベル",
		},
		{
			name:      "quality範囲外_0未満",
			args:      []string{"compress", validJPEG, "--quality", "-1"},
			wantInErr: "品質は1〜100の範囲",
		},
		{
			name:      "quality範囲外_101",
			args:      []string{"compress", validJPEG, "--quality", "101"},
			wantInErr: "品質は1〜100の範囲",
		},
		{
			name:      "非対応フォーマットbmp",
			args:      []string{"compress", bmpFile},
			wantInErr: "サポートされていない画像形式",
		},
		{
			name:      "破損画像ファイル",
			args:      []string{"compress", corruptJPEG},
			wantInErr: "圧縮に失敗",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCompress(t, tt.args...)
			if err == nil {
				t.Fatal("エラーが返されるべきですが、nilでした")
			}
			if tt.wantInErr != "" && !strings.Contains(err.Error(), tt.wantInErr) {
				t.Errorf("エラーメッセージに %q が含まれていません: %v", tt.wantInErr, err)
			}
		})
	}
}

func TestE2E_ディレクトリ圧縮_一部失敗(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	setupTestDir(t, inputDir, map[string][]byte{
		"good.jpg":    createTestJPEG(t, 50, 50, 90),
		"corrupt.jpg": []byte("this is not a valid jpeg"),
	})

	out, err := executeCompress(t, "compress", inputDir, "-r", "-o", outputDir)

	// Should return error because some files failed.
	if err == nil {
		t.Fatal("一部失敗時にエラーが返されるべきです")
	}

	// Verify the output reports success and failure counts.
	if !strings.Contains(out, "成功 1") {
		t.Errorf("出力に「成功 1」が含まれていません: %s", out)
	}
	if !strings.Contains(out, "失敗 1") {
		t.Errorf("出力に「失敗 1」が含まれていません: %s", out)
	}

	// The valid file should still be compressed.
	verifyImageFile(t, filepath.Join(outputDir, "good.jpg"), "jpeg")
}

// --- Step 7: Output Verification Tests ---

func TestE2E_出力検証_JPEG品質指定vsレベル(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.jpg")
	outLow := filepath.Join(tmpDir, "low.jpg")
	outHigh := filepath.Join(tmpDir, "high.jpg")

	// Create a larger image so quality differences are more apparent.
	jpegData := createTestJPEG(t, 300, 300, 100)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Compress with quality=30.
	_, err := executeCompress(t, "compress", inputPath, "-o", outLow, "--quality", "30")
	if err != nil {
		t.Fatalf("quality=30 compression failed: %v", err)
	}

	// Compress with quality=90.
	_, err = executeCompress(t, "compress", inputPath, "-o", outHigh, "--quality", "90")
	if err != nil {
		t.Fatalf("quality=90 compression failed: %v", err)
	}

	verifyImageFile(t, outLow, "jpeg")
	verifyImageFile(t, outHigh, "jpeg")

	lowInfo, _ := os.Stat(outLow)
	highInfo, _ := os.Stat(outHigh)

	if lowInfo.Size() >= highInfo.Size() {
		t.Errorf("quality=30 のサイズ(%d) >= quality=90 のサイズ(%d)", lowInfo.Size(), highInfo.Size())
	}
}

func TestE2E_出力検証_PNG圧縮レベル(t *testing.T) {
	levels := []string{"low", "medium", "high"}

	for _, lvl := range levels {
		t.Run("level_"+lvl, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "input.png")
			outputPath := filepath.Join(tmpDir, "output.png")

			pngData := createTestPNG(t, 100, 100)
			if err := os.WriteFile(inputPath, pngData, 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := executeCompress(t, "compress", inputPath, "-o", outputPath, "--level", lvl)
			if err != nil {
				t.Fatalf("level=%s compression failed: %v", lvl, err)
			}

			verifyImageFile(t, outputPath, "png")
		})
	}
}
