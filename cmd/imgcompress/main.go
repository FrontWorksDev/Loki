package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/FrontWorksDev/image-compressor/internal/cli"
	"github.com/FrontWorksDev/image-compressor/internal/compressor"
	imgpkg "github.com/FrontWorksDev/image-compressor/internal/image"
)

var version = "1.0.0"

func main() {
	opts, err := cli.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[エラー] %v\n", err)
		os.Exit(1)
	}

	if opts.Version {
		fmt.Printf("imgcompress version %s\n", version)
		os.Exit(0)
	}

	if err := opts.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "[エラー] %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	processor := imgpkg.NewProcessor()
	exitCode := 0

	for _, inputPath := range opts.InputPaths {
		if err := processFile(processor, inputPath, opts); err != nil {
			fmt.Fprintf(os.Stderr, "[エラー] %s: %v\n", inputPath, err)
			exitCode = 1
			continue
		}
	}

	os.Exit(exitCode)
}

func processFile(processor *imgpkg.Processor, inputPath string, opts *cli.Options) error {
	if opts.Verbose {
		fmt.Printf("[処理中] %s\n", inputPath)
	}

	// 入力ファイルのサイズを取得
	inputSize, err := processor.GetFileSize(inputPath)
	if err != nil {
		return fmt.Errorf("ファイルが見つかりません")
	}

	// 画像を読み込み
	img, format, err := processor.Load(inputPath)
	if err != nil {
		return err
	}

	// 圧縮器を取得
	comp := compressor.GetCompressor(format)
	if comp == nil {
		return fmt.Errorf("サポートされていない形式です (対応形式: JPEG, PNG)")
	}

	// 圧縮
	compressed, err := comp.Compress(img, opts.Quality)
	if err != nil {
		return fmt.Errorf("圧縮に失敗しました: %w", err)
	}

	// 出力パスを生成
	outputPath := processor.GenerateOutputPath(inputPath, opts.Suffix)

	// ファイルに書き込み
	if err := os.WriteFile(outputPath, compressed, 0644); err != nil {
		return fmt.Errorf("ファイルの書き込みに失敗しました: %w", err)
	}

	// 出力ファイルのサイズを取得
	outputSize, _ := processor.GetFileSize(outputPath)

	// 結果を表示
	reduction := float64(inputSize-outputSize) / float64(inputSize) * 100
	fmt.Printf("[完了] %s -> %s (%s -> %s, %.0f%%削減)\n",
		inputPath,
		outputPath,
		formatSize(inputSize),
		formatSize(outputSize),
		reduction,
	)

	return nil
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)

	switch {
	case size >= MB:
		return fmt.Sprintf("%.1fMB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.1fKB", float64(size)/KB)
	default:
		return fmt.Sprintf("%dB", size)
	}
}
