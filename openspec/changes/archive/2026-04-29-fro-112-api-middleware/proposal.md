## Why

現在Loki APIサーバー（Huma v2 + Chi v5）はビジネスロジック（圧縮・変換エンドポイント）のみ実装されており、CORS・構造化ロギング・ボディサイズ制限・レートリミット・ヘルスチェックといった運用上必須の横断的関心事（cross-cutting concerns）がミドルウェア層として整備されていない。Cloud Run上で公開APIとして運用するために、これらの基盤機能をLinear FRO-112の要件として実装する必要がある。

## Non-goals

- 認証・認可（APIキー、JWT等）の実装は本変更のスコープ外（別タスク）
- 分散レートリミット（Redis等を用いたインスタンス間共有）は対象外。本変更ではインメモリのIPベースレートリミットに限定する
- メトリクス・トレーシング（OpenTelemetry等）の導入は対象外
- WAF（Web Application Firewall）相当の高度な防御機構は対象外
- 既存エンドポイント（`/api/v1/compress`、`/api/v1/convert`）の挙動変更は行わない（ミドルウェアによる横断的な制約適用のみ）

## What Changes

- `internal/api/middleware/` パッケージを新設し、以下のミドルウェアを実装する
  - **CORS**: 許可オリジン・メソッド・ヘッダーを設定ファイルから読み込んで適用
  - **構造化ロギング**: `log/slog` によるJSON形式のリクエスト/レスポンスログ（メソッド・パス・ステータス・処理時間・リモートIP）
  - **リクエストサイズ制限**: ボディサイズ上限（デフォルト50MB）を超えた場合に413を返す
  - **レートリミット**: IPベースのインメモリレートリミット（デフォルト30リクエスト/分）。超過時は429を返す
- `GET /api/v1/health` エンドポイントを `internal/api/routes.go` に追加（既に存在するが、ヘルスチェックcapabilityとして仕様化）
- `internal/api/server.go` でミドルウェアをChiルーターに登録する
- `Config` 構造体にCORS・ロギング・サイズ制限・レートリミットの設定項目を追加し、`configs/default.yaml` および環境変数からの読み込みに対応
- 各ミドルウェアにテーブル駆動テストを追加

## Capabilities

### New Capabilities

- `api-middleware`: CORS、構造化ロギング、リクエストサイズ制限、IPベースレートリミット、ヘルスチェックを含むAPIサーバーの横断的ミドルウェア層の仕様

### Modified Capabilities

なし（既存capabilityの要件変更はなし。既存エンドポイントはミドルウェアの影響下に入るが、正常系の振る舞いは変わらない）

## Impact

- **コード**:
  - `internal/api/middleware/cors.go`、`logging.go`、`bodylimit.go`、`ratelimit.go`（新規）
  - `internal/api/middleware/*_test.go`（新規）
  - `internal/api/server.go`（ミドルウェア登録、Config拡張）
  - `internal/api/config.go`（新規 / もしくはserver.go内Configを拡張）
  - `cmd/api/main.go`（設定読み込みの追加）
  - `configs/default.yaml`（API関連設定セクションの追加）
- **API**:
  - 全エンドポイントにCORSヘッダーが付与される
  - 50MB超のリクエストは413
  - レート超過は429
  - `GET /api/v1/health` が200を返す（既存）
- **依存関係**: 新規導入候補
  - `github.com/go-chi/cors` または `github.com/rs/cors`（CORS）
  - `golang.org/x/time/rate`（トークンバケットによるレートリミット）
  - 既存の `spf13/viper` を活用して設定読み込み
- **テスト**: 各ミドルウェアの単体テストおよびサーバー統合テスト（既存の `internal/api/server_test.go` への追加）
- **運用**: Cloud RunでJSON構造化ログがそのまま Cloud Logging に取り込めるようになる
