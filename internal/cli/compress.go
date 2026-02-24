package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/FrontWorksDev/Loki/internal/cli/tui"
	"github.com/FrontWorksDev/Loki/pkg/processor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	quality   int
	level     string
	output    string
	recursive bool
	useTUI    bool
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
	compressCmd.Flags().BoolVar(&useTUI, "tui", false, "TUIモードでプログレスバーを表示する")
}

// bindCompressFlags binds compress command flags to Viper keys.
// Called from initConfig() so bindings are re-established after viper.Reset().
func bindCompressFlags() {
	_ = viper.BindPFlag("compress.quality", compressCmd.Flags().Lookup("quality"))
	_ = viper.BindPFlag("compress.level", compressCmd.Flags().Lookup("level"))
	_ = viper.BindPFlag("compress.output", compressCmd.Flags().Lookup("output"))
	_ = viper.BindPFlag("compress.recursive", compressCmd.Flags().Lookup("recursive"))
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

	q := viper.GetInt("compress.quality")
	l := viper.GetString("compress.level")

	compLevel, err := parseCompressionLevel(l)
	if err != nil {
		return err
	}

	if q != 0 && (q < 1 || q > 100) {
		return fmt.Errorf("品質は1〜100の範囲で指定してください (指定値: %d)", q)
	}

	opts := processor.CompressOptions{
		Quality: q,
		Level:   compLevel,
	}

	if info.IsDir() {
		return compressDirectory(cmd, inputPath, opts)
	}
	return compressSingleFile(cmd, inputPath, opts)
}

func compressSingleFile(cmd *cobra.Command, inputPath string, opts processor.CompressOptions) error {
	format, err := detectFormat(inputPath)
	if err != nil {
		return err
	}

	outputPath := viper.GetString("compress.output")
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
	defer func() { _ = inFile.Close() }()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("出力ファイルの作成に失敗しました: %w", err)
	}

	var proc processor.Processor
	switch format {
	case processor.FormatJPEG:
		proc = processor.NewJPEGProcessor()
	case processor.FormatPNG:
		proc = processor.NewPNGProcessor()
	}

	result, err := proc.Compress(cmd.Context(), inFile, outFile, opts)
	if err != nil {
		_ = outFile.Close()
		_ = os.Remove(outputPath)
		return fmt.Errorf("圧縮に失敗しました: %w", err)
	}

	if err := outFile.Close(); err != nil {
		_ = os.Remove(outputPath)
		return fmt.Errorf("出力ファイルの書き込みに失敗しました: %w", err)
	}

	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "圧縮完了: %s → %s\n", inputPath, outputPath)
	_, _ = fmt.Fprintf(out, "  元サイズ: %d bytes\n", result.OriginalSize)
	_, _ = fmt.Fprintf(out, "  圧縮後: %d bytes\n", result.CompressedSize)
	_, _ = fmt.Fprintf(out, "  削減率: %.1f%%\n", result.SavedPercentage())

	return nil
}

func compressDirectory(cmd *cobra.Command, inputDir string, opts processor.CompressOptions) error {
	r := viper.GetBool("compress.recursive")
	if !r {
		return fmt.Errorf("ディレクトリを処理するには --recursive (-r) フラグが必要です")
	}

	outputDir := viper.GetString("compress.output")
	if outputDir == "" {
		outputDir = filepath.Clean(inputDir) + "_compressed"
	}

	items, err := processor.ScanDirectory(inputDir, outputDir, processor.WithCompressOptions(opts))
	if err != nil {
		return fmt.Errorf("ディレクトリのスキャンに失敗しました: %w", err)
	}

	out := cmd.OutOrStdout()

	if len(items) == 0 {
		_, _ = fmt.Fprintln(out, "対象の画像ファイルが見つかりませんでした")
		return nil
	}

	if useTUI {
		return compressDirectoryWithTUI(cmd, items)
	}
	return compressDirectoryWithText(cmd, items)
}

func compressDirectoryWithText(cmd *cobra.Command, items []processor.BatchItem) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	_, _ = fmt.Fprintf(out, "%d 個の画像ファイルを処理します...\n", len(items))

	var mu sync.Mutex
	bp := processor.NewDefaultBatchProcessor(
		processor.WithProgressCallback(func(p processor.Progress) {
			mu.Lock()
			_, _ = fmt.Fprintf(out, "  [%d/%d] %s\n", p.Completed+p.Failed, p.Total, p.Current)
			mu.Unlock()
		}),
	)

	results, err := bp.ProcessBatch(cmd.Context(), items)
	if err != nil {
		return fmt.Errorf("バッチ処理に失敗しました: %w", err)
	}

	successCount := 0
	failCount := 0
	for _, res := range results {
		if res.IsSuccess() {
			successCount++
		} else {
			failCount++
			_, _ = fmt.Fprintf(errOut, "  エラー: %s: %v\n", res.Item.InputPath, res.Error)
		}
	}

	_, _ = fmt.Fprintf(out, "完了: 成功 %d, 失敗 %d\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("%d 件の画像の圧縮に失敗しました", failCount)
	}

	return nil
}

func compressDirectoryWithTUI(cmd *cobra.Command, items []processor.BatchItem) error {
	m := tui.NewModel()
	p := tea.NewProgram(m)

	go func() {
		p.Send(tui.BatchStartMsg{TotalFiles: len(items)})

		bp := processor.NewDefaultBatchProcessor(
			processor.WithProgressCallback(func(prog processor.Progress) {
				p.Send(tui.ProgressMsg{Progress: prog})
			}),
		)

		results, err := bp.ProcessBatch(cmd.Context(), items)
		if err != nil {
			p.Send(tui.BatchErrorMsg{Err: err})
			return
		}

		p.Send(tui.BatchCompleteMsg{
			Results: results,
		})
	}()

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUIの実行に失敗しました: %w", err)
	}

	fm := finalModel.(tui.Model)
	if fm.Err() != nil {
		return fmt.Errorf("バッチ処理に失敗しました: %w", fm.Err())
	}

	if fm.Failed() > 0 {
		return fmt.Errorf("%d 件の画像の圧縮に失敗しました", fm.Failed())
	}

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
