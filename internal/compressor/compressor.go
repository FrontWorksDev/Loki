package compressor

import (
	"bytes"
	"image"
	"io"

	imgpkg "github.com/FrontWorksDev/image-compressor/internal/image"
)

// Compressor は画像圧縮のインターフェース
type Compressor interface {
	// Compress は画像を圧縮してバイト列を返す
	Compress(img image.Image, quality int) ([]byte, error)
	// Format は対応する形式を返す
	Format() imgpkg.Format
}

// GetCompressor は形式に対応するCompressorを返す
func GetCompressor(format imgpkg.Format) Compressor {
	switch format {
	case imgpkg.FormatJPEG:
		return &JPEGCompressor{}
	case imgpkg.FormatPNG:
		return &PNGCompressor{}
	default:
		return nil
	}
}

// compressWithEncoder は共通のエンコード処理
func compressWithEncoder(img image.Image, encoder func(w io.Writer, img image.Image) error) ([]byte, error) {
	var buf bytes.Buffer
	if err := encoder(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
