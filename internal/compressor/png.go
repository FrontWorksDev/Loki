package compressor

import (
	"bytes"
	"image"
	"image/png"

	imgpkg "github.com/FrontWorksDev/image-compressor/internal/image"
)

// PNGCompressor はPNG形式の圧縮を行う
type PNGCompressor struct{}

// Compress はPNG画像を圧縮する
func (c *PNGCompressor) Compress(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: imgpkg.QualityToPNGCompression(quality)}
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Format は形式を返す
func (c *PNGCompressor) Format() imgpkg.Format {
	return imgpkg.FormatPNG
}
