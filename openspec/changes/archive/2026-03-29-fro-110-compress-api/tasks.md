## 1. ブランチ作成

- [x] 1.1 `feature/fro-110-compress-api` ブランチを main から作成する

## 2. ハンドラー実装

- [x] 2.1 `internal/handler/compress.go` を作成し、型定義を実装する（CompressFormData, CompressOutput, CompressHandler）
- [x] 2.2 `detectFormatFromMIME` ヘルパー関数を実装する（MIMEタイプ → ImageFormat 変換）
- [x] 2.3 `parseCompressionLevel` ヘルパー関数を実装する（文字列 → CompressionLevel 変換）
- [x] 2.4 `CompressHandler.Handle` メソッドを実装する（ファイル取得 → フォーマット検出 → 圧縮 → レスポンス構築）

## 3. サーバーへの統合

- [x] 3.1 `internal/api/server.go` の `NewServer` にプロセッサ初期化と CompressHandler 生成を追加する
- [x] 3.2 `internal/api/routes.go` の `RegisterRoutes` シグネチャを変更し、圧縮エンドポイントを登録する（MaxBodyBytes: 50MB）

## 4. テスト実装

- [x] 4.1 `internal/handler/compress_test.go` を作成し、テスト用画像生成ヘルパーを実装する
- [x] 4.2 正常系テストを実装する（JPEG/PNG/WebP圧縮、quality指定、level指定）
- [x] 4.3 異常系テストを実装する（ファイル未指定、非対応フォーマット）
- [x] 4.4 レスポンスヘッダーテストを実装する（X-Original-Size, X-Compressed-Size, X-Compression-Ratio）

## 5. 検証とコミット

- [x] 5.1 `golangci-lint run ./...` が通ることを確認する
- [x] 5.2 `go test -race ./...` が全て通ることを確認する
- [x] 5.3 変更をコミットする
