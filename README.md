# Loki

[![CI](https://github.com/FrontWorksDev/Loki/actions/workflows/test.yml/badge.svg)](https://github.com/FrontWorksDev/Loki/actions/workflows/test.yml)
[![Build](https://github.com/FrontWorksDev/Loki/actions/workflows/build.yml/badge.svg)](https://github.com/FrontWorksDev/Loki/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/FrontWorksDev/Loki/graph/badge.svg?token=RDMgY2TG7P)](https://codecov.io/gh/FrontWorksDev/Loki)
![Go Version](https://img.shields.io/badge/Go-1.25.6-00ADD8?logo=go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 概要

Loki は Go 製の画像圧縮 CLI ツールです。JPEG / PNG 形式に対応し、品質やレベルを指定して効率的に画像を圧縮できます。

## 機能

- **JPEG / PNG 圧縮** - 品質 (1-100) や圧縮レベル (low / medium / high) を指定可能
- **ディレクトリ再帰処理** - ディレクトリ内の画像を一括で処理
- **バッチ並列処理** - CPU コア数に応じた並列圧縮で高速処理
- **TUI プログレスバー** - Bubble Tea ベースのリアルタイム進捗表示
- **YAML 設定ファイル** - Viper による設定ファイル対応（CLI フラグで上書き可能）
- **クロスプラットフォーム** - Linux / macOS / Windows 対応

## インストール

### go install

```bash
go install github.com/FrontWorksDev/Loki/cmd/img-cli@latest
```

### ソースからビルド

```bash
git clone https://github.com/FrontWorksDev/Loki.git
cd Loki
go build -o build/img-cli ./cmd/img-cli
```

## 使い方

### 基本的なコマンド

```bash
# 単一ファイルの圧縮
img-cli compress photo.jpg

# 品質を指定して圧縮
img-cli compress photo.jpg -q 70

# 圧縮レベルと出力先を指定
img-cli compress photo.jpg -l high -o output.jpg

# ディレクトリ内の画像を再帰的に圧縮
img-cli compress images/ -r -o images_compressed/

# TUI プログレスバー付きでディレクトリ圧縮
img-cli compress images/ -r --tui
```

### フラグ一覧

| フラグ | 短縮 | 型 | デフォルト | 説明 |
|--------|------|------|-----------|------|
| `--quality` | `-q` | int | `0` | JPEG 品質 (1-100)。0 の場合は level に基づく |
| `--level` | `-l` | string | `medium` | 圧縮レベル (`low` / `medium` / `high`) |
| `--output` | `-o` | string | (自動生成) | 出力パス。省略時は `{name}_compressed.{ext}` |
| `--recursive` | `-r` | bool | `false` | ディレクトリを再帰的に処理 |
| `--tui` | - | bool | `false` | TUI プログレスバーを表示 |
| `--config` | - | string | `~/.image-compresser.yaml` | 設定ファイルのパス |

### 圧縮レベル

| レベル | JPEG 品質 | PNG 圧縮 | 用途 |
|--------|----------|----------|------|
| `low` | 60 | BestSpeed | 高速処理優先 |
| `medium` | 75 | DefaultCompression | バランス型（デフォルト） |
| `high` | 90 | BestCompression | 最大圧縮 |

### 設定ファイル

YAML 形式の設定ファイルでデフォルト値を指定できます。

```yaml
# ~/.image-compresser.yaml
compress:
  quality: 0          # JPEG品質 (1-100)。0の場合はlevelに基づく
  level: "medium"     # 圧縮レベル (low/medium/high)
  output: ""          # 出力パス (空の場合は自動生成)
  recursive: false    # ディレクトリを再帰的に処理する
```

**設定の優先順位:** CLI フラグ > 設定ファイル > ビルトインデフォルト

## 開発者向け

### 前提条件

- [asdf](https://asdf-vm.com/) (バージョンマネージャー)
- Go 1.25.6 (`asdf install` で自動インストール)

### ビルド・テスト・リント

```bash
# ビルド
go build ./...
go build -o build/img-cli ./cmd/img-cli

# テスト
go test ./...
go test -v ./...
go test -cover ./...

# カバレッジ
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -html=coverage.out

# リント・フォーマット
golangci-lint run ./...
go fmt ./...
goimports -w ./...
```

### プロジェクト構造

```
Loki/
├── cmd/
│   ├── img-cli/           # CLI アプリケーションのエントリーポイント
│   └── tui-demo/          # TUI デモアプリケーション
├── internal/
│   ├── cli/               # CLI コマンド定義・設定管理・TUI 統合
│   └── imageproc/         # 画像処理ユーティリティ（リサイズ等）
├── pkg/
│   └── processor/         # 画像圧縮コアライブラリ（JPEG/PNG/バッチ処理）
├── configs/               # デフォルト設定ファイル
└── .github/workflows/     # CI/CD（テスト・ビルド）
```

## ライセンス

[MIT License](LICENSE)
