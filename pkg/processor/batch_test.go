package processor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// writeTestFile writes test image data to a file in the given directory.
func writeTestFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	return path
}

func TestNewDefaultBatchProcessor(t *testing.T) {
	tests := []struct {
		name           string
		opts           []BatchProcessorOption
		wantMaxWorkers int
	}{
		{
			name:           "デフォルト設定",
			wantMaxWorkers: runtime.NumCPU(),
		},
		{
			name:           "ワーカー数を指定",
			opts:           []BatchProcessorOption{WithMaxWorkers(4)},
			wantMaxWorkers: 4,
		},
		{
			name:           "ワーカー数0は1に補正される",
			opts:           []BatchProcessorOption{WithMaxWorkers(0)},
			wantMaxWorkers: 1,
		},
		{
			name:           "負のワーカー数は1に補正される",
			opts:           []BatchProcessorOption{WithMaxWorkers(-1)},
			wantMaxWorkers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bp := NewDefaultBatchProcessor(tt.opts...)
			if bp == nil {
				t.Fatal("NewDefaultBatchProcessor() returned nil")
			}
			if bp.jpegProc == nil {
				t.Error("jpegProc is nil")
			}
			if bp.pngProc == nil {
				t.Error("pngProc is nil")
			}
			if bp.webpProc == nil {
				t.Error("webpProc is nil")
			}
			if bp.maxWorkers != tt.wantMaxWorkers {
				t.Errorf("maxWorkers = %d, want %d", bp.maxWorkers, tt.wantMaxWorkers)
			}
		})
	}
}

func TestDefaultBatchProcessor_ProcessBatch_空バッチ(t *testing.T) {
	bp := NewDefaultBatchProcessor()
	results, err := bp.ProcessBatch(context.Background(), []BatchItem{})
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("ProcessBatch() returned %d results, want 0", len(results))
	}
}

func TestDefaultBatchProcessor_ProcessBatch_単一ファイル(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 100, 100, 95)
	inputPath := writeTestFile(t, inputDir, "test.jpg", jpegData)
	outputPath := filepath.Join(outputDir, "test.jpg")

	bp := NewDefaultBatchProcessor(WithMaxWorkers(1))
	items := []BatchItem{
		{
			InputPath:  inputPath,
			OutputPath: outputPath,
			Options:    DefaultCompressOptions(),
		},
	}

	results, err := bp.ProcessBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("ProcessBatch() returned %d results, want 1", len(results))
	}
	if !results[0].IsSuccess() {
		t.Fatalf("result is not success: %v", results[0].Error)
	}
	if results[0].Result.OriginalSize == 0 {
		t.Error("OriginalSize should not be 0")
	}
	if results[0].Result.CompressedSize == 0 {
		t.Error("CompressedSize should not be 0")
	}

	// Verify output file exists.
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("output file was not created")
	}
}

func TestDefaultBatchProcessor_ProcessBatch_複数ファイル(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 50, 50, 90)
	pngData := createTestPNG(t, 50, 50)

	inputJPEG := writeTestFile(t, inputDir, "photo.jpg", jpegData)
	inputPNG := writeTestFile(t, inputDir, "icon.png", pngData)

	items := []BatchItem{
		{
			InputPath:  inputJPEG,
			OutputPath: filepath.Join(outputDir, "photo.jpg"),
			Options:    DefaultCompressOptions(),
		},
		{
			InputPath:  inputPNG,
			OutputPath: filepath.Join(outputDir, "icon.png"),
			Options:    DefaultCompressOptions(),
		},
	}

	bp := NewDefaultBatchProcessor(WithMaxWorkers(2))
	results, err := bp.ProcessBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("ProcessBatch() returned %d results, want 2", len(results))
	}

	for i, r := range results {
		if !r.IsSuccess() {
			t.Errorf("result[%d] is not success: %v", i, r.Error)
		}
	}
}

func TestDefaultBatchProcessor_ProcessBatch_存在しないファイル(t *testing.T) {
	bp := NewDefaultBatchProcessor(WithMaxWorkers(1))
	items := []BatchItem{
		{
			InputPath:  "/nonexistent/file.jpg",
			OutputPath: filepath.Join(t.TempDir(), "out.jpg"),
			Options:    DefaultCompressOptions(),
		},
	}

	results, err := bp.ProcessBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("ProcessBatch() returned %d results, want 1", len(results))
	}
	if results[0].IsSuccess() {
		t.Error("result should be failure for nonexistent file")
	}
	if results[0].Error == nil {
		t.Error("result.Error should not be nil")
	}
}

func TestDefaultBatchProcessor_ProcessBatch_サポート外形式(t *testing.T) {
	inputDir := t.TempDir()
	inputPath := writeTestFile(t, inputDir, "file.bmp", []byte("fake bmp data"))

	bp := NewDefaultBatchProcessor(WithMaxWorkers(1))
	items := []BatchItem{
		{
			InputPath:  inputPath,
			OutputPath: filepath.Join(t.TempDir(), "file.bmp"),
			Options:    DefaultCompressOptions(),
		},
	}

	results, err := bp.ProcessBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if results[0].IsSuccess() {
		t.Error("result should be failure for unsupported format")
	}
}

func TestDefaultBatchProcessor_ProcessBatch_破損画像(t *testing.T) {
	inputDir := t.TempDir()
	inputPath := writeTestFile(t, inputDir, "corrupt.jpg", []byte("not a valid jpeg"))

	bp := NewDefaultBatchProcessor(WithMaxWorkers(1))
	items := []BatchItem{
		{
			InputPath:  inputPath,
			OutputPath: filepath.Join(t.TempDir(), "corrupt.jpg"),
			Options:    DefaultCompressOptions(),
		},
	}

	results, err := bp.ProcessBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if results[0].IsSuccess() {
		t.Error("result should be failure for corrupt image")
	}
}

func TestDefaultBatchProcessor_ProcessBatch_コンテキストキャンセル(t *testing.T) {
	inputDir := t.TempDir()
	jpegData := createTestJPEG(t, 100, 100, 95)

	var items []BatchItem
	for i := range 10 {
		name := filepath.Join(inputDir, "img"+string(rune('0'+i))+".jpg")
		writeTestFile(t, inputDir, "img"+string(rune('0'+i))+".jpg", jpegData)
		items = append(items, BatchItem{
			InputPath:  name,
			OutputPath: filepath.Join(t.TempDir(), "img"+string(rune('0'+i))+".jpg"),
			Options:    DefaultCompressOptions(),
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	bp := NewDefaultBatchProcessor(WithMaxWorkers(1))
	results, err := bp.ProcessBatch(ctx, items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}

	// All results should be populated with an error after cancellation.
	for i, r := range results {
		if r.Error == nil {
			t.Errorf("result[%d] expected error due to context cancellation, got nil", i)
		}
	}
}

func TestDefaultBatchProcessor_ProcessBatch_コンテキストタイムアウト(t *testing.T) {
	inputDir := t.TempDir()
	jpegData := createTestJPEG(t, 100, 100, 95)
	inputPath := writeTestFile(t, inputDir, "test.jpg", jpegData)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure timeout.

	bp := NewDefaultBatchProcessor(WithMaxWorkers(1))
	items := []BatchItem{
		{
			InputPath:  inputPath,
			OutputPath: filepath.Join(t.TempDir(), "test.jpg"),
			Options:    DefaultCompressOptions(),
		},
	}

	results, err := bp.ProcessBatch(ctx, items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("ProcessBatch() returned %d results, want 1", len(results))
	}
	if results[0].IsSuccess() {
		t.Error("result should be failure due to timeout")
	}
}

func TestDefaultBatchProcessor_ProcessBatch_シーケンシャル(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 50, 50, 90)
	input1 := writeTestFile(t, inputDir, "a.jpg", jpegData)
	input2 := writeTestFile(t, inputDir, "b.jpg", jpegData)

	items := []BatchItem{
		{InputPath: input1, OutputPath: filepath.Join(outputDir, "a.jpg"), Options: DefaultCompressOptions()},
		{InputPath: input2, OutputPath: filepath.Join(outputDir, "b.jpg"), Options: DefaultCompressOptions()},
	}

	bp := NewDefaultBatchProcessor(WithMaxWorkers(1))
	results, err := bp.ProcessBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}

	for i, r := range results {
		if !r.IsSuccess() {
			t.Errorf("result[%d] is not success: %v", i, r.Error)
		}
	}
}

func TestDefaultBatchProcessor_ProcessBatch_複数ワーカー(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 50, 50, 90)

	var items []BatchItem
	for i := range 8 {
		name := "img" + string(rune('a'+i)) + ".jpg"
		inputPath := writeTestFile(t, inputDir, name, jpegData)
		items = append(items, BatchItem{
			InputPath:  inputPath,
			OutputPath: filepath.Join(outputDir, name),
			Options:    DefaultCompressOptions(),
		})
	}

	bp := NewDefaultBatchProcessor(WithMaxWorkers(4))
	results, err := bp.ProcessBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if len(results) != 8 {
		t.Fatalf("ProcessBatch() returned %d results, want 8", len(results))
	}

	for i, r := range results {
		if !r.IsSuccess() {
			t.Errorf("result[%d] is not success: %v", i, r.Error)
		}
	}
}

func TestDefaultBatchProcessor_ProcessBatch_進捗通知(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 50, 50, 90)
	input1 := writeTestFile(t, inputDir, "a.jpg", jpegData)
	input2 := writeTestFile(t, inputDir, "b.jpg", jpegData)
	input3 := writeTestFile(t, inputDir, "c.jpg", jpegData)

	var mu sync.Mutex
	var progressUpdates []Progress

	bp := NewDefaultBatchProcessor(
		WithMaxWorkers(1),
		WithProgressCallback(func(p Progress) {
			mu.Lock()
			defer mu.Unlock()
			progressUpdates = append(progressUpdates, p)
		}),
	)

	items := []BatchItem{
		{InputPath: input1, OutputPath: filepath.Join(outputDir, "a.jpg"), Options: DefaultCompressOptions()},
		{InputPath: input2, OutputPath: filepath.Join(outputDir, "b.jpg"), Options: DefaultCompressOptions()},
		{InputPath: input3, OutputPath: filepath.Join(outputDir, "c.jpg"), Options: DefaultCompressOptions()},
	}

	results, err := bp.ProcessBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}

	for i, r := range results {
		if !r.IsSuccess() {
			t.Errorf("result[%d] is not success: %v", i, r.Error)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if len(progressUpdates) != 3 {
		t.Fatalf("got %d progress updates, want 3", len(progressUpdates))
	}

	// Verify final progress state.
	last := progressUpdates[len(progressUpdates)-1]
	if last.Total != 3 {
		t.Errorf("Total = %d, want 3", last.Total)
	}
	if last.Completed != 3 {
		t.Errorf("Completed = %d, want 3", last.Completed)
	}
	if last.Failed != 0 {
		t.Errorf("Failed = %d, want 0", last.Failed)
	}
}

func TestDefaultBatchProcessor_ProcessBatch_進捗通知_失敗含む(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 50, 50, 90)
	input1 := writeTestFile(t, inputDir, "good.jpg", jpegData)
	input2 := writeTestFile(t, inputDir, "bad.jpg", []byte("not an image"))

	var mu sync.Mutex
	var progressUpdates []Progress

	bp := NewDefaultBatchProcessor(
		WithMaxWorkers(1),
		WithProgressCallback(func(p Progress) {
			mu.Lock()
			defer mu.Unlock()
			progressUpdates = append(progressUpdates, p)
		}),
	)

	items := []BatchItem{
		{InputPath: input1, OutputPath: filepath.Join(outputDir, "good.jpg"), Options: DefaultCompressOptions()},
		{InputPath: input2, OutputPath: filepath.Join(outputDir, "bad.jpg"), Options: DefaultCompressOptions()},
	}

	_, err := bp.ProcessBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	last := progressUpdates[len(progressUpdates)-1]
	if last.Total != 2 {
		t.Errorf("Total = %d, want 2", last.Total)
	}
	if last.Completed+last.Failed != 2 {
		t.Errorf("Completed(%d) + Failed(%d) != 2", last.Completed, last.Failed)
	}
	if last.Failed == 0 {
		t.Error("Failed should be > 0")
	}
}

func TestDetectFormatFromPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantFormat ImageFormat
		wantErr    bool
	}{
		{name: ".jpg拡張子", path: "photo.jpg", wantFormat: FormatJPEG},
		{name: ".jpeg拡張子", path: "photo.jpeg", wantFormat: FormatJPEG},
		{name: ".png拡張子", path: "icon.png", wantFormat: FormatPNG},
		{name: ".JPG大文字", path: "PHOTO.JPG", wantFormat: FormatJPEG},
		{name: ".JPEG大文字", path: "PHOTO.JPEG", wantFormat: FormatJPEG},
		{name: ".PNG大文字", path: "ICON.PNG", wantFormat: FormatPNG},
		{name: "パス付き", path: "/path/to/photo.jpg", wantFormat: FormatJPEG},
		{name: "未対応拡張子_bmp", path: "file.bmp", wantErr: true},
		{name: "未対応拡張子_gif", path: "file.gif", wantErr: true},
		{name: ".webp拡張子", path: "file.webp", wantFormat: FormatWEBP},
		{name: ".WEBP大文字", path: "FILE.WEBP", wantFormat: FormatWEBP},
		{name: "拡張子なし", path: "noext", wantErr: true},
		{name: "空文字", path: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := detectFormatFromPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("detectFormatFromPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr && format != tt.wantFormat {
				t.Errorf("detectFormatFromPath(%q) = %v, want %v", tt.path, format, tt.wantFormat)
			}
		})
	}
}

func TestScanDirectory_空ディレクトリ(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	items, err := ScanDirectory(inputDir, outputDir)
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("ScanDirectory() returned %d items, want 0", len(items))
	}
}

func TestScanDirectory_混合ファイル(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 10, 10, 80)
	pngData := createTestPNG(t, 10, 10)
	webpData := createTestWEBP(t, 10, 10, 80)

	writeTestFile(t, inputDir, "photo.jpg", jpegData)
	writeTestFile(t, inputDir, "icon.png", pngData)
	writeTestFile(t, inputDir, "image.webp", webpData)
	writeTestFile(t, inputDir, "readme.txt", []byte("hello"))
	writeTestFile(t, inputDir, "data.csv", []byte("a,b,c"))

	items, err := ScanDirectory(inputDir, outputDir)
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}
	if len(items) != 3 {
		t.Errorf("ScanDirectory() returned %d items, want 3", len(items))
	}

	// Verify that only image files are included.
	for _, item := range items {
		ext := filepath.Ext(item.InputPath)
		if ext != ".jpg" && ext != ".png" && ext != ".webp" {
			t.Errorf("unexpected file extension: %s", ext)
		}
	}
}

func TestScanDirectory_サブディレクトリ(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 10, 10, 80)
	pngData := createTestPNG(t, 10, 10)

	writeTestFile(t, inputDir, "top.jpg", jpegData)
	writeTestFile(t, inputDir, "sub/nested.png", pngData)
	writeTestFile(t, inputDir, "sub/deep/photo.jpeg", jpegData)

	items, err := ScanDirectory(inputDir, outputDir)
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}
	if len(items) != 3 {
		t.Errorf("ScanDirectory() returned %d items, want 3", len(items))
	}

	// Verify output paths maintain directory structure.
	for _, item := range items {
		relInput, _ := filepath.Rel(inputDir, item.InputPath)
		relOutput, _ := filepath.Rel(outputDir, item.OutputPath)
		if relInput != relOutput {
			t.Errorf("directory structure not maintained: input=%s, output=%s", relInput, relOutput)
		}
	}
}

func TestScanDirectory_存在しないディレクトリ(t *testing.T) {
	_, err := ScanDirectory("/nonexistent/dir", t.TempDir())
	if err == nil {
		t.Error("ScanDirectory() should return error for nonexistent directory")
	}
}

func TestScanDirectory_WithCompressOptions(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 10, 10, 80)
	writeTestFile(t, inputDir, "test.jpg", jpegData)

	opts := CompressOptions{Quality: 50, Level: CompressionLow}
	items, err := ScanDirectory(inputDir, outputDir, WithCompressOptions(opts))
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("ScanDirectory() returned %d items, want 1", len(items))
	}
	if items[0].Options.Quality != 50 {
		t.Errorf("Quality = %d, want 50", items[0].Options.Quality)
	}
	if items[0].Options.Level != CompressionLow {
		t.Errorf("Level = %v, want %v", items[0].Options.Level, CompressionLow)
	}
}

func TestScanDirectory_大文字拡張子(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 10, 10, 80)
	pngData := createTestPNG(t, 10, 10)

	writeTestFile(t, inputDir, "PHOTO.JPG", jpegData)
	writeTestFile(t, inputDir, "IMAGE.JPEG", jpegData)
	writeTestFile(t, inputDir, "ICON.PNG", pngData)

	items, err := ScanDirectory(inputDir, outputDir)
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}
	if len(items) != 3 {
		t.Errorf("ScanDirectory() returned %d items, want 3", len(items))
	}
}

// --- BatchConvert Tests ---

func TestDefaultBatchProcessor_ProcessBatchConvert_空バッチ(t *testing.T) {
	bp := NewDefaultBatchProcessor()
	results, err := bp.ProcessBatchConvert(context.Background(), []BatchConvertItem{})
	if err != nil {
		t.Fatalf("ProcessBatchConvert() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("ProcessBatchConvert() returned %d results, want 0", len(results))
	}
}

func TestDefaultBatchProcessor_ProcessBatchConvert_単一ファイル(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	pngData := createTestPNG(t, 100, 100)
	inputPath := writeTestFile(t, inputDir, "test.png", pngData)
	outputPath := filepath.Join(outputDir, "test.jpg")

	bp := NewDefaultBatchProcessor(WithMaxWorkers(1))
	items := []BatchConvertItem{
		{
			InputPath:  inputPath,
			OutputPath: outputPath,
			Options:    DefaultConvertOptions(FormatJPEG),
		},
	}

	results, err := bp.ProcessBatchConvert(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatchConvert() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("ProcessBatchConvert() returned %d results, want 1", len(results))
	}
	if !results[0].IsSuccess() {
		t.Fatalf("result is not success: %v", results[0].Error)
	}
	if results[0].Result.OriginalSize == 0 {
		t.Error("OriginalSize should not be 0")
	}
	if results[0].Result.CompressedSize == 0 {
		t.Error("CompressedSize should not be 0")
	}

	// Verify output file exists.
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("output file was not created")
	}
}

func TestDefaultBatchProcessor_ProcessBatchConvert_複数ファイル(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 50, 50, 90)
	pngData := createTestPNG(t, 50, 50)

	inputJPEG := writeTestFile(t, inputDir, "photo.jpg", jpegData)
	inputPNG := writeTestFile(t, inputDir, "icon.png", pngData)

	items := []BatchConvertItem{
		{
			InputPath:  inputJPEG,
			OutputPath: filepath.Join(outputDir, "photo.webp"),
			Options:    DefaultConvertOptions(FormatWEBP),
		},
		{
			InputPath:  inputPNG,
			OutputPath: filepath.Join(outputDir, "icon.webp"),
			Options:    DefaultConvertOptions(FormatWEBP),
		},
	}

	bp := NewDefaultBatchProcessor(WithMaxWorkers(2))
	results, err := bp.ProcessBatchConvert(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatchConvert() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("ProcessBatchConvert() returned %d results, want 2", len(results))
	}

	for i, r := range results {
		if !r.IsSuccess() {
			t.Errorf("result[%d] is not success: %v", i, r.Error)
		}
	}
}

func TestDefaultBatchProcessor_ProcessBatchConvert_コンテキストキャンセル(t *testing.T) {
	inputDir := t.TempDir()
	pngData := createTestPNG(t, 100, 100)

	var items []BatchConvertItem
	for i := range 10 {
		name := "img" + string(rune('0'+i)) + ".png"
		writeTestFile(t, inputDir, name, pngData)
		items = append(items, BatchConvertItem{
			InputPath:  filepath.Join(inputDir, name),
			OutputPath: filepath.Join(t.TempDir(), "img"+string(rune('0'+i))+".jpg"),
			Options:    DefaultConvertOptions(FormatJPEG),
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	bp := NewDefaultBatchProcessor(WithMaxWorkers(1))
	results, err := bp.ProcessBatchConvert(ctx, items)
	if err != nil {
		t.Fatalf("ProcessBatchConvert() error = %v", err)
	}

	for i, r := range results {
		if r.Error == nil {
			t.Errorf("result[%d] expected error due to context cancellation, got nil", i)
		}
	}
}

func TestDefaultBatchProcessor_ProcessBatchConvert_進捗通知(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 50, 50, 90)
	input1 := writeTestFile(t, inputDir, "a.jpg", jpegData)
	input2 := writeTestFile(t, inputDir, "b.jpg", jpegData)

	var mu sync.Mutex
	var progressUpdates []Progress

	bp := NewDefaultBatchProcessor(
		WithMaxWorkers(1),
		WithProgressCallback(func(p Progress) {
			mu.Lock()
			defer mu.Unlock()
			progressUpdates = append(progressUpdates, p)
		}),
	)

	items := []BatchConvertItem{
		{InputPath: input1, OutputPath: filepath.Join(outputDir, "a.png"), Options: DefaultConvertOptions(FormatPNG)},
		{InputPath: input2, OutputPath: filepath.Join(outputDir, "b.png"), Options: DefaultConvertOptions(FormatPNG)},
	}

	results, err := bp.ProcessBatchConvert(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatchConvert() error = %v", err)
	}

	for i, r := range results {
		if !r.IsSuccess() {
			t.Errorf("result[%d] is not success: %v", i, r.Error)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if len(progressUpdates) != 2 {
		t.Fatalf("got %d progress updates, want 2", len(progressUpdates))
	}

	last := progressUpdates[len(progressUpdates)-1]
	if last.Total != 2 {
		t.Errorf("Total = %d, want 2", last.Total)
	}
	if last.Completed != 2 {
		t.Errorf("Completed = %d, want 2", last.Completed)
	}
}

func TestScanDirectoryForConvert_空ディレクトリ(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	items, err := ScanDirectoryForConvert(inputDir, outputDir, FormatWEBP)
	if err != nil {
		t.Fatalf("ScanDirectoryForConvert() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("ScanDirectoryForConvert() returned %d items, want 0", len(items))
	}
}

func TestScanDirectoryForConvert_拡張子変更確認(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 10, 10, 80)
	pngData := createTestPNG(t, 10, 10)

	writeTestFile(t, inputDir, "photo.jpg", jpegData)
	writeTestFile(t, inputDir, "icon.png", pngData)

	items, err := ScanDirectoryForConvert(inputDir, outputDir, FormatWEBP)
	if err != nil {
		t.Fatalf("ScanDirectoryForConvert() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("ScanDirectoryForConvert() returned %d items, want 2", len(items))
	}

	// Verify output paths have .webp extension.
	for _, item := range items {
		ext := filepath.Ext(item.OutputPath)
		if ext != ".webp" {
			t.Errorf("output extension = %s, want .webp (path: %s)", ext, item.OutputPath)
		}
	}
}

func TestScanDirectoryForConvert_同一フォーマットスキップ(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 10, 10, 80)
	pngData := createTestPNG(t, 10, 10)
	webpData := createTestWEBP(t, 10, 10, 80)

	writeTestFile(t, inputDir, "photo.jpg", jpegData)
	writeTestFile(t, inputDir, "icon.png", pngData)
	writeTestFile(t, inputDir, "image.webp", webpData)

	// Target format is WebP, so image.webp should be skipped.
	items, err := ScanDirectoryForConvert(inputDir, outputDir, FormatWEBP)
	if err != nil {
		t.Fatalf("ScanDirectoryForConvert() error = %v", err)
	}
	if len(items) != 2 {
		t.Errorf("ScanDirectoryForConvert() returned %d items, want 2 (webp should be skipped)", len(items))
	}

	for _, item := range items {
		if filepath.Ext(item.InputPath) == ".webp" {
			t.Error("WebP file should have been skipped")
		}
	}
}

func TestScanDirectoryForConvert_サブディレクトリ(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 10, 10, 80)
	pngData := createTestPNG(t, 10, 10)

	writeTestFile(t, inputDir, "top.jpg", jpegData)
	writeTestFile(t, inputDir, "sub/nested.png", pngData)
	writeTestFile(t, inputDir, "sub/deep/photo.jpeg", jpegData)

	items, err := ScanDirectoryForConvert(inputDir, outputDir, FormatWEBP)
	if err != nil {
		t.Fatalf("ScanDirectoryForConvert() error = %v", err)
	}
	if len(items) != 3 {
		t.Errorf("ScanDirectoryForConvert() returned %d items, want 3", len(items))
	}

	// Verify output paths maintain directory structure with new extensions.
	for _, item := range items {
		relInput, _ := filepath.Rel(inputDir, item.InputPath)
		relOutput, _ := filepath.Rel(outputDir, item.OutputPath)

		inputNoExt := strings.TrimSuffix(relInput, filepath.Ext(relInput))
		outputNoExt := strings.TrimSuffix(relOutput, filepath.Ext(relOutput))
		if inputNoExt != outputNoExt {
			t.Errorf("directory structure not maintained: input=%s, output=%s", relInput, relOutput)
		}

		if filepath.Ext(item.OutputPath) != ".webp" {
			t.Errorf("output extension should be .webp, got %s", filepath.Ext(item.OutputPath))
		}
	}
}

func TestScanDirectoryForConvert_WithConvertOptions(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 10, 10, 80)
	writeTestFile(t, inputDir, "test.jpg", jpegData)

	opts := ConvertOptions{
		Format: FormatPNG,
		CompressOptions: CompressOptions{
			Quality: 50,
			Level:   CompressionLow,
		},
	}
	items, err := ScanDirectoryForConvert(inputDir, outputDir, FormatPNG, WithConvertOptions(opts))
	if err != nil {
		t.Fatalf("ScanDirectoryForConvert() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("ScanDirectoryForConvert() returned %d items, want 1", len(items))
	}
	if items[0].Options.Quality != 50 {
		t.Errorf("Quality = %d, want 50", items[0].Options.Quality)
	}
	if items[0].Options.Level != CompressionLow {
		t.Errorf("Level = %v, want %v", items[0].Options.Level, CompressionLow)
	}
}

func TestScanDirectory_非ディレクトリ入力(t *testing.T) {
	// Create a regular file and try to scan it as a directory.
	tmpDir := t.TempDir()
	filePath := writeTestFile(t, tmpDir, "notadir.jpg", createTestJPEG(t, 10, 10, 80))

	_, err := ScanDirectory(filePath, t.TempDir())
	if err == nil {
		t.Error("ScanDirectory() should return error for non-directory input")
	}
}

func TestDefaultBatchProcessor_統合テスト(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	jpegData := createTestJPEG(t, 100, 100, 95)
	pngData := createTestPNG(t, 100, 100)
	webpData := createTestWEBP(t, 100, 100, 95)

	writeTestFile(t, inputDir, "photo1.jpg", jpegData)
	writeTestFile(t, inputDir, "photo2.jpeg", jpegData)
	writeTestFile(t, inputDir, "icon.png", pngData)
	writeTestFile(t, inputDir, "image.webp", webpData)
	writeTestFile(t, inputDir, "sub/nested.jpg", jpegData)
	writeTestFile(t, inputDir, "readme.txt", []byte("not an image"))

	// Scan directory.
	items, err := ScanDirectory(inputDir, outputDir)
	if err != nil {
		t.Fatalf("ScanDirectory() error = %v", err)
	}
	if len(items) != 5 {
		t.Fatalf("ScanDirectory() returned %d items, want 5", len(items))
	}

	// Process batch.
	bp := NewDefaultBatchProcessor(WithMaxWorkers(2))
	results, err := bp.ProcessBatch(context.Background(), items)
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("ProcessBatch() returned %d results, want 5", len(results))
	}

	// Verify all succeeded.
	for i, r := range results {
		if !r.IsSuccess() {
			t.Errorf("result[%d] (%s) is not success: %v", i, r.Item.InputPath, r.Error)
			continue
		}
		if r.Result.OriginalSize == 0 {
			t.Errorf("result[%d] OriginalSize should not be 0", i)
		}
		if r.Result.CompressedSize == 0 {
			t.Errorf("result[%d] CompressedSize should not be 0", i)
		}
	}

	// Verify output files exist.
	for _, r := range results {
		if r.IsSuccess() {
			if _, err := os.Stat(r.Item.OutputPath); os.IsNotExist(err) {
				t.Errorf("output file not created: %s", r.Item.OutputPath)
			}
		}
	}

	// Verify directory structure is maintained.
	nestedOutput := filepath.Join(outputDir, "sub", "nested.jpg")
	if _, err := os.Stat(nestedOutput); os.IsNotExist(err) {
		t.Errorf("nested output file not created: %s", nestedOutput)
	}
}
