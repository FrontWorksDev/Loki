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
	convertFormat    string
	convertQuality   int
	convertLevel     string
	convertOutput    string
	convertRecursive bool
	convertUseTUI    bool
)

var convertCmd = &cobra.Command{
	Use:   "convert <input-path>",
	Short: "画像ファイルまたはディレクトリのフォーマットを変換する",
	Long: `画像ファイルまたはディレクトリのフォーマットを変換します。

対応フォーマット: JPEG (.jpg, .jpeg), PNG (.png), WebP (.webp)

例:
  img-cli convert photo.png --format webp
  img-cli convert photo.jpg -f png -q 90
  img-cli convert images/ -f webp -r
  img-cli convert images/ -f jpeg -r -o images_jpeg/`,
	Args: cobra.ExactArgs(1),
	RunE: runConvert,
}

func init() {
	convertCmd.Flags().StringVarP(&convertFormat, "format", "f", "", "出力フォーマット (jpeg/jpg/png/webp) [必須]")
	convertCmd.Flags().IntVarP(&convertQuality, "quality", "q", 0, "JPEG/WebP品質 (1-100)。0の場合はlevelに基づく")
	convertCmd.Flags().StringVarP(&convertLevel, "level", "l", "medium", "圧縮レベル (low/medium/high)")
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "", "出力パス (省略時は自動生成)")
	convertCmd.Flags().BoolVarP(&convertRecursive, "recursive", "r", false, "ディレクトリを再帰的に処理する")
	convertCmd.Flags().BoolVar(&convertUseTUI, "tui", false, "TUIモードでプログレスバーを表示する")
	_ = convertCmd.MarkFlagRequired("format")
}

// bindConvertFlags binds convert command flags to Viper keys.
// Called from initConfig() so bindings are re-established after viper.Reset().
func bindConvertFlags() {
	_ = viper.BindPFlag("convert.format", convertCmd.Flags().Lookup("format"))
	_ = viper.BindPFlag("convert.quality", convertCmd.Flags().Lookup("quality"))
	_ = viper.BindPFlag("convert.level", convertCmd.Flags().Lookup("level"))
	_ = viper.BindPFlag("convert.output", convertCmd.Flags().Lookup("output"))
	_ = viper.BindPFlag("convert.recursive", convertCmd.Flags().Lookup("recursive"))
}

// parseImageFormat parses a string into an ImageFormat.
func parseImageFormat(s string) (processor.ImageFormat, error) {
	switch strings.ToLower(s) {
	case "jpeg", "jpg":
		return processor.FormatJPEG, nil
	case "png":
		return processor.FormatPNG, nil
	case "webp":
		return processor.FormatWEBP, nil
	default:
		return 0, fmt.Errorf("不正なフォーマットです: %q (jpeg/jpg/png/webp を指定してください)", s)
	}
}

func runConvert(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	info, err := os.Stat(inputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("入力パスが存在しません: %s", inputPath)
		}
		return fmt.Errorf("入力パスの確認に失敗しました: %w", err)
	}

	f := viper.GetString("convert.format")
	targetFormat, err := parseImageFormat(f)
	if err != nil {
		return err
	}

	q := viper.GetInt("convert.quality")
	l := viper.GetString("convert.level")

	compLevel, err := parseCompressionLevel(l)
	if err != nil {
		return err
	}

	if q != 0 && (q < 1 || q > 100) {
		return fmt.Errorf("品質は1〜100の範囲で指定してください (指定値: %d)", q)
	}

	opts := processor.ConvertOptions{
		Format: targetFormat,
		CompressOptions: processor.CompressOptions{
			Quality: q,
			Level:   compLevel,
		},
	}

	if info.IsDir() {
		return convertDirectory(cmd, inputPath, targetFormat, opts)
	}
	return convertSingleFile(cmd, inputPath, targetFormat, opts)
}

func convertSingleFile(cmd *cobra.Command, inputPath string, targetFormat processor.ImageFormat, opts processor.ConvertOptions) error {
	srcFormat, err := detectFormat(inputPath)
	if err != nil {
		return err
	}

	if srcFormat == targetFormat {
		return fmt.Errorf("入力ファイルは既に%sフォーマットです。同一フォーマットの圧縮にはcompressコマンドを使用してください", targetFormat)
	}

	outputPath := viper.GetString("convert.output")
	if outputPath == "" {
		outputPath = defaultConvertOutputPath(inputPath, targetFormat)
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

	// Select processor based on output format.
	var proc processor.Processor
	switch targetFormat {
	case processor.FormatJPEG:
		proc = processor.NewJPEGProcessor()
	case processor.FormatPNG:
		proc = processor.NewPNGProcessor()
	case processor.FormatWEBP:
		proc = processor.NewWEBPProcessor()
	}

	result, err := proc.Convert(cmd.Context(), inFile, outFile, opts)
	if err != nil {
		_ = outFile.Close()
		_ = os.Remove(outputPath)
		return fmt.Errorf("変換に失敗しました: %w", err)
	}

	if err := outFile.Close(); err != nil {
		_ = os.Remove(outputPath)
		return fmt.Errorf("出力ファイルの書き込みに失敗しました: %w", err)
	}

	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "変換完了: %s → %s\n", inputPath, outputPath)
	_, _ = fmt.Fprintf(out, "  元サイズ: %d bytes\n", result.OriginalSize)
	_, _ = fmt.Fprintf(out, "  変換後: %d bytes\n", result.CompressedSize)
	_, _ = fmt.Fprintf(out, "  フォーマット: %s → %s\n", srcFormat, targetFormat)

	return nil
}

// defaultConvertOutputPath generates a default output path by changing the extension to the target format.
func defaultConvertOutputPath(inputPath string, format processor.ImageFormat) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	return base + format.Extension()
}

func convertDirectory(cmd *cobra.Command, inputDir string, targetFormat processor.ImageFormat, opts processor.ConvertOptions) error {
	r := viper.GetBool("convert.recursive")
	if !r {
		return fmt.Errorf("ディレクトリを処理するには --recursive (-r) フラグが必要です")
	}

	outputDir := viper.GetString("convert.output")
	if outputDir == "" {
		outputDir = filepath.Clean(inputDir) + "_converted"
	}

	items, err := processor.ScanDirectoryForConvert(inputDir, outputDir, targetFormat, processor.WithConvertOptions(opts))
	if err != nil {
		return fmt.Errorf("ディレクトリのスキャンに失敗しました: %w", err)
	}

	out := cmd.OutOrStdout()

	if len(items) == 0 {
		_, _ = fmt.Fprintln(out, "変換対象の画像ファイルが見つかりませんでした")
		return nil
	}

	if convertUseTUI {
		return convertDirectoryWithTUI(cmd, items)
	}
	return convertDirectoryWithText(cmd, items)
}

func convertDirectoryWithText(cmd *cobra.Command, items []processor.BatchConvertItem) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	_, _ = fmt.Fprintf(out, "%d 個の画像ファイルを変換します...\n", len(items))

	var mu sync.Mutex
	bp := processor.NewDefaultBatchProcessor(
		processor.WithProgressCallback(func(p processor.Progress) {
			mu.Lock()
			defer mu.Unlock()
			_, _ = fmt.Fprintf(out, "  [%d/%d] %s\n", p.Completed+p.Failed, p.Total, p.Current)
		}),
	)

	results, err := bp.ProcessBatchConvert(cmd.Context(), items)
	if err != nil {
		return fmt.Errorf("バッチ変換に失敗しました: %w", err)
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
		return fmt.Errorf("%d 件の画像の変換に失敗しました", failCount)
	}

	return nil
}

func convertDirectoryWithTUI(cmd *cobra.Command, items []processor.BatchConvertItem) error {
	m := tui.NewModel()
	p := tea.NewProgram(m)

	go func() {
		p.Send(tui.BatchStartMsg{TotalFiles: len(items)})

		bp := processor.NewDefaultBatchProcessor(
			processor.WithProgressCallback(func(prog processor.Progress) {
				p.Send(tui.ProgressMsg{Progress: prog})
			}),
		)

		results, err := bp.ProcessBatchConvert(cmd.Context(), items)
		if err != nil {
			p.Send(tui.BatchErrorMsg{Err: err})
			return
		}

		// Convert BatchConvertResult to BatchResult for TUI compatibility.
		batchResults := make([]processor.BatchResult, len(results))
		for i, r := range results {
			batchResults[i] = processor.BatchResult{
				Item: processor.BatchItem{
					InputPath:  r.Item.InputPath,
					OutputPath: r.Item.OutputPath,
				},
				Result: r.Result,
				Error:  r.Error,
			}
		}

		p.Send(tui.BatchCompleteMsg{
			Results: batchResults,
		})
	}()

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUIの実行に失敗しました: %w", err)
	}

	fm := finalModel.(tui.Model)
	if fm.Err() != nil {
		return fmt.Errorf("バッチ変換に失敗しました: %w", fm.Err())
	}

	if fm.Failed() > 0 {
		return fmt.Errorf("%d 件の画像の変換に失敗しました", fm.Failed())
	}

	return nil
}
