## Context

Loki APIは現在 `POST /api/v1/compress` で画像圧縮を提供している。フォーマット変換機能は `pkg/processor.Processor.Convert()` として既に実装済みだが、APIエンドポイントが未整備のため外部から利用できない。既存の圧縮APIと同一のアーキテクチャパターン（Huma v2 + multipart/form-data + StreamResponse）に従い、変換APIを追加する。

依存関係の方向: `cmd/api/` → `internal/api/` → `internal/handler/` → `pkg/processor/`

## Goals / Non-Goals

**Goals:**
- JPEG/PNG/WebP間の相互フォーマット変換をAPI経由で提供する
- 既存の圧縮APIと一貫したインターフェース設計
- quality/levelパラメータによる出力品質制御

**Non-Goals:**
- 新規画像フォーマット（GIF, AVIF等）の追加
- バッチ変換・非同期処理
- 画像リサイズとの統合

## Decisions

### 1. ConvertHandlerをCompressHandlerと同じパターンで実装

**選択**: `internal/handler/convert.go` に `ConvertHandler` を新規作成し、`CompressHandler` と同じ構造（processors map依存注入、multipart入力、StreamResponse出力）を踏襲する。

**理由**: 既存パターンとの一貫性を保ち、コードの理解・保守を容易にする。共通のヘルパー関数（`detectFormatFromMIME`, `parseCompressionLevel`, `isDecodeError`）を再利用できる。

**代替案**: CompressHandlerにConvertロジックを統合する案 → ハンドラが肥大化し責務が曖昧になるため却下。

### 2. 同一フォーマット変換は圧縮にフォールバック

**選択**: 入力フォーマットと出力フォーマットが同一の場合、`proc.Compress()` を呼び出して圧縮処理を行う。

**理由**: Linear FRO-111の要件（「同一フォーマット変換の場合は圧縮にフォールバック or エラー」）に従う。エラーを返すよりユーザー体験が良い。

**代替案**: エラーを返す → ユーザーが明示的に圧縮APIへリダイレクトする必要があり不便。

### 3. 出力フォーマットのプロセッサでConvertを呼ぶ

**選択**: 出力フォーマットに対応するプロセッサの `Convert()` を呼ぶ。例: JPEG→WebP変換では `WebPProcessor.Convert()` を使用。

**理由**: 各プロセッサの `Convert()` は「指定フォーマットへの変換」として実装されており、出力先プロセッサを選択するのが自然。

## Risks / Trade-offs

- **[リスク] 大きな画像のメモリ使用量** → 既存の `MaxFileSize` (50MB) 制限で軽減。圧縮APIと同じ制限を適用。
- **[トレードオフ] ハンドラ間の共通コード** → `detectFormatFromMIME` 等は `compress.go` に定義済みで同一パッケージ内から参照可能。将来的に共通化が必要になれば別ファイルに抽出可能。
