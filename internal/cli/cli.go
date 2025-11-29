package cli

import (
	"flag"
	"fmt"
	"os"
)

const (
	DefaultQuality = 80
	DefaultSuffix  = "_compressed"
)

// Options はCLIオプションを保持する
type Options struct {
	InputPaths []string
	Quality    int
	Suffix     string
	Verbose    bool
	Version    bool
}

// Parse はコマンドライン引数をパースする
func Parse() (*Options, error) {
	opts := &Options{}

	flag.IntVar(&opts.Quality, "q", DefaultQuality, "圧縮品質 (1-100)")
	flag.IntVar(&opts.Quality, "quality", DefaultQuality, "圧縮品質 (1-100)")
	flag.StringVar(&opts.Suffix, "s", DefaultSuffix, "出力ファイルのサフィックス")
	flag.StringVar(&opts.Suffix, "suffix", DefaultSuffix, "出力ファイルのサフィックス")
	flag.BoolVar(&opts.Verbose, "v", false, "詳細出力")
	flag.BoolVar(&opts.Verbose, "verbose", false, "詳細出力")
	flag.BoolVar(&opts.Version, "version", false, "バージョン表示")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "使用方法: imgcompress [オプション] <画像ファイル>...\n\n")
		fmt.Fprintf(os.Stderr, "JPEG/PNG画像を圧縮して同じフォルダに出力します。\n\n")
		fmt.Fprintf(os.Stderr, "オプション:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n例:\n")
		fmt.Fprintf(os.Stderr, "  imgcompress image.jpg\n")
		fmt.Fprintf(os.Stderr, "  imgcompress -q 70 image.jpg image2.png\n")
		fmt.Fprintf(os.Stderr, "  imgcompress -s _min -v *.jpg\n")
	}

	flag.Parse()

	opts.InputPaths = flag.Args()

	return opts, nil
}

// Validate はオプションの妥当性を検証する
func (opts *Options) Validate() error {
	if opts.Version {
		return nil
	}

	if len(opts.InputPaths) == 0 {
		return fmt.Errorf("画像ファイルを指定してください")
	}

	if opts.Quality < 1 || opts.Quality > 100 {
		return fmt.Errorf("品質は1-100の範囲で指定してください: %d", opts.Quality)
	}

	if opts.Suffix == "" {
		return fmt.Errorf("サフィックスを空にすることはできません")
	}

	return nil
}
