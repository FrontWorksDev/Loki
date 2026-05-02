## Why

FRO-109〜112 で API サーバ・compress/convert エンドポイント・CORS/レート/ボディ制限ミドルウェアが揃ったが、本番運用 (FRO-114 Cloud Run デプロイ) に必要な要素が 2 つ欠けている: (1) リッスンアドレス (`host`) を設定で切り替えられないため localhost バインドや IPv6 切り替えができない、(2) Huma が自動生成する OpenAPI スペックがメタ情報・エラーレスポンス・リクエスト例ともに最低限で、API クライアント実装やドキュメントとしての自己完結性が不足している。本変更でこの 2 点を埋め、デプロイチケットの前提条件を満たす。

## What Changes

- `api.host` 設定キーを Viper 設定に追加（デフォルト `"0.0.0.0"`、環境変数 `LOKI_API_HOST` で上書き可能）。HTTP サーバの `Addr` を `net.JoinHostPort(host, port)` で構築する形に変更。
- `configs/default.yaml` に `api.host` を追加。
- OpenAPI メタ情報を充実: `Info.Description` を API 概要・対応フォーマット・認証ポリシー・レート制限要約を含む複数行に拡張し、`Info.Contact`（FrontWorksDev）と `Info.License`（MIT）を設定。
- 全画像エンドポイント (compress / convert) に共通エラーレスポンス（400 / 413 / 422 / 429 / 500）を `huma.ErrorModel` ベースで OpenAPI 上に明示。共通ヘルパー関数 `commonErrorResponses()` を新設して再利用する。
- compress / convert の主要入力フィールド（`quality` / `level` / `format`）に Huma の `example` 構造体タグを付与し、OpenAPI スキーマに値の例が出るようにする。
- README に `api.host` 設定項目と OpenAPI スペック (`/openapi.json` / `/docs`) の参照方法を追記。
- 設定読み込みテストに host 検証を追加。新規 `internal/api/openapi_test.go` で OpenAPI スペックがメタ情報・エラーレスポンス・example を含むことを検証。

## Capabilities

### New Capabilities

- `api-openapi-docs`: API の OpenAPI スペック品質 (Info メタ、共通エラーレスポンス、リクエスト/レスポンス例) を担保する横断仕様。各エンドポイントスペックが個別に書く必要のない、ドキュメント品質に関する共通要件を集約する。

### Modified Capabilities

- `api-middleware`: 既存の「設定の外部化」要件を拡張し、`api.host` 設定キーをサポート対象に追加する。サーバの `Addr` 構築方式 (`net.JoinHostPort`) も併せて要件化する。

## Impact

- **コード**:
  - `internal/api/config.go`: `Config.Host` フィールド・デフォルト値・YAML/環境変数読み込み追加
  - `internal/api/server.go`: `httpServer.Addr` 構築変更、`humaConfig.Info` 拡充
  - `internal/api/routes.go`: compress/convert の `Operation.Responses` にエラーレスポンス追加
  - `internal/api/errors.go` (新規): 共通エラーレスポンスヘルパー
  - `internal/api/openapi_test.go` (新規): OpenAPI スペック検証
  - `internal/api/config_test.go`: host 検証追加
  - `internal/handler/compress.go`, `internal/handler/convert.go`: 入力フィールドに `example` タグ追加
- **設定**: `configs/default.yaml` に `api.host` 追加。
- **依存**: 既存 `huma/v2 v2.37.2`, `viper v1.21.0` の機能のみ使用（追加依存なし）。
- **ドキュメント**: `README.md` 更新。
- **後方互換**: `Addr` のホスト指定方式変更で挙動が `:port` から `0.0.0.0:port` に変わるが、デフォルト挙動 (全インタフェースで listen) は同等。設定ファイルや環境変数の既存キーは変更なし。Breaking change なし。
- **下流チケット**: FRO-114 (Cloud Run デプロイ) の前提条件を解消する。
