## 1. セットアップ

- [x] 1.1 `feature/fro-112-api-middleware` ブランチを作成する
- [x] 1.2 `go get github.com/go-chi/cors` および `go get golang.org/x/time/rate` で依存を追加し、`go.mod` / `go.sum` を更新する

## 2. 設定スキーマ拡張

- [x] 2.1 `internal/api/config.go` を新規作成し、`api.Config` を `CORS` / `BodyLimitBytes` / `RateLimit` / `Logging` のサブ構造体を含む形に拡張する。`DefaultConfig()` は本変更前と同等の挙動になる安全側のデフォルト値を返すようにする
- [x] 2.2 `internal/api/server.go` の既存 `Config` 定義を `config.go` 側へ移し、`NewServer` のシグネチャ互換を維持する
- [x] 2.3 `configs/default.yaml` に `api:` セクション（cors, body_limit_bytes, rate_limit, logging）を追加する
- [x] 2.4 `cmd/api/main.go` で `viper` を用いた `configs/default.yaml` 読み込みおよび `LOKI_API_*` 環境変数バインドを実装し、読み込んだ値で `api.Config` を構築して `NewServer` に渡す

## 3. ロギングミドルウェア

- [x] 3.1 `internal/api/middleware/logging.go` を新規作成し、`log/slog` のJSONハンドラを使う `NewLogging(logger *slog.Logger) func(http.Handler) http.Handler` を実装する
- [x] 3.2 ステータスコード・レスポンスバイト数を捕捉するための `responseWriter` ラッパーを実装する（ボディは記録しない）
- [x] 3.3 ログフィールド（method, path, status, duration_ms, remote_ip, bytes_out, request_id, user_agent）を出力する
- [x] 3.4 `internal/api/middleware/logging_test.go` を作成し、テーブル駆動でフィールド・ログレベル（5xx は error）・request_id 伝播をテストする

## 4. CORSミドルウェア

- [x] 4.1 `internal/api/middleware/cors.go` を新規作成し、`go-chi/cors` を `api.CORSConfig` から構築するファクトリ `NewCORS(cfg CORSConfig) func(http.Handler) http.Handler` を実装する
- [x] 4.2 `internal/api/middleware/cors_test.go` を作成し、許可オリジン・非許可オリジン・プリフライト・ワイルドカードのケースをテストする

## 5. ボディサイズ制限ミドルウェア

- [x] 5.1 `internal/api/middleware/bodylimit.go` を新規作成し、`http.MaxBytesReader` を被せて413（Hum互換JSONエラー）を返す `NewBodyLimit(maxBytes int64) func(http.Handler) http.Handler` を実装する
- [x] 5.2 `internal/api/middleware/bodylimit_test.go` を作成し、上限以下・上限超過・設定値での上書きをテストする

## 6. レートリミットミドルウェア

- [x] 6.1 `internal/api/middleware/ratelimit.go` を新規作成し、`RateLimiter interface { Allow(key string) bool }` を定義する
- [x] 6.2 `golang.org/x/time/rate` を用いた `inMemoryLimiter`（IPごとの `*rate.Limiter` を `sync.Map` で保持、定期クリーンアップ付き）を実装する
- [x] 6.3 `OPTIONS` メソッドを除外するクライアントIP抽出（`X-Forwarded-For` 最左 → `RemoteAddr`）と429+`Retry-After` レスポンス生成を実装する
- [x] 6.4 `internal/api/middleware/ratelimit_test.go` を作成し、レート以内・超過・異IP独立・X-Forwarded-For抽出・OPTIONS除外・クリーンアップをテストする

## 7. サーバー組み立てとミドルウェア登録

- [x] 7.1 `internal/api/server.go` の `NewServer` で `chi.Use` を用いて `RequestID → Recoverer → Logging → CORS → RateLimit → BodyLimit` の順にミドルウェアを登録する
- [x] 7.2 ヘルスチェックエンドポイント（既存 `GET /api/v1/health`）がレートリミット・ボディサイズ制限の影響を受けないようルーターをサブグループで分離するか、ミドルウェア内で除外する
- [x] 7.3 既存 `internal/api/server_test.go` に統合テスト（CORSヘッダー付与、413、429、ヘルスチェックの非影響）を追加する

## 8. ドキュメント更新

- [x] 8.1 `README.md` に「APIサーバー設定」節を追加し、CORS/レートリミット/ボディ上限/環境変数の説明を記載する

## 9. 検証・コミット

- [x] 9.1 `goimports -w ./...` および `go fmt ./...` を実行する
- [x] 9.2 `golangci-lint run ./...` でlintエラーがないことを確認する
- [x] 9.3 `go test -race -v ./...` で全テストが通ることを確認する
- [x] 9.4 `lefthook install` 済みであることを確認の上、変更をコミットする（コミットメッセージは日本語、Linear FRO-112を含める）
