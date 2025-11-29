package image

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Format は画像形式を表す
type Format string

const (
	FormatJPEG    Format = "jpeg"
	FormatPNG     Format = "png"
	FormatUnknown Format = "unknown"
)

// Processor は画像の読み込み・保存・形式検出を行う
type Processor struct{}

// NewProcessor は新しいProcessorを作成する
func NewProcessor() *Processor {
	return &Processor{}
}

// Load は指定されたパスから画像を読み込む
func (p *Processor) Load(path string) (image.Image, Format, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, FormatUnknown, fmt.Errorf("ファイルを開けません: %w", err)
	}
	defer file.Close()

	format, err := p.detectFormat(file)
	if err != nil {
		return nil, FormatUnknown, err
	}

	// ファイルポインタを先頭に戻す
	if _, err := file.Seek(0, 0); err != nil {
		return nil, FormatUnknown, fmt.Errorf("ファイルの読み込みに失敗しました: %w", err)
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, FormatUnknown, fmt.Errorf("画像のデコードに失敗しました: %w", err)
	}

	return img, format, nil
}

// Save は画像を指定されたパスに保存する
func (p *Processor) Save(img image.Image, path string, format Format, quality int) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("ファイルを作成できません: %w", err)
	}
	defer file.Close()

	return p.Encode(file, img, format, quality)
}

// Encode は画像をWriterにエンコードする
func (p *Processor) Encode(w io.Writer, img image.Image, format Format, quality int) error {
	switch format {
	case FormatJPEG:
		return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
	case FormatPNG:
		encoder := png.Encoder{CompressionLevel: p.qualityToPNGCompression(quality)}
		return encoder.Encode(w, img)
	default:
		return fmt.Errorf("サポートされていない形式です: %s", format)
	}
}

// DetectFormatByPath はファイルパスから画像形式を推測する
func (p *Processor) DetectFormatByPath(path string) Format {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return FormatJPEG
	case ".png":
		return FormatPNG
	default:
		return FormatUnknown
	}
}

// detectFormat はファイルのマジックバイトから形式を検出する
func (p *Processor) detectFormat(r io.Reader) (Format, error) {
	header := make([]byte, 8)
	n, err := r.Read(header)
	if err != nil {
		return FormatUnknown, fmt.Errorf("ファイルの読み込みに失敗しました: %w", err)
	}
	if n < 2 {
		return FormatUnknown, fmt.Errorf("ファイルが小さすぎます")
	}

	// JPEG: FF D8
	if header[0] == 0xFF && header[1] == 0xD8 {
		return FormatJPEG, nil
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if n >= 8 && header[0] == 0x89 && header[1] == 0x50 && header[2] == 0x4E && header[3] == 0x47 {
		return FormatPNG, nil
	}

	return FormatUnknown, fmt.Errorf("サポートされていない画像形式です")
}

// qualityToPNGCompression は品質値をPNG圧縮レベルに変換する
func (p *Processor) qualityToPNGCompression(quality int) png.CompressionLevel {
	// 品質が低いほど圧縮率を高くする
	if quality >= 90 {
		return png.NoCompression
	} else if quality >= 70 {
		return png.DefaultCompression
	} else if quality >= 50 {
		return png.BestSpeed
	}
	return png.BestCompression
}

// GenerateOutputPath は出力ファイルパスを生成する
func (p *Processor) GenerateOutputPath(inputPath, suffix string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	return filepath.Join(dir, base+suffix+ext)
}

// GetFileSize はファイルサイズを取得する
func (p *Processor) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
