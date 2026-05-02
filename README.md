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

## APIサーバー設定

`cmd/api` を起動すると Huma v2 + Chi v5 ベースのAPIサーバーが立ち上がります。Cloud Run 等の公開環境を想定し、CORS、構造化ロギング、ボディサイズ制限、IPベースレートリミット、ヘルスチェックを標準で備えています。

### 設定値（`configs/default.yaml`）

```yaml
api:
  host: "0.0.0.0"             # リッスンアドレス。"127.0.0.1" でローカル限定、"::" でIPv6
  port: 8080
  cors:
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "OPTIONS"]
    allowed_headers: ["Content-Type", "Authorization"]
    allow_credentials: false
    max_age: 300
  body_limit_bytes: 33554432   # 32 MiB (Cloud Run HTTP/1 上限と整合)
  rate_limit:
    requests_per_minute: 30
    burst: 10
  logging:
    level: "info"              # debug/info/warn/error
```

| キー | 既定値 | 環境変数 | 用途 |
|---|---|---|---|
| `api.host` | `0.0.0.0` | `LOKI_API_HOST` | リッスンアドレス。`127.0.0.1` でループバック限定、`::` で IPv6 デュアルスタック |
| `api.port` | `8080` | `LOKI_API_PORT` | リッスンポート |
| `api.body_limit_bytes` | `33554432` (32 MiB) | `LOKI_API_BODY_LIMIT_BYTES` | リクエストボディ上限。超過時は 413。既定値は Cloud Run HTTP/1 上限 (32 MiB) と整合 |
| `api.rate_limit.requests_per_minute` | `30` | `LOKI_API_RATE_LIMIT_REQUESTS_PER_MINUTE` | クライアント IP あたりの 1 分間許容リクエスト数 |
| `api.rate_limit.burst` | `10` | `LOKI_API_RATE_LIMIT_BURST` | バーストキャパシティ |
| `api.cors.allowed_origins` | `["*"]` | `LOKI_API_CORS_ALLOWED_ORIGINS` | CORS 許可オリジン |
| `api.logging.level` | `info` | `LOKI_API_LOGGING_LEVEL` | ログレベル (debug/info/warn/error) |

### 環境変数による上書き

`LOKI_API_*` プレフィックスでネスト項目をアンダースコア区切りで指定できます。

```bash
LOKI_API_HOST=127.0.0.1 \
LOKI_API_PORT=8000 \
LOKI_API_BODY_LIMIT_BYTES=10485760 \
LOKI_API_RATE_LIMIT_REQUESTS_PER_MINUTE=60 \
go run ./cmd/api
```

### OpenAPI スペック・ドキュメント

サーバ起動後、Huma が以下のパスで OpenAPI 関連リソースを自動公開します。

| パス | 内容 |
|---|---|
| `/openapi.json` | OpenAPI 3.1 仕様（JSON） |
| `/openapi.yaml` | OpenAPI 3.1 仕様（YAML） |
| `/openapi-3.0.json` | OpenAPI 3.0 ダウングレード版（JSON） |
| `/docs` | Stoplight Elements ベースのインタラクティブ API ドキュメント |
| `/schemas/*` | コンポーネントスキーマの個別取得 |

`/docs` を開けば、各エンドポイントのリクエスト例 (`example`) や RFC 9457 (`application/problem+json`) 形式の共通エラーレスポンス（400/413/422/429/500）が確認できます。

### ミドルウェアの挙動

| 機能 | 内容 |
|---|---|
| CORS | `allowed_origins` に設定したオリジンへ `Access-Control-Allow-*` を付与。プリフライトは2xxを返却 |
| ロギング | `log/slog` JSONハンドラで標準出力。Cloud Logging 互換のフィールド構造 |
| ボディサイズ制限 | 上限超過時は 413 + `application/problem+json` |
| レートリミット | クライアントIP単位（`X-Forwarded-For` 最左 → `RemoteAddr`）で 1分あたり N requests / バースト B。超過時は 429 + `Retry-After` |
| ヘルスチェック | `GET /api/v1/health` はレートリミット・ボディサイズ制限の対象外 |

本番運用では `allowed_origins` を具体的なドメインに変更してください。インメモリのレートリミットはマルチインスタンス展開では各インスタンスが独立に判定するため、厳密な共有が必要な場合は分散実装に差し替えてください。

## 開発者向け

### 前提条件

- [asdf](https://asdf-vm.com/) (バージョンマネージャー)
- Go 1.25.6 (`asdf install` で自動インストール)
- lefthook 2.1.1 (`asdf install` で自動インストール)

### セットアップ

```bash
# asdf で Go と lefthook をインストール
asdf plugin add golang    # 初回のみ
asdf plugin add lefthook  # 初回のみ
asdf install

# lefthook の Git フックを有効化
lefthook install
```

lefthook により、コミット時に自動で `goimports` と `golangci-lint` が実行されます。

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

### API サーバの起動

```bash
# 開発時の主な起動方法 (IDE デバッガ統合容易)
go run ./cmd/api

# 本番イメージとほぼ同等の環境で起動 (Cloud Run と同じ Dockerfile を使用)
docker compose up -d
curl http://localhost:8080/api/v1/health
docker compose down
```

`docker-compose.yml` は本番デプロイ前の動作確認専用です。ホットリロードは入れていないため、コード編集時は `go run ./cmd/api` を使ってください。

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

## デプロイ

API サーバは Google Cloud Run へのデプロイを前提に整備しています。`Dockerfile` (マルチステージ・distroless ベース・CGO 対応) と `.github/workflows/deploy.yml` (Workload Identity Federation 認証 → 自動デプロイ) を用意。

詳細手順は以下を参照してください。

- [`docs/deployment/gcp-setup.md`](docs/deployment/gcp-setup.md): GCP プロジェクト初回セットアップ (Artifact Registry / Service Account / Workload Identity Federation / GitHub Secrets)。コピペ実行可能な `gcloud` コマンド集。
- [`docs/deployment/cloud-run.md`](docs/deployment/cloud-run.md): Cloud Run サービス構成、初回手動デプロイ、ロールバック手順、ログ閲覧、コスト保護の運用ドキュメント。

## ライセンス

[MIT License](LICENSE)
