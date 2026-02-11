package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FrontWorksDev/Loki/pkg/processor"
	"github.com/spf13/cobra"
)

var (
	quality   int
	level     string
	output    string
	recursive bool
)

var compressCmd = &cobra.Command{
	Use:   "compress <input-path>",
	Short: "画像ファイルまたはディレクトリを圧縮する",
	Long: `画像ファイルまたはディレクトリを圧縮します。

対応フォーマット: JPEG (.jpg, .jpeg), PNG (.png)

例:
  img-cli compress photo.jpg
  img-cli compress photo.jpg -q 70
  img-cli compress photo.jpg -l high -o output.jpg
  img-cli compress images/ -r -o images_compressed/`,
	Args: cobra.ExactArgs(1),
	RunE: runCompress,
}

func init() {
	compressCmd.Flags().IntVarP(&quality, "quality", "q", 0, "JPEG品質 (1-100)。0の場合はlevelに基づく")
	compressCmd.Flags().StringVarP(&level, "level", "l", "medium", "圧縮レベル (low/medium/high)")
	compressCmd.Flags().StringVarP(&output, "output", "o", "", "出力パス (省略時は自動生成)")
	compressCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "ディレクトリを再帰的に処理する")
}

func runCompress(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	info, err := os.Stat(inputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("入力パスが存在しません: %s", inputPath)
		}
		return fmt.Errorf("入力パスの確認に失敗しました: %w", err)
	}

	compLevel, err := parseCompressionLevel(level)
	if err != nil {
		return err
	}

	opts := processor.CompressOptions{
		Quality: quality,
		Level:   compLevel,
	}

	if info.IsDir() {
		return compressDirectory(inputPath, opts)
	}
	return compressSingleFile(inputPath, opts)
}

func compressSingleFile(inputPath string, opts processor.CompressOptions) error {
	format, err := detectFormat(inputPath)
	if err != nil {
		return err
	}

	outputPath := output
	if outputPath == "" {
		outputPath = defaultOutputPath(inputPath)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("出力ディレクトリの作成に失敗しました: %w", err)
	}

	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("入力ファイルを開けません: %w", err)
	}
	defer inFile.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("出力ファイルの作成に失敗しました: %w", err)
	}
	defer outFile.Close()

	var proc processor.Processor
	switch format {
	case processor.FormatJPEG:
		proc = processor.NewJPEGProcessor()
	case processor.FormatPNG:
		proc = processor.NewPNGProcessor()
	}

	result, err := proc.Compress(context.Background(), inFile, outFile, opts)
	if err != nil {
		outFile.Close()
		os.Remove(outputPath)
		return fmt.Errorf("圧縮に失敗しました: %w", err)
	}

	fmt.Fprintf(os.Stdout, "圧縮完了: %s → %s\n", inputPath, outputPath)
	fmt.Fprintf(os.Stdout, "  元サイズ: %d bytes\n", result.OriginalSize)
	fmt.Fprintf(os.Stdout, "  圧縮後: %d bytes\n", result.CompressedSize)
	fmt.Fprintf(os.Stdout, "  削減率: %.1f%%\n", result.SavedPercentage())

	return nil
}

func compressDirectory(inputDir string, opts processor.CompressOptions) error {
	if !recursive {
		return fmt.Errorf("ディレクトリを処理するには --recursive (-r) フラグが必要です")
	}

	outputDir := output
	if outputDir == "" {
		outputDir = strings.TrimRight(inputDir, string(filepath.Separator)) + "_compressed"
	}

	items, err := processor.ScanDirectory(inputDir, outputDir, processor.WithCompressOptions(opts))
	if err != nil {
		return fmt.Errorf("ディレクトリのスキャンに失敗しました: %w", err)
	}

	if len(items) == 0 {
		fmt.Fprintln(os.Stdout, "対象の画像ファイルが見つかりませんでした")
		return nil
	}

	fmt.Fprintf(os.Stdout, "%d 個の画像ファイルを処理します...\n", len(items))

	bp := processor.NewDefaultBatchProcessor(
		processor.WithProgressCallback(func(p processor.Progress) {
			fmt.Fprintf(os.Stdout, "  [%d/%d] %s\n", p.Completed+p.Failed, p.Total, p.Current)
		}),
	)

	results, err := bp.ProcessBatch(context.Background(), items)
	if err != nil {
		return fmt.Errorf("バッチ処理に失敗しました: %w", err)
	}

	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.IsSuccess() {
			successCount++
		} else {
			failCount++
			fmt.Fprintf(os.Stderr, "  エラー: %s: %v\n", r.Item.InputPath, r.Error)
		}
	}

	fmt.Fprintf(os.Stdout, "完了: 成功 %d, 失敗 %d\n", successCount, failCount)

	return nil
}

// parseCompressionLevel converts a string to a CompressionLevel.
func parseCompressionLevel(s string) (processor.CompressionLevel, error) {
	switch strings.ToLower(s) {
	case "low":
		return processor.CompressionLow, nil
	case "medium":
		return processor.CompressionMedium, nil
	case "high":
		return processor.CompressionHigh, nil
	default:
		return 0, fmt.Errorf("不正な圧縮レベルです: %q (low/medium/high を指定してください)", s)
	}
}

// detectFormat detects the image format from a file path extension.
func detectFormat(path string) (processor.ImageFormat, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return processor.FormatJPEG, nil
	case ".png":
		return processor.FormatPNG, nil
	default:
		return 0, fmt.Errorf("サポートされていない画像形式です: %s", ext)
	}
}

// defaultOutputPath generates a default output path by appending "_compressed" before the extension.
func defaultOutputPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	return base + "_compressed" + ext
}
