## Why

Lokiは画像圧縮CLIツールとして機能しているが、Webサービスやフロントエンドアプリケーションから利用するためのAPIエンドポイントが存在しない。FRO-109でHumaフレームワークの基盤が構築済みであり、次のステップとして画像圧縮APIを提供することで、プログラムからの画像圧縮を可能にする。

## What Changes

- 画像圧縮APIエンドポイント `POST /api/v1/compress` を新規追加
- multipart/form-dataで画像ファイル（JPEG/PNG/WebP）を受け取り、圧縮して返す
- 圧縮品質（quality: 1-100）と圧縮レベル（level: low/medium/high）をオプションで指定可能
- レスポンスヘッダーに圧縮結果メタデータ（元サイズ、圧縮後サイズ、圧縮率）を付与
- ファイルサイズ上限（50MB）のバリデーション
- 非対応フォーマットに対する適切なエラーレスポンス
- 既存の `pkg/processor.Processor` インターフェースをAPI層から活用するためのDI構造を導入

## Capabilities

### New Capabilities

- `image-compress-api`: 画像ファイルを受け取り圧縮して返すHTTP APIエンドポイント。リクエストバリデーション、フォーマット検出、圧縮実行、レスポンスヘッダー付きバイナリレスポンスを含む。

### Modified Capabilities

（既存specへの要件変更なし）

## Impact

- **新規ファイル**: `internal/handler/compress.go`, `internal/handler/compress_test.go`
- **修正ファイル**: `internal/api/server.go`（プロセッサDI追加）, `internal/api/routes.go`（エンドポイント登録）
- **依存関係**: 既存の `pkg/processor` パッケージを使用（変更なし）
- **API**: 新規エンドポイント `POST /api/v1/compress` が追加される（OpenAPIドキュメントに自動反映）
- **ブロック**: FRO-124（APIクライアント実装）とFRO-113（API設定管理）がこのタスクに依存
