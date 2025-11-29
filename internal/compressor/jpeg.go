package compressor

import (
	"bytes"
	"image"
	"image/jpeg"

	imgpkg "github.com/FrontWorksDev/image-compressor/internal/image"
)

// JPEGCompressor はJPEG形式の圧縮を行う
type JPEGCompressor struct{}

// Compress はJPEG画像を圧縮する
func (c *JPEGCompressor) Compress(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(&buf, img, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Format は形式を返す
func (c *JPEGCompressor) Format() imgpkg.Format {
	return imgpkg.FormatJPEG
}
