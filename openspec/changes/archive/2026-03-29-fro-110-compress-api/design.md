## Context

Lokiプロジェクトは画像圧縮CLIツールとして `pkg/processor` パッケージに JPEG/PNG/WebP の圧縮機能を持つ。FRO-109で Huma v2 + Chi v5 によるAPIサーバー基盤が構築済みで、ヘルスチェックエンドポイント（`GET /api/v1/health`）が稼働している。

現在のAPIサーバー（`internal/api/server.go`）は `RegisterRoutes(api huma.API)` でルートを登録するシンプルな構造。プロセッサへの依存注入の仕組みはまだない。

## Goals / Non-Goals

**Goals:**

- `POST /api/v1/compress` エンドポイントを実装し、画像アップロード→圧縮→バイナリレスポンスの一連のフローを提供する
- 既存の `pkg/processor.Processor` インターフェースを再利用し、API層とのクリーンな接続を実現する
- 圧縮結果メタデータをレスポンスヘッダーで返す
- テスト可能な設計（ハンドラー単体テスト + 統合テスト）

**Non-Goals:**

- 非同期処理やジョブキューによるバックグラウンド圧縮
- 画像フォーマット変換（`Convert` APIは別タスク）
- 認証・認可の実装
- レート制限やキャッシュ
- 画像リサイズ機能の統合

## Decisions

### 1. ハンドラーの配置: `internal/handler/` パッケージ

ハンドラーロジックを `internal/handler/compress.go` に配置する。

**理由**: `internal/handler/` ディレクトリは既に準備されており、ルーティング（`internal/api/`）とビジネスロジック（ハンドラー）の責務を分離できる。将来の Convert API 等も同パッケージに追加できる。

**代替案**: `internal/api/routes.go` にインラインでハンドラーを書く方法もあるが、テスタビリティとコードの見通しが悪くなる。

### 2. プロセッサの注入方法: `NewServer` 内部で生成

`NewServer` 内で全プロセッサを生成し、`CompressHandler` に渡す。`RegisterRoutes` のシグネチャを `RegisterRoutes(api huma.API, compressHandler *handler.CompressHandler)` に変更する。

**理由**: プロセッサはステートレスで設定不要のため、外部からの注入は過剰。テスト時はハンドラーを直接構築できるため、テスタビリティも確保される。

**代替案**: `Config` 構造体にプロセッサマップを持たせる方法。現時点では不要な複雑さ。

### 3. レスポンス方式: `[]byte` Body + ヘッダー構造体

Huma の出力構造体に `Body []byte` と `header` タグ付きフィールドを使用する。

```go
type CompressOutput struct {
    ContentType       string `header:"Content-Type"`
    XOriginalSize     int64  `header:"X-Original-Size"`
    XCompressedSize   int64  `header:"X-Compressed-Size"`
    XCompressionRatio string `header:"X-Compression-Ratio"`
    Body              []byte
}
```

**理由**: Huma は `[]byte` Body を検出すると JSON マーシャリングをスキップし、直接バイナリを書き出す。`header` タグでカスタムヘッダーも設定でき、`StreamResponse` より宣言的でシンプル。

**代替案**: `huma.StreamResponse` を使う方法。柔軟だが、ヘッダー構造体の方がテストしやすく OpenAPI ドキュメントにも反映される。

### 4. フォーマット検出: Content-Type ベース

アップロードされたファイルの Content-Type から画像フォーマットを判定する。Huma の `FormFile` は multipart ヘッダーの Content-Type を読み、未指定の場合は `http.DetectContentType`（マジックバイト検出）にフォールバックする。

**理由**: Huma の `contentType` タグでバリデーションも兼ねられる（`image/jpeg,image/png,image/webp`）。

### 5. ファイルサイズ制限: Huma の `MaxBodyBytes`

`huma.Operation` の `MaxBodyBytes` を `50 * 1024 * 1024`（50MB）に設定する。

**理由**: Huma のデフォルトは 1MB。Operation レベルで設定することで、エンドポイントごとに制限を変えられる。

## Risks / Trade-offs

- **メモリ使用量**: 画像全体をメモリに読み込み、圧縮結果もバッファリングするため、50MB近い画像では最大100MB+/リクエストのメモリを消費する → 初期実装としては許容。将来的にストリーミング処理を検討。
- **Huma multipart の制約**: `huma.MultipartFormFiles` と非ファイルフォームフィールドの組み合わせが想定通り動作するか確認が必要 → テストで検証。
- **OpenAPI ドキュメントのバイナリレスポンス表記**: `[]byte` Body が OpenAPI spec でどう表現されるか → 自動生成を確認し、必要に応じて Operation に手動設定を追加。
