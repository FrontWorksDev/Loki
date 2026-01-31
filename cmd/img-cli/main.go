package main

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"

	"github.com/FrontWorksDev/Loki/internal/imageproc"
	"github.com/disintegration/imaging"
)

func main() {
	// テスト用画像ファイルのデフォルトパス
	inputPath := "testdata/sample_compressed.jpg"

	// コマンドライン引数で入力ファイルパスが指定されている場合はそれを使用
	if len(os.Args) > 1 {
		inputPath = os.Args[1]
	} else {
		fmt.Println("使用方法: img-cli <input-file-path>")
		fmt.Printf("入力ファイルが指定されていないため、デフォルトのテスト画像を使用します: %s\n", inputPath)
	}

	// 入力ファイルの存在チェック
	if _, err := os.Stat(inputPath); err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("入力ファイルが存在しません: %s\n使用方法: img-cli <input-file-path>", inputPath)
		}
		log.Fatalf("入力ファイルの確認中にエラーが発生しました: %v", err)
	}

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

	// リサイズのテスト（幅を半分にする）- internal/imageprocパッケージを使用
	resized := imageproc.ResizeImageByWidth(img, bounds.Dx()/2)
	if resized == nil {
		log.Fatalf("画像のリサイズに失敗しました")
	}
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
