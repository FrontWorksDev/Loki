package imageproc

import (
	"image"

	"github.com/disintegration/imaging"
)

// ResizeImage は画像を指定されたサイズにリサイズします
func ResizeImage(img image.Image, width, height int) image.Image {
	return imaging.Resize(img, width, height, imaging.Lanczos)
}

// ResizeImageByWidth は画像を指定された幅にリサイズします（アスペクト比を維持）
func ResizeImageByWidth(img image.Image, width int) image.Image {
	return imaging.Resize(img, width, 0, imaging.Lanczos)
}

// ResizeImageByHeight は画像を指定された高さにリサイズします（アスペクト比を維持）
func ResizeImageByHeight(img image.Image, height int) image.Image {
	return imaging.Resize(img, 0, height, imaging.Lanczos)
}
