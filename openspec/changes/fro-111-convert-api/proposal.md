## Why

現在Loki APIは画像圧縮（POST /api/v1/compress）のみ対応しているが、JPEG→WebPなどのフォーマット変換機能がない。ユーザーが画像フォーマットを変更するには別ツールが必要であり、圧縮と変換を一つのAPIで完結させることで利便性が向上する。Linear FRO-111として要求されている。

## Non-goals

- 対応フォーマットの追加（GIF、AVIF等）は本変更のスコープ外
- バッチ変換（複数ファイル同時変換）は対象外
- 画像のリサイズ・トリミングとの統合は対象外

## What Changes

- `POST /api/v1/convert` エンドポイントを新規追加
  - 画像ファイルと出力フォーマットを受け取り、変換後の画像をバイナリで返す
  - 対応フォーマット: JPEG / PNG / WebP（入力・出力とも）
  - quality（1-100）および level（low/medium/high）による品質制御
  - 同一フォーマット指定時は圧縮にフォールバック
- レスポンスヘッダーに変換メタデータ（元サイズ、変換後サイズ、元フォーマット、出力フォーマット）を付与
- Swagger/OpenAPIドキュメントへの反映

## Capabilities

### New Capabilities
- `image-convert-api`: 画像フォーマット変換APIエンドポイント（POST /api/v1/convert）の仕様

### Modified Capabilities

なし

## Impact

- **コード**: `internal/handler/convert.go`（新規）、`internal/api/routes.go`・`server.go`（修正）
- **API**: 新規エンドポイント追加（既存エンドポイントへの影響なし）
- **依存関係**: 追加なし（既存の `pkg/processor.Processor.Convert()` を活用）
- **テスト**: `internal/handler/convert_test.go`（新規）
