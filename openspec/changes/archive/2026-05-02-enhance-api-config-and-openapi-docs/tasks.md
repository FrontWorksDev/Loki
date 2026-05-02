## 1. ブランチ作成と前提確認

- [x] 1.1 `feature/fro-113-api-config-and-openapi-docs` ブランチを `main` から作成
- [x] 1.2 `lefthook install` 済みであることを確認 (`ls .git/hooks/pre-commit` 等)

## 2. 設定管理: `api.host` 追加

- [x] 2.1 `internal/api/config.go` に `defaultHost = "0.0.0.0"` 定数と `Config.Host string` フィールドを追加
- [x] 2.2 `internal/api/config.go` の `DefaultConfig()` / `setDefaults()` / `LoadConfig()` で `api.host` を扱うよう更新
- [x] 2.3 `configs/default.yaml` の `api:` セクションに `host: "0.0.0.0"` を追加し、コメントに `LOKI_API_HOST` 例を併記
- [x] 2.4 `internal/api/server.go` の `httpServer.Addr` を `net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))` 形式に変更（`net` / `strconv` パッケージを import）
- [x] 2.5 `Server.Start()` のログ出力に host も含める

## 3. OpenAPI Info メタの拡充

- [x] 3.1 `internal/api/server.go` の `humaConfig.Info.Description` を API 概要・対応フォーマット・認証なし・レート制限要約を含む複数行テキストに変更
- [x] 3.2 `humaConfig.Info.Contact` (`Name`, `URL`) を設定
- [x] 3.3 `humaConfig.Info.License` (`Name`, `URL`) を設定 (リポジトリ実態に合わせて MIT 等)

## 4. 共通エラーレスポンスの導入

- [x] 4.1 `internal/api/errors.go` を新規作成し、`commonErrorCodes() []int` ヘルパーを実装（実装中に Huma の `Operation.Errors []int` 機構を採用に変更。design.md D3 を改訂）
- [x] 4.2 `internal/api/routes.go` の compress / convert の `Operation.Errors` に `commonErrorCodes()` を設定
- [x] 4.3 `RegisterHealth` には共通エラー定義を追加しない（既存仕様維持）ことをコードコメントで明示

## 5. リクエスト例 (example) の付与

- [x] 5.1 `internal/handler/compress.go` の `CompressFormData.Quality` に `example:"75"`、`Level` に `example:"medium"` を追加
- [x] 5.2 `internal/handler/convert.go` の `ConvertFormData.Format` に `example:"webp"`、`Quality` に `example:"75"`、`Level` に `example:"medium"` を追加

## 6. テスト追加・修正

- [x] 6.1 `internal/api/config_test.go` の `TestDefaultConfig_AllFields` に `Host == "0.0.0.0"` 検証を追加
- [x] 6.2 `internal/api/config_test.go` の `TestLoadConfig_FromYAML` に `api.host: "127.0.0.1"` 読み込み検証を追加
- [x] 6.3 `internal/api/config_test.go` の `TestLoadConfig_EnvOverridesYAML` に `LOKI_TEST_OVR_API_HOST` 上書き検証を追加
- [x] 6.4 `internal/api/openapi_test.go` を新規作成し、`NewServer(DefaultConfig()).API().OpenAPI()` 経由で以下を検証:
  - Info.Title / Version / Description (キーワード含有) / Contact / License
  - compress / convert の Responses に `400` / `413` / `422` / `429` / `500` がすべて存在
  - 各エラーレスポンスが `application/problem+json` を持つ
  - 健康チェックには共通エラーが含まれない
  - compress / convert の input フィールドに example が設定されている

## 7. ドキュメント更新

- [x] 7.1 `README.md` の API 設定セクションに `api.host` 設定項目と環境変数 `LOKI_API_HOST` を追加
- [x] 7.2 `README.md` に OpenAPI スペックの参照方法 (`/openapi.json` / `/docs` 等、Huma の実際のパスを起動時に確認して記載) を追記

## 8. 検証 (verification)

- [x] 8.1 `goimports -w .` を実行して整形
- [x] 8.2 `golangci-lint run ./...` を実行してエラー 0 を確認
- [x] 8.3 `go test -race ./...` でテスト全 pass を確認
- [x] 8.4 `go build ./...` で全パッケージビルド成功を確認
- [x] 8.5 手動 E2E: `LOKI_API_HOST=127.0.0.1 LOKI_API_PORT=18080 go run ./cmd/api` で 200 health 応答確認
- [x] 8.6 手動 E2E: `/openapi.json` の `info` に title/version/contact/license/description (557文字) が含まれることを確認
- [x] 8.7 手動 E2E: compress / convert の Responses に `[200,400,413,422,429,500]` が揃い、エラーが `application/problem+json` で配信されることを確認。example も `format=webp` `quality=75` `level=medium` で配信

## 9. コミット・PR

- [x] 9.1 変更内容を確認し、不要ファイル (.serena/project.yml / testdata/) を除外
- [x] 9.2 日本語コミットメッセージで commit (`APIの設定管理とドキュメント整備 (FRO-113)`)
- [x] 9.3 lefthook の pre-commit (fmt/lint) / pre-push (race test) が成功
- [x] 9.4 リモートに push し `gh pr create` で PR #24 を作成 (https://github.com/FrontWorksDev/Loki/pull/24)
