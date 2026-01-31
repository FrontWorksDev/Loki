package imageproc

import (
	"image"

	"github.com/disintegration/imaging"
)

// ResizeImage は画像を指定されたサイズにリサイズします
// img が nil の場合、または width/height が 0 以下の場合は nil を返します
func ResizeImage(img image.Image, width, height int) image.Image {
	if img == nil {
		return nil
	}
	if width <= 0 || height <= 0 {
		return nil
	}
	return imaging.Resize(img, width, height, imaging.Lanczos)
}

// ResizeImageByWidth は画像を指定された幅にリサイズします（アスペクト比を維持）
// img が nil の場合、または width が 0 以下の場合は nil を返します
func ResizeImageByWidth(img image.Image, width int) image.Image {
	if img == nil {
		return nil
	}
	if width <= 0 {
		return nil
	}
	return imaging.Resize(img, width, 0, imaging.Lanczos)
}

// ResizeImageByHeight は画像を指定された高さにリサイズします（アスペクト比を維持）
// img が nil の場合、または height が 0 以下の場合は nil を返します
func ResizeImageByHeight(img image.Image, height int) image.Image {
	if img == nil {
		return nil
	}
	if height <= 0 {
		return nil
	}
	return imaging.Resize(img, 0, height, imaging.Lanczos)
}
