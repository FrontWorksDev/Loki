# imgcompress

JPEG/PNG画像を圧縮するシンプルなCLIツール

## インストール

```bash
go install github.com/FrontWorksDev/image-compressor/cmd/imgcompress@latest
```

または、リポジトリをクローンしてビルド:

```bash
git clone https://github.com/FrontWorksDev/image-compressor.git
cd image-compressor
make build
```

## 使い方

```bash
# 単一ファイルを圧縮
imgcompress image.jpg

# 複数ファイルを圧縮
imgcompress image1.jpg image2.png image3.jpg

# 品質を指定（1-100、デフォルト: 80）
imgcompress -q 70 image.jpg

# サフィックスを変更（デフォルト: _compressed）
imgcompress -s _min image.jpg

# 詳細出力
imgcompress -v image.jpg

# バージョン表示
imgcompress --version
```

## オプション

| オプション | 短縮形 | 説明 | デフォルト |
|-----------|-------|------|-----------|
| `--quality` | `-q` | 圧縮品質 (1-100) | 80 |
| `--suffix` | `-s` | 出力ファイルのサフィックス | `_compressed` |
| `--verbose` | `-v` | 詳細出力 | false |
| `--version` | | バージョン表示 | |

## 出力例

```
[完了] photo.jpg -> photo_compressed.jpg (1.2MB -> 450KB, 62%削減)
```

## 対応形式

- JPEG (.jpg, .jpeg)
- PNG (.png)

## 開発

```bash
# ビルド
make build

# テスト実行
make test

# テストカバレッジ
make test-coverage

# クリーン
make clean
```

## ライセンス

MIT License
