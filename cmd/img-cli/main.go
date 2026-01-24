package main

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"

	"github.com/disintegration/imaging"
)

func main() {
	// テスト用画像ファイルのパス
	inputPath := "testdata/sample_compressed.jpg"

	// 画像を読み込む
	img, err := imaging.Open(inputPath)
	if err != nil {
		log.Fatalf("画像の読み込みに失敗しました: %v", err)
	}

	// 画像情報を表示
	bounds := img.Bounds()
	fmt.Printf("画像読み込み成功: %s\n", inputPath)
	fmt.Printf("サイズ: %dx%d\n", bounds.Dx(), bounds.Dy())
	fmt.Printf("カラーモデル: %T\n", img.ColorModel())

	// リサイズのテスト（幅を半分にする）
	resized := imaging.Resize(img, bounds.Dx()/2, 0, imaging.Lanczos)
	fmt.Printf("リサイズ後: %dx%d\n", resized.Bounds().Dx(), resized.Bounds().Dy())

	// 出力先ディレクトリを作成
	if err := os.MkdirAll("testdata/output", 0755); err != nil {
		log.Fatalf("出力ディレクトリの作成に失敗しました: %v", err)
	}

	// リサイズした画像を保存
	outputPath := "testdata/output/sample_resized.jpg"
	if err := imaging.Save(resized, outputPath); err != nil {
		log.Fatalf("画像の保存に失敗しました: %v", err)
	}

	fmt.Printf("リサイズした画像を保存しました: %s\n", outputPath)
}
