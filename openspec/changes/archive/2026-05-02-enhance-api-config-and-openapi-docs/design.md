## Context

API サーバ実装は `internal/api/` 配下に集約されており、設定 (`config.go`)・サーバ初期化と OpenAPI 構成 (`server.go`)・ルート登録 (`routes.go`) の責務が明確に分離されている。Huma v2 (`v2.37.2`) は struct タグ駆動で OpenAPI スペックを自動生成し、`huma.Operation.Responses` への明示記述で個別レスポンスを上書きできる。設定は Viper (`v1.21.0`) で `api.*` 名前空間に統一され、環境変数 `LOKI_API_*` のオートバインドが既に動作している。本変更はこの既存構造を崩さず、欠けているスロットを埋める。

## Goals / Non-Goals

**Goals:**

- リッスンアドレス (`api.host`) を設定経由で切り替え可能にし、Cloud Run / ローカル両方の運用要件を満たす。
- OpenAPI スペックを「読むだけで API クライアントが書ける」自己文書性まで引き上げる (Info メタ・共通エラーレスポンス・主要フィールドの example)。
- 既存の `api.*` 設定キー命名と `LOKI_API_*` 環境変数規約を維持し、既存ユーザに破壊的変更を与えない。

**Non-Goals:**

- API バージョニング戦略 (現状 v1 のみで十分、将来チケット)。
- 認証・認可の追加 (要件外、本 API は認証なしを継続)。
- レート制限の永続化や分散対応 (FRO-112 で確立した InMemory 実装を維持)。
- multipart/form-data の完全な OpenAPI Examples 記述 (struct タグの `example` 付与で十分とユーザ確認済み)。
- Cloud Run 固有の設定 (Dockerfile, デプロイマニフェスト) — FRO-114 のスコープ。

## Decisions

### D1. 設定キー命名は既存 `api.*` を維持し `api.host` を追加

**選択**: 既存の `api.port` / `api.body_limit_bytes` / `api.cors.*` と同じ階層に `api.host` を追加。

**代替案**:
- (A) FRO-113 チケット記載通り `server.*` / `cors.*` のフラット構成へ移行。
- (B) 新旧両キーをサポートする互換レイヤ。

**理由**: FRO-110/111/112 で確立した命名規約と整合。マイグレーションコスト・テスト改修コスト・ドキュメント混乱を避ける。ユーザ確認済 (proposal 作成前)。

### D2. `httpServer.Addr` 構築に `net.JoinHostPort` を使用

**選択**: `fmt.Sprintf(":%d", cfg.Port)` を `net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))` に置換。

**理由**: IPv6 アドレス (`::1` 等) を host に指定した際にもブラケットを正しく付与する標準ライブラリのイディオム。`fmt.Sprintf("%s:%d", ...)` では IPv6 で誤動作する。

### D3. OpenAPI 共通エラーレスポンスは `Operation.Errors []int` を採用

**選択**: 新規ファイル `internal/api/errors.go` に `commonErrorCodes() []int` ヘルパー (400 / 413 / 422 / 429 / 500 のステータスコード列を返す) を定義し、各 `Operation.Errors` に渡す。Huma v2 が `defineErrors`（`huma.go:1624`）で自動的に `application/problem+json` + `huma.ErrorModel` のレスポンスを `Operation.Responses` に追加する。

**代替案**:
- (A) `commonErrorResponses() map[string]*huma.Response` ヘルパーを手書きし `Operation.Responses` にマージ（当初案）。
- (B) 独自エラー型を定義する。
- (C) Huma の `huma.NewError` フックを上書きしてレスポンス形式を統一。

**理由**: 実装中に Huma v2 の `Operation.Errors []int` フィールドが、RFC9457 準拠の `application/problem+json` + `huma.ErrorModel` レスポンスを自動生成すると判明。コード量・保守性・ランタイム動作との一貫性で当初案 (A) を上回る。当初案で生成されるはずの OpenAPI 出力と完全に等価で、要件 (compress/convert に 400/413/422/429/500 が `application/problem+json` で明示される) を満たす。`huma.Error400BadRequest` 等の既存ハンドラ実装と完全に整合。

### D4. ヘルスチェックには共通エラーレスポンスを適用しない

**選択**: `RegisterHealth` の Operation には共通エラー定義を追加しない。

**理由**: 既存仕様 (`api-middleware`「ヘルスチェックエンドポイント」) でレートリミット・サイズ制限・認証の影響を受けないと規定済み。実態として 200 以外を返さないため OpenAPI 上の追加定義は不要。

### D5. リクエスト例は struct タグの `example` で付与

**選択**: `internal/handler/compress.go` / `convert.go` の `CompressFormData` / `ConvertFormData` の `Quality` / `Level` / `Format` に `example:"…"` を追加。multipart の完全リクエスト例 (`Operation.Examples`) は記述しない。

**理由**: Huma が struct タグから OpenAPI schema の `example` フィールドを自動生成する。保守性が高くテスト容易。multipart の完全 Example は記述が冗長で、Swagger UI での体験向上効果が限定的とユーザ確認済。

### D6. OpenAPI Info メタは `humaConfig.Info` 直接拡張で完結

**選択**: `server.go` 内の `humaConfig := huma.DefaultConfig(...)` 直後で `Info.Description` / `Info.Contact` / `Info.License` を設定。別関数化はしない (1 箇所での記述で十分シンプル)。

### D7. OpenAPI スペック検証テストは `huma.API.OpenAPI()` を直接利用

**選択**: 新規 `internal/api/openapi_test.go` で `humatest.New(t)` パターンを流用し、`api.OpenAPI()` から `*huma.OpenAPI` を取得して構造体フィールドをアサーション。`/openapi.json` の HTTP 取得は行わない。

**理由**: 既存の compress/convert ハンドラテスト (`internal/handler/compress_test.go`) と同じ humatest パターンで一貫性を保つ。HTTP 経由は Marshal/Unmarshal を経るため壊れやすい。

## Risks / Trade-offs

- **R1. デフォルト host 変更による挙動差**:
  - 旧: `Addr: ":8080"` (Go 標準で全インタフェース listen)
  - 新: `Addr: "0.0.0.0:8080"` (実質同等だが、IPv6 接続が来ない環境差は理論上存在)
  - **Mitigation**: デフォルト値を `0.0.0.0` で固定、IPv6 を含むデュアルスタックが必要な場合は `LOKI_API_HOST=::` で対応可能と README に明記。
- **R2. 共通エラーレスポンス追加による OpenAPI スキーマ膨張**:
  - 各エンドポイントに 5 つのエラーレスポンスが追加され、生成 spec のサイズが増える。
  - **Mitigation**: 全て `huma.ErrorModel` を `$ref` で参照する形式 (Huma が自動的に components/schemas に移動) なので、増加は許容範囲内。
- **R3. example タグ追加による既存テスト破壊**:
  - 既存の compress/convert ハンドラテストが struct タグに依存していると壊れる可能性。
  - **Mitigation**: `example` タグは挙動に影響しない (バリデーションも JSON 表現も変えない)。実装後に `go test -race ./...` で確認。
- **R4. Cloud Run 環境固有のホスト要件**:
  - Cloud Run は `0.0.0.0` でなく `::` (IPv6) を要求するケースがある。
  - **Mitigation**: 設定経由で切り替え可能なので、デプロイ時の環境変数で対応。FRO-114 で具体検証。
