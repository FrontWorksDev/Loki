//go:build ignore

package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	// このファイルのディレクトリを取得
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// 100x100のテスト画像を生成
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// グラデーションパターンで塗りつぶし
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 2),
				G: uint8(y * 2),
				B: uint8((x + y)),
				A: 255,
			})
		}
	}

	// JPEG保存
	jpegFile, _ := os.Create(filepath.Join(dir, "sample.jpg"))
	defer jpegFile.Close()
	jpeg.Encode(jpegFile, img, &jpeg.Options{Quality: 100})

	// PNG保存
	pngFile, _ := os.Create(filepath.Join(dir, "sample.png"))
	defer pngFile.Close()
	png.Encode(pngFile, img)
}
